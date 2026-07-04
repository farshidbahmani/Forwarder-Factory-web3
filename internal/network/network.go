package network

import "fmt"

type Config struct {
	Name       string `json:"name"`
	ChainID    int64  `json:"chainId"`
	Symbol     string `json:"symbol"`
	RPCEnvKey  string `json:"-"`
	DefaultRPC string `json:"-"`
	EnvSuffix  string `json:"envSuffix"`
	IsTestnet  bool   `json:"isTestnet"`
}

var testnets = map[string]bool{
	"bscTestnet":               true,
	"ethereumSepoliaTestnet":   true,
	"polygonAmoyTestnet":       true,
	"arbitrumSepoliaTestnet":   true,
	"optimismSepoliaTestnet":   true,
	"baseSepoliaTestnet":       true,
	"avalancheFujiTestnet":     true,
}

var defs = []Config{
	{Name: "bscMainnet", ChainID: 56, Symbol: "BNB", RPCEnvKey: "BSC_MAINNET_RPC", DefaultRPC: "https://bsc-dataseed1.bnbchain.org", EnvSuffix: "BSC_MAINNET"},
	{Name: "bscTestnet", ChainID: 97, Symbol: "tBNB", RPCEnvKey: "BSC_TESTNET_RPC", DefaultRPC: "https://data-seed-prebsc-1-s1.bnbchain.org:8545", EnvSuffix: "BSC_TESTNET"},
	{Name: "opBNBMainnet", ChainID: 204, Symbol: "BNB", RPCEnvKey: "OPBNB_MAINNET_RPC", DefaultRPC: "https://opbnb-mainnet-rpc.bnbchain.org", EnvSuffix: "OPBNB_MAINNET"},
	{Name: "ethereumMainnet", ChainID: 1, Symbol: "ETH", RPCEnvKey: "ETHEREUM_MAINNET_RPC", DefaultRPC: "https://eth.llamarpc.com", EnvSuffix: "ETHEREUM_MAINNET"},
	{Name: "ethereumSepoliaTestnet", ChainID: 11155111, Symbol: "ETH", RPCEnvKey: "SEPOLIA_TESTNET_RPC", DefaultRPC: "https://rpc.sepolia.org", EnvSuffix: "ETHEREUM_SEPOLIA_TESTNET"},
	{Name: "polygonMainnet", ChainID: 137, Symbol: "MATIC", RPCEnvKey: "POLYGON_MAINNET_RPC", DefaultRPC: "https://polygon-rpc.com", EnvSuffix: "POLYGON_MAINNET"},
	{Name: "polygonAmoyTestnet", ChainID: 80002, Symbol: "MATIC", RPCEnvKey: "POLYGON_AMOY_TESTNET_RPC", DefaultRPC: "https://rpc-amoy.polygon.technology", EnvSuffix: "POLYGON_AMOY_TESTNET"},
	{Name: "arbitrumMainnet", ChainID: 42161, Symbol: "ETH", RPCEnvKey: "ARBITRUM_MAINNET_RPC", DefaultRPC: "https://arb1.arbitrum.io/rpc", EnvSuffix: "ARBITRUM_MAINNET"},
	{Name: "arbitrumSepoliaTestnet", ChainID: 421614, Symbol: "ETH", RPCEnvKey: "ARBITRUM_SEPOLIA_TESTNET_RPC", DefaultRPC: "https://sepolia-rollup.arbitrum.io/rpc", EnvSuffix: "ARBITRUM_SEPOLIA_TESTNET"},
	{Name: "optimismMainnet", ChainID: 10, Symbol: "ETH", RPCEnvKey: "OPTIMISM_MAINNET_RPC", DefaultRPC: "https://mainnet.optimism.io", EnvSuffix: "OPTIMISM_MAINNET"},
	{Name: "optimismSepoliaTestnet", ChainID: 11155420, Symbol: "ETH", RPCEnvKey: "OPTIMISM_SEPOLIA_TESTNET_RPC", DefaultRPC: "https://sepolia.optimism.io", EnvSuffix: "OPTIMISM_SEPOLIA_TESTNET"},
	{Name: "baseMainnet", ChainID: 8453, Symbol: "ETH", RPCEnvKey: "BASE_MAINNET_RPC", DefaultRPC: "https://mainnet.base.org", EnvSuffix: "BASE_MAINNET"},
	{Name: "baseSepoliaTestnet", ChainID: 84532, Symbol: "ETH", RPCEnvKey: "BASE_SEPOLIA_TESTNET_RPC", DefaultRPC: "https://sepolia.base.org", EnvSuffix: "BASE_SEPOLIA_TESTNET"},
	{Name: "avalancheMainnet", ChainID: 43114, Symbol: "AVAX", RPCEnvKey: "AVALANCHE_MAINNET_RPC", DefaultRPC: "https://api.avax.network/ext/bc/C/rpc", EnvSuffix: "AVALANCHE_MAINNET"},
	{Name: "avalancheFujiTestnet", ChainID: 43113, Symbol: "AVAX", RPCEnvKey: "AVALANCHE_FUJI_TESTNET_RPC", DefaultRPC: "https://api.avax-test.network/ext/bc/C/rpc", EnvSuffix: "AVALANCHE_FUJI_TESTNET"},
}

var All []Config

func init() {
	All = make([]Config, len(defs))
	for i, d := range defs {
		d.IsTestnet = testnets[d.Name]
		All[i] = d
	}
}

func Get(name string) (Config, error) {
	for _, n := range All {
		if n.Name == name {
			return n, nil
		}
	}
	return Config{}, fmt.Errorf(`unknown network "%s"`, name)
}

func EnvKey(base string, net Config) string {
	return base + "_" + net.EnvSuffix
}

func RPCURL(net Config, getenv func(string) string) string {
	if v := getenv(net.RPCEnvKey); v != "" {
		return v
	}
	return net.DefaultRPC
}
