package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"forwarder-factory/internal/apperror"
	"forwarder-factory/internal/env"
	"forwarder-factory/internal/network"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	env *env.Store
}

func NewClient(e *env.Store) *Client {
	return &Client{env: e}
}

func (c *Client) RPC(networkName string) (*ethclient.Client, network.Config, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return nil, net, apperror.BadRequest(err.Error())
	}
	url := network.RPCURL(net, c.env.Get)
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, net, err
	}
	return client, net, nil
}

func (c *Client) Signer(networkName, role string) (*bind.TransactOpts, *ethclient.Client, network.Config, error) {
	client, net, err := c.RPC(networkName)
	if err != nil {
		return nil, nil, net, err
	}

	keyName := "DEPLOYER_PRIVATE_KEY"
	if role == "relayer" {
		keyName = "RELAYER_PRIVATE_KEY"
	}
	pkHex := c.env.GetForNetwork(keyName, net.EnvSuffix)
	if pkHex == "" || pkHex == "0x..." {
		return nil, client, net, apperror.BadRequest(
			fmt.Sprintf("Missing %s (or global %s) in .env", network.EnvKey(keyName, net), keyName),
		)
	}
	pkHex = strings.TrimPrefix(pkHex, "0x")
	key, err := crypto.HexToECDSA(pkHex)
	if err != nil {
		return nil, client, net, apperror.BadRequest("Invalid private key in .env")
	}
	chainID := big.NewInt(net.ChainID)
	opts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, client, net, err
	}
	return opts, client, net, nil
}

type Artifact struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode struct {
		Object string `json:"object"`
	} `json:"bytecode"`
}

func LoadFactoryArtifact() (Artifact, abi.ABI, error) {
	return loadArtifact("ForwarderFactory.sol", "ForwarderFactory.json")
}

func LoadTronFactoryArtifact() (Artifact, abi.ABI, error) {
	return loadArtifact("ForwarderFactoryTron.sol", "ForwarderFactoryTron.json")
}

func LoadForwarderArtifact() (Artifact, abi.ABI, error) {
	return loadArtifact("Forwarder.sol", "Forwarder.json")
}

func loadArtifact(solFile, jsonFile string) (Artifact, abi.ABI, error) {
	path := filepath.Join(mustWd(), "out", solFile, jsonFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return Artifact{}, abi.ABI{}, fmt.Errorf("read factory artifact (run `forge build`): %w", err)
	}
	var art Artifact
	if err := json.Unmarshal(raw, &art); err != nil {
		return Artifact{}, abi.ABI{}, err
	}
	if art.Bytecode.Object == "" {
		return Artifact{}, abi.ABI{}, fmt.Errorf("empty factory bytecode in %s", path)
	}
	parsed, err := abi.JSON(strings.NewReader(string(art.ABI)))
	if err != nil {
		return Artifact{}, abi.ABI{}, err
	}
	return art, parsed, nil
}

func FactoryBytecode(art Artifact) []byte {
	return common.FromHex(art.Bytecode.Object)
}

func (c *Client) FactoryAddress(networkName string) (common.Address, network.Config, error) {
	net, err := network.Get(networkName)
	if err != nil {
		return common.Address{}, net, apperror.BadRequest(err.Error())
	}
	key := network.EnvKey("FACTORY_ADDRESS", net)
	addrStr := c.env.GetForNetwork("FACTORY_ADDRESS", net.EnvSuffix)
	if addrStr == "" {
		return common.Address{}, net, apperror.BadRequest("No factory deployed. Set " + key + " in .env")
	}
	if !common.IsHexAddress(addrStr) {
		return common.Address{}, net, apperror.BadRequest("Invalid " + key + " in .env")
	}
	return common.HexToAddress(addrStr), net, nil
}

func (c *Client) AssertDeployed(ctx context.Context, client *ethclient.Client, net network.Config, addr common.Address) error {
	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return err
	}
	if len(code) == 0 {
		return apperror.BadRequest(fmt.Sprintf(
			"No contract at %s on %s. Check %s — it may be a wallet address, not the deployed factory.",
			addr.Hex(), net.Name, network.EnvKey("FACTORY_ADDRESS", net),
		))
	}
	return nil
}

func mustWd() string {
	wd, _ := os.Getwd()
	if wd == "" {
		return "."
	}
	return wd
}
