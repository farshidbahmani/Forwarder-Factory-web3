package tron

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/blockchain"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/network"

	tronclient "github.com/fbsobreira/gotron-sdk/pkg/client"
	tronaddr "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

const (
	deployFeeLimit       = 1_000_000_000 // 1000 TRX max
	deployOriginEnergy   = 10_000_000
	deployConsumePercent = 100
)

type DeployService struct {
	env  *env.Store
	tron *Client
}

func NewDeployService(envStore *env.Store, tronClient *Client) *DeployService {
	return &DeployService{env: envStore, tron: tronClient}
}

type DeployResult struct {
	Network               string  `json:"network"`
	FactoryAddress        string  `json:"factoryAddress"`
	ImplementationAddress string  `json:"implementationAddress"`
	DeployerAddress       string  `json:"deployerAddress"`
	DeployerBalance       string  `json:"deployerBalance"`
	Symbol                string  `json:"symbol"`
	Verified              bool    `json:"verified"`
	VerificationMessage   *string `json:"verificationMessage,omitempty"`
	EnvKey                string  `json:"envKey"`
}

func (s *DeployService) Deploy(ctx context.Context, networkName string, verify bool) (*DeployResult, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if !network.IsTron(net) {
		return nil, apperror.BadRequest(networkName + " is not a Tron network")
	}

	motherRaw := s.env.GetForNetwork("MOTHER_WALLET", net.EnvSuffix)
	relayerRaw := s.env.GetForNetwork("RELAYER_ADDRESS", net.EnvSuffix)
	if motherRaw == "" || relayerRaw == "" {
		return nil, apperror.BadRequest(fmt.Sprintf(
			"Set %s and %s in .env",
			network.EnvKey("MOTHER_WALLET", net), network.EnvKey("RELAYER_ADDRESS", net),
		))
	}
	mother, err := NormalizeAddress(motherRaw)
	if err != nil {
		return nil, err
	}
	relayer, err := NormalizeAddress(relayerRaw)
	if err != nil {
		return nil, err
	}

	art, parsedABI, err := blockchain.LoadTronFactoryArtifact()
	if err != nil {
		return nil, err
	}

	key, from, _, err := s.tron.PrivateKey(networkName, "deployer")
	if err != nil {
		return nil, err
	}

	grpc, _, err := s.tron.GRPC(networkName)
	if err != nil {
		return nil, err
	}

	balance, err := s.accountBalance(ctx, grpc, from)
	if err != nil {
		return nil, err
	}
	if balance.Sign() == 0 {
		return nil, apperror.BadRequest(fmt.Sprintf(
			"Deployer %s has 0 %s. Fund it before deploying.", from, net.Symbol,
		))
	}

	bytecode := blockchain.FactoryBytecode(art)
	motherETH, err := ToETHAddress(mother)
	if err != nil {
		return nil, err
	}
	relayerETH, err := ToETHAddress(relayer)
	if err != nil {
		return nil, err
	}
	constructorArgs, err := parsedABI.Constructor.Inputs.Pack(motherETH, relayerETH)
	if err != nil {
		return nil, fmt.Errorf("pack constructor args: %w", err)
	}
	deployCode := append(append([]byte{}, bytecode...), constructorArgs...)

	factoryAddr, err := s.broadcastDeploy(ctx, grpc, key, from, "ForwarderFactoryTron", deployCode)
	if err != nil {
		return nil, err
	}

	impl, err := s.ensureImplementation(ctx, networkName, grpc, key, from, factoryAddr)
	if err != nil {
		return nil, err
	}

	return s.buildResult(networkName, net, factoryAddr, impl, from, balance, verify)
}

// CompleteSetup deploys Forwarder + setImplementation for the factory already in .env.
func (s *DeployService) CompleteSetup(ctx context.Context, networkName string, verify bool) (*DeployResult, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if !network.IsTron(net) {
		return nil, apperror.BadRequest(networkName + " is not a Tron network")
	}

	factoryAddr, _, err := s.tron.FactoryAddress(networkName)
	if err != nil {
		return nil, err
	}

	key, from, _, err := s.tron.PrivateKey(networkName, "deployer")
	if err != nil {
		return nil, err
	}

	grpc, _, err := s.tron.GRPC(networkName)
	if err != nil {
		return nil, err
	}

	balance, err := s.accountBalance(ctx, grpc, from)
	if err != nil {
		return nil, err
	}

	impl, err := s.ensureImplementation(ctx, networkName, grpc, key, from, factoryAddr)
	if err != nil {
		return nil, err
	}

	return s.buildResult(networkName, net, factoryAddr, impl, from, balance, verify)
}

func (s *DeployService) buildResult(networkName string, net network.Config, factoryAddr, impl, from string, balance *big.Int, verify bool) (*DeployResult, error) {
	envKey := network.EnvKey("FACTORY_ADDRESS", net)

	verified := false
	var verificationMessage *string
	if verify {
		msg := "Tron contract verification: publish source on https://tronscan.org (manual or TronScan API)"
		verificationMessage = &msg
	}

	return &DeployResult{
		Network:               networkName,
		FactoryAddress:        factoryAddr,
		ImplementationAddress: impl,
		DeployerAddress:       from,
		DeployerBalance:       sunToTRX(balance),
		Symbol:                net.Symbol,
		Verified:              verified,
		VerificationMessage:   verificationMessage,
		EnvKey:                envKey,
	}, nil
}

