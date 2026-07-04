package wallet

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/blockchain"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/network"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
)

type Service struct {
	env   *env.Store
	chain *blockchain.Client
}

func New(envStore *env.Store, chain *blockchain.Client) *Service {
	return &Service{env: envStore, chain: chain}
}

type GeneratedWallet struct {
	Address    string  `json:"address"`
	PrivateKey string  `json:"privateKey"`
	Mnemonic   *string `json:"mnemonic,omitempty"`
}

type NetworkWallets struct {
	Network  string          `json:"network"`
	Deployer GeneratedWallet `json:"deployer"`
	Relayer  GeneratedWallet `json:"relayer"`
	Mother   GeneratedWallet `json:"mother"`
}

type EnvSnippet struct {
	Network string   `json:"network"`
	Lines   []string `json:"lines"`
}

type Balance struct {
	Network string `json:"network"`
	ChainID int64  `json:"chainId"`
	Symbol  string `json:"symbol"`
	Address string `json:"address"`
	Balance string `json:"balance"`
}

type Status struct {
	Network        string  `json:"network"`
	DeployerKey    bool    `json:"deployerKey"`
	RelayerKey     bool    `json:"relayerKey"`
	RelayerAddress *string `json:"relayerAddress"`
	MotherWallet   *string `json:"motherWallet"`
	FactoryAddress *string `json:"factoryAddress"`
}

func (s *Service) ListNetworks() []network.Config { return network.All }

func (s *Service) GenerateForNetwork(networkName string) (*NetworkWallets, error) {
	if _, err := network.Get(networkName); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	d, err := randomWallet()
	if err != nil {
		return nil, err
	}
	r, err := randomWallet()
	if err != nil {
		return nil, err
	}
	m, err := randomWallet()
	if err != nil {
		return nil, err
	}
	return &NetworkWallets{Network: networkName, Deployer: d, Relayer: r, Mother: m}, nil
}

func (s *Service) ToEnvSnippet(wallets *NetworkWallets) (*EnvSnippet, error) {
	net, err := network.Get(wallets.Network)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	lines := []string{
		"# " + wallets.Network,
		network.EnvKey("DEPLOYER_PRIVATE_KEY", net) + "=" + wallets.Deployer.PrivateKey,
		network.EnvKey("RELAYER_PRIVATE_KEY", net) + "=" + wallets.Relayer.PrivateKey,
		network.EnvKey("RELAYER_ADDRESS", net) + "=" + wallets.Relayer.Address,
		network.EnvKey("MOTHER_WALLET", net) + "=" + wallets.Mother.Address,
	}
	return &EnvSnippet{Network: wallets.Network, Lines: lines}, nil
}

func (s *Service) ToEnvText(wallets *NetworkWallets) (string, error) {
	snippet, err := s.ToEnvSnippet(wallets)
	if err != nil {
		return "", err
	}
	return strings.Join(snippet.Lines, "\n") + "\n", nil
}

func (s *Service) CheckBalance(ctx context.Context, networkName, address string) (*Balance, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if !common.IsHexAddress(address) {
		return nil, apperror.BadRequest("Invalid address")
	}
	client, _, err := s.chain.RPC(networkName)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	bal, err := client.BalanceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return nil, err
	}
	return &Balance{
		Network: networkName,
		ChainID: net.ChainID,
		Symbol:  net.Symbol,
		Address: address,
		Balance: weiToEther(bal),
	}, nil
}

func (s *Service) GetEnvStatus(networkName string) (*Status, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	return &Status{
		Network:        networkName,
		DeployerKey:    s.env.GetForNetwork("DEPLOYER_PRIVATE_KEY", net.EnvSuffix) != "",
		RelayerKey:     s.env.GetForNetwork("RELAYER_PRIVATE_KEY", net.EnvSuffix) != "",
		RelayerAddress: strPtr(s.env.GetForNetwork("RELAYER_ADDRESS", net.EnvSuffix)),
		MotherWallet:   strPtr(s.env.GetForNetwork("MOTHER_WALLET", net.EnvSuffix)),
		FactoryAddress: strPtr(s.env.Get(network.EnvKey("FACTORY_ADDRESS", net))),
	}, nil
}

func randomWallet() (GeneratedWallet, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return GeneratedWallet{}, err
	}
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return fromKey(key, nil), nil
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return fromKey(key, nil), nil
	}
	return fromKey(key, &mnemonic), nil
}

func fromKey(key *ecdsa.PrivateKey, mnemonic *string) GeneratedWallet {
	addr := crypto.PubkeyToAddress(key.PublicKey)
	pk := "0x" + hex.EncodeToString(crypto.FromECDSA(key))
	return GeneratedWallet{Address: addr.Hex(), PrivateKey: pk, Mnemonic: mnemonic}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func weiToEther(wei *big.Int) string {
	f := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(1e18))
	v, _ := f.Float64()
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.18f", v), "0"), ".")
}
