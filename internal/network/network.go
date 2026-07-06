package network

import "fmt"

type ChainType string

const (
	ChainEVM  ChainType = "evm"
	ChainTron ChainType = "tron"
)

type Config struct {
	Name       string    `json:"name"`
	ChainID    int64     `json:"chainId"`
	ChainType  ChainType `json:"chainType"`
	Symbol     string    `json:"symbol"`
	RPCEnvKey  string    `json:"-"`
	DefaultRPC string    `json:"-"`
	EnvSuffix  string    `json:"envSuffix"`
	IsTestnet  bool      `json:"isTestnet"`
}

var testnets = map[string]bool{
	"bnbTestnet":               true,
	"ethereumSepoliaTestnet":   true,
	"polygonAmoyTestnet":       true,
	"arbitrumSepoliaTestnet":   true,
	"optimismSepoliaTestnet":   true,
	"baseSepoliaTestnet":       true,
	"avalancheFujiTestnet":     true,
	"tronShasta":               true,
}

var defs = []Config{
	{Name: "bnbMainnet", ChainID: 56, Symbol: "BNB", RPCEnvKey: "BNB_MAINNET_RPC", DefaultRPC: "https://bsc-dataseed1.bnbchain.org", EnvSuffix: "BNB_MAINNET"},
	{Name: "bnbTestnet", ChainID: 97, Symbol: "tBNB", RPCEnvKey: "BNB_TESTNET_RPC", DefaultRPC: "https://data-seed-prebsc-1-s1.bnbchain.org:8545", EnvSuffix: "BNB_TESTNET"},
	{Name: "opBNBMainnet", ChainID: 204, Symbol: "BNB", RPCEnvKey: "OPBNB_MAINNET_RPC", DefaultRPC: "https://opbnb-mainnet-rpc.bnbchain.org", EnvSuffix: "OPBNB_MAINNET"},
	{Name: "ethereumMainnet", ChainID: 1, Symbol: "ETH", RPCEnvKey: "ETHEREUM_MAINNET_RPC", DefaultRPC: "https://eth.llamarpc.com", EnvSuffix: "ETHEREUM_MAINNET"},
	{Name: "ethereumSepoliaTestnet", ChainID: 11155111, Symbol: "ETH", RPCEnvKey: "SEPOLIA_TESTNET_RPC", DefaultRPC: "https://rpc.sepolia.org", EnvSuffix: "ETHEREUM_SEPOLIA_TESTNET"},
	{Name: "polygonMainnet", ChainID: 137, Symbol: "POL", RPCEnvKey: "POLYGON_MAINNET_RPC", DefaultRPC: "https://polygon-rpc.com", EnvSuffix: "POLYGON_MAINNET"},
	{Name: "polygonAmoyTestnet", ChainID: 80002, Symbol: "POL", RPCEnvKey: "POLYGON_AMOY_TESTNET_RPC", DefaultRPC: "https://rpc-amoy.polygon.technology", EnvSuffix: "POLYGON_AMOY_TESTNET"},
	{Name: "arbitrumMainnet", ChainID: 42161, Symbol: "ETH", RPCEnvKey: "ARBITRUM_MAINNET_RPC", DefaultRPC: "https://arb1.arbitrum.io/rpc", EnvSuffix: "ARBITRUM_MAINNET"},
	{Name: "arbitrumSepoliaTestnet", ChainID: 421614, Symbol: "ETH", RPCEnvKey: "ARBITRUM_SEPOLIA_TESTNET_RPC", DefaultRPC: "https://sepolia-rollup.arbitrum.io/rpc", EnvSuffix: "ARBITRUM_SEPOLIA_TESTNET"},
	{Name: "optimismMainnet", ChainID: 10, Symbol: "ETH", RPCEnvKey: "OPTIMISM_MAINNET_RPC", DefaultRPC: "https://mainnet.optimism.io", EnvSuffix: "OPTIMISM_MAINNET"},
	{Name: "optimismSepoliaTestnet", ChainID: 11155420, Symbol: "ETH", RPCEnvKey: "OPTIMISM_SEPOLIA_TESTNET_RPC", DefaultRPC: "https://sepolia.optimism.io", EnvSuffix: "OPTIMISM_SEPOLIA_TESTNET"},
	{Name: "baseMainnet", ChainID: 8453, Symbol: "ETH", RPCEnvKey: "BASE_MAINNET_RPC", DefaultRPC: "https://mainnet.base.org", EnvSuffix: "BASE_MAINNET"},
	{Name: "baseSepoliaTestnet", ChainID: 84532, Symbol: "ETH", RPCEnvKey: "BASE_SEPOLIA_TESTNET_RPC", DefaultRPC: "https://sepolia.base.org", EnvSuffix: "BASE_SEPOLIA_TESTNET"},
	{Name: "avalancheMainnet", ChainID: 43114, Symbol: "AVAX", RPCEnvKey: "AVALANCHE_MAINNET_RPC", DefaultRPC: "https://api.avax.network/ext/bc/C/rpc", EnvSuffix: "AVALANCHE_MAINNET"},
	{Name: "avalancheFujiTestnet", ChainID: 43113, Symbol: "AVAX", RPCEnvKey: "AVALANCHE_FUJI_TESTNET_RPC", DefaultRPC: "https://api.avax-test.network/ext/bc/C/rpc", EnvSuffix: "AVALANCHE_FUJI_TESTNET"},
	{Name: "tronMainnet", ChainID: 728126428, ChainType: ChainTron, Symbol: "TRX", RPCEnvKey: "TRON_MAINNET_RPC", DefaultRPC: "grpc.trongrid.io:50051", EnvSuffix: "TRON_MAINNET"},
	{Name: "tronShasta", ChainID: 2494104990, ChainType: ChainTron, Symbol: "TRX", RPCEnvKey: "TRON_SHASTA_RPC", DefaultRPC: "grpc.shasta.trongrid.io:50051", EnvSuffix: "TRON_SHASTA"},
}

var All []Config

func init() {
	All = make([]Config, len(defs))
	for i, d := range defs {
		if d.ChainType == "" {
			d.ChainType = ChainEVM
		}
		d.IsTestnet = testnets[d.Name]
		All[i] = d
	}
}

func IsTron(net Config) bool { return net.ChainType == ChainTron }

func Get(name string) (Config, error) {
	for _, n := range All {
		if n.Name == name {
			return n, nil
		}
	}
	return Config{}, fmt.Errorf(`unknown network "%s"`, name)
}

// Names returns supported network keys in definition order.
func Names() []string {
	names := make([]string, len(All))
	for i, n := range All {
		names[i] = n.Name
	}
	return names
}

func EnvKey(base string, net Config) string {
	return net.EnvSuffix + "_" + base
}

func RPCURL(net Config, getenv func(string) string) string {
	if v := getenv(net.RPCEnvKey); v != "" {
		return v
	}
	if net.RPCEnvKey == "BNB_TESTNET_RPC" {
		if v := getenv("BSC_TESTNET_RPC"); v != "" {
			return v
		}
	}
	return net.DefaultRPC
}