func (s *DeployService) ensureImplementation(ctx context.Context, networkName string, grpc *tronclient.GrpcClient, key *ecdsa.PrivateKey, from, factoryAddr string) (string, error) {
	impl, err := s.readImplementationETH(ctx, grpc, factoryAddr)
	if err == nil && impl != (common.Address{}) {
		return FormatETHAddress(impl), nil
	}

	forwarderArt, forwarderABI, err := blockchain.LoadForwarderArtifact()
	if err != nil {
		return "", err
	}

	factoryETH, err := ToETHAddress(factoryAddr)
	if err != nil {
		return "", err
	}
	forwarderArgs, err := forwarderABI.Constructor.Inputs.Pack(factoryETH)
	if err != nil {
		return "", fmt.Errorf("pack forwarder constructor: %w", err)
	}
	forwarderCode := append(append([]byte{}, blockchain.FactoryBytecode(forwarderArt)...), forwarderArgs...)

	implAddr, err := s.broadcastDeploy(ctx, grpc, key, from, "Forwarder", forwarderCode)
	if err != nil {
		return "", fmt.Errorf("deploy forwarder: %w", err)
	}

	if err := s.callSetImplementation(ctx, grpc, key, from, factoryAddr, implAddr); err != nil {
		return "", fmt.Errorf("setImplementation: %w", err)
	}

	implETH, err := s.readImplementationETH(ctx, grpc, factoryAddr)
	if err != nil {
		return "", err
	}
	if implETH == (common.Address{}) {
		return "", fmt.Errorf("setImplementation succeeded but implementation is still zero")
	}
	return FormatETHAddress(implETH), nil
}

func (s *DeployService) callSetImplementation(ctx context.Context, grpc *tronclient.GrpcClient, key *ecdsa.PrivateKey, from, factoryAddr, implAddr string) error {
	_, parsed, err := blockchain.LoadTronFactoryArtifact()
	if err != nil {
		return err
	}
	implETH, err := ToETHAddress(implAddr)
	if err != nil {
		return err
	}
	cs := NewContractService(s.tron, parsed)
	_, _, _, err = cs.transact(ctx, grpc, key, from, factoryAddr, "setImplementation", implETH)
	return err
}

func (s *DeployService) broadcastDeploy(ctx context.Context, grpc *tronclient.GrpcClient, key *ecdsa.PrivateKey, from, contractName string, deployCode []byte) (string, error) {
	txExt, err := grpc.DeployContractCtx(ctx, from, contractName, &core.SmartContract_ABI{},
		"0x"+hex.EncodeToString(deployCode), deployFeeLimit, deployConsumePercent, deployOriginEnergy)
	if err != nil {
		return "", err
	}
	if txExt.Result != nil && txExt.Result.Code > 0 {
		return "", fmt.Errorf("%s", string(txExt.Result.Message))
	}

	signed, err := signTransaction(key, txExt)
	if err != nil {
		return "", err
	}
	if _, err := grpc.BroadcastCtx(ctx, signed); err != nil {
		return "", err
	}

	info, err := waitDeployMined(ctx, grpc, txExt.Txid)
	if err != nil {
		return "", err
	}
	if len(info.ContractAddress) == 0 {
		return "", fmt.Errorf("deployment succeeded but no contract address in receipt")
	}
	return tronaddr.Address(info.ContractAddress).String(), nil
}

func (s *DeployService) readImplementationETH(ctx context.Context, grpc *tronclient.GrpcClient, factoryAddr string) (common.Address, error) {
	_, parsed, err := blockchain.LoadTronFactoryArtifact()
	if err != nil {
		return common.Address{}, err
	}
	data, err := parsed.Pack("implementation")
	if err != nil {
		return common.Address{}, err
	}
	tx, err := grpc.TriggerConstantContractWithDataCtx(ctx, "", factoryAddr, data)
	if err != nil {
		return common.Address{}, err
	}
	if tx.Result != nil && tx.Result.Code > 0 {
		return common.Address{}, fmt.Errorf("%s", string(tx.Result.Message))
	}
	if len(tx.ConstantResult) == 0 {
		return common.Address{}, fmt.Errorf("empty implementation result")
	}
	if reason, err := abi.UnpackRevert(tx.ConstantResult[0]); err == nil && reason != "" {
		return common.Address{}, fmt.Errorf("%s", reason)
	}
	vals, err := parsed.Unpack("implementation", tx.ConstantResult[0])
	if err != nil {
		return common.Address{}, err
	}
	addr, ok := vals[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("unexpected implementation type")
	}
	return addr, nil
}

func (s *DeployService) accountBalance(ctx context.Context, grpc *tronclient.GrpcClient, addr string) (*big.Int, error) {
	acc, err := grpc.GetAccountCtx(ctx, addr)
	if err != nil {
		return nil, err
	}
	return big.NewInt(acc.Balance), nil
}

func sunToTRX(sun *big.Int) string {
	f := new(big.Float).Quo(new(big.Float).SetInt(sun), big.NewFloat(1e6))
	v, _ := f.Float64()
	return fmt.Sprintf("%g", v)
}
