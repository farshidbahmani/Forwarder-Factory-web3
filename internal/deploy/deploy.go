package deploy

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/blockchain"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/network"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Service struct {
	env   *env.Store
	chain *blockchain.Client
}

func New(envStore *env.Store, chain *blockchain.Client) *Service {
	return &Service{env: envStore, chain: chain}
}

type Result struct {
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

func (s *Service) Compile() error {
	forge, err := blockchain.ForgeBin()
	if err != nil {
		return apperror.BadRequest(err.Error())
	}
	cmd := exec.Command(forge, "build")
	cmd.Dir = mustWd()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("forge build failed: %s: %w", string(out), err)
	}
	return nil
}

func (s *Service) Deploy(ctx context.Context, networkName string, verify bool) (*Result, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	mother := s.env.GetForNetwork("MOTHER_WALLET", net.EnvSuffix)
	relayer := s.env.GetForNetwork("RELAYER_ADDRESS", net.EnvSuffix)
	if mother == "" || relayer == "" {
		return nil, apperror.BadRequest(fmt.Sprintf(
			"Set MOTHER_WALLET_%s and RELAYER_ADDRESS_%s in .env", net.EnvSuffix, net.EnvSuffix,
		))
	}

	if err := s.Compile(); err != nil {
		return nil, err
	}

	art, parsedABI, err := blockchain.LoadFactoryArtifact()
	if err != nil {
		return nil, err
	}

	opts, client, _, err := s.chain.Signer(networkName, "deployer")
	if err != nil {
		return nil, err
	}
	defer client.Close()

	balance, err := client.BalanceAt(ctx, opts.From, nil)
	if err != nil {
		return nil, err
	}
	if balance.Sign() == 0 {
		return nil, apperror.BadRequest(fmt.Sprintf(
			"Deployer %s has 0 %s. Fund it before deploying.", opts.From.Hex(), net.Symbol,
		))
	}

	bytecode := blockchain.FactoryBytecode(art)
	addr, tx, contract, err := bind.DeployContract(
		opts, parsedABI, bytecode, client,
		common.HexToAddress(mother), common.HexToAddress(relayer),
	)
	if err != nil {
		return nil, err
	}
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		return nil, fmt.Errorf("factory deployment reverted")
	}

	impl, err := readImplementation(ctx, contract)
	if err != nil {
		return nil, err
	}

	factoryAddr := addr.Hex()
	envKey := network.EnvKey("FACTORY_ADDRESS", net)
	if err := s.env.SetMany(map[string]string{envKey: factoryAddr}); err != nil {
		return nil, err
	}

	verified := false
	var verificationMessage *string
	if verify {
		msg, ok := s.verify(net, factoryAddr, mother, relayer)
		verified = ok
		if msg != "" {
			verificationMessage = &msg
		}
	}

	return &Result{
		Network:               networkName,
		FactoryAddress:        factoryAddr,
		ImplementationAddress: impl.Hex(),
		DeployerAddress:       opts.From.Hex(),
		DeployerBalance:       weiToEther(balance),
		Symbol:                net.Symbol,
		Verified:              verified,
		VerificationMessage:   verificationMessage,
		EnvKey:                envKey,
	}, nil
}

func readImplementation(ctx context.Context, contract *bind.BoundContract) (common.Address, error) {
	var out []interface{}
	if err := contract.Call(&bind.CallOpts{Context: ctx}, &out, "implementation"); err != nil {
		return common.Address{}, err
	}
	if len(out) == 0 {
		return common.Address{}, fmt.Errorf("empty implementation result")
	}
	switch v := out[0].(type) {
	case common.Address:
		return v, nil
	default:
		return common.Address{}, fmt.Errorf("unexpected implementation type")
	}
}

func (s *Service) verify(net network.Config, factoryAddr, mother, relayer string) (string, bool) {
	apiKey := explorerAPIKey(net.EnvSuffix, s.env.Get)
	if apiKey == "" {
		return "missing block explorer API key in .env", false
	}

	cast, err := blockchain.CastBin()
	if err != nil {
		return err.Error(), false
	}
	forge, err := blockchain.ForgeBin()
	if err != nil {
		return err.Error(), false
	}

	args, err := exec.Command(cast, "abi-encode", "constructor(address,address)", mother, relayer).CombinedOutput()
	if err != nil {
		return string(args), false
	}

	cmd := exec.Command(
		forge, "verify-contract",
		factoryAddr,
		"contracts/ForwarderFactory.sol:ForwarderFactory",
		"--chain-id", strconv.FormatInt(net.ChainID, 10),
		"--constructor-args", strings.TrimSpace(string(args)),
		"--etherscan-api-key", apiKey,
		"--rpc-url", network.RPCURL(net, s.env.Get),
	)
	cmd.Dir = mustWd()
	out, err := cmd.CombinedOutput()
	msg := string(out)
	if err != nil {
		if strings.Contains(strings.ToLower(msg), "already verified") {
			return "Already verified", true
		}
		if len(msg) > 500 {
			msg = msg[:500]
		}
		return msg, false
	}
	return "", true
}

func explorerAPIKey(envSuffix string, getenv func(string) string) string {
	switch {
	case strings.HasPrefix(envSuffix, "BSC"), strings.HasPrefix(envSuffix, "OPBNB"):
		return getenv("BSCSCAN_API_KEY")
	case strings.HasPrefix(envSuffix, "ETHEREUM"):
		return getenv("ETHERSCAN_API_KEY")
	case strings.HasPrefix(envSuffix, "POLYGON"):
		return getenv("POLYGONSCAN_API_KEY")
	case strings.HasPrefix(envSuffix, "ARBITRUM"):
		return getenv("ARBISCAN_API_KEY")
	case strings.HasPrefix(envSuffix, "OPTIMISM"):
		return getenv("OPTIMISM_API_KEY")
	case strings.HasPrefix(envSuffix, "BASE"):
		return getenv("BASESCAN_API_KEY")
	case strings.HasPrefix(envSuffix, "AVALANCHE"):
		return getenv("SNOWTRACE_API_KEY")
	default:
		return ""
	}
}

func mustWd() string {
	wd, _ := os.Getwd()
	if wd == "" {
		return "."
	}
	return wd
}

func weiToEther(wei *big.Int) string {
	f := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(1e18))
	v, _ := f.Float64()
	return fmt.Sprintf("%g", v)
}
