export type NetworkName =
  | "bscMainnet"
  | "bscTestnet"
  | "opBNBMainnet"
  | "ethereumMainnet"
  | "ethereumSepoliaTestnet"
  | "polygonMainnet"
  | "polygonAmoyTestnet"
  | "arbitrumMainnet"
  | "arbitrumSepoliaTestnet"
  | "optimismMainnet"
  | "optimismSepoliaTestnet"
  | "baseMainnet"
  | "baseSepoliaTestnet"
  | "avalancheMainnet"
  | "avalancheFujiTestnet";

export type NetworkConfig = {
  name: NetworkName;
  chainId: number;
  symbol: string;
  rpcEnvKey: string;
  defaultRpc: string;
  envSuffix: string;
  isTestnet: boolean;
};

const NETWORK_DEFS: Omit<NetworkConfig, "isTestnet">[] = [
  { name: "bscMainnet", chainId: 56, symbol: "BNB", rpcEnvKey: "BSC_MAINNET_RPC", defaultRpc: "https://bsc-dataseed1.bnbchain.org", envSuffix: "BSC_MAINNET" },
  { name: "bscTestnet", chainId: 97, symbol: "tBNB", rpcEnvKey: "BSC_TESTNET_RPC", defaultRpc: "https://data-seed-prebsc-1-s1.bnbchain.org:8545", envSuffix: "BSC_TESTNET" },
  { name: "opBNBMainnet", chainId: 204, symbol: "BNB", rpcEnvKey: "OPBNB_MAINNET_RPC", defaultRpc: "https://opbnb-mainnet-rpc.bnbchain.org", envSuffix: "OPBNB_MAINNET" },
  { name: "ethereumMainnet", chainId: 1, symbol: "ETH", rpcEnvKey: "ETHEREUM_MAINNET_RPC", defaultRpc: "https://eth.llamarpc.com", envSuffix: "ETHEREUM_MAINNET" },
  { name: "ethereumSepoliaTestnet", chainId: 11155111, symbol: "ETH", rpcEnvKey: "SEPOLIA_TESTNET_RPC", defaultRpc: "https://rpc.sepolia.org", envSuffix: "ETHEREUM_SEPOLIA_TESTNET" },
  { name: "polygonMainnet", chainId: 137, symbol: "MATIC", rpcEnvKey: "POLYGON_MAINNET_RPC", defaultRpc: "https://polygon-rpc.com", envSuffix: "POLYGON_MAINNET" },
  { name: "polygonAmoyTestnet", chainId: 80002, symbol: "MATIC", rpcEnvKey: "POLYGON_AMOY_TESTNET_RPC", defaultRpc: "https://rpc-amoy.polygon.technology", envSuffix: "POLYGON_AMOY_TESTNET" },
  { name: "arbitrumMainnet", chainId: 42161, symbol: "ETH", rpcEnvKey: "ARBITRUM_MAINNET_RPC", defaultRpc: "https://arb1.arbitrum.io/rpc", envSuffix: "ARBITRUM_MAINNET" },
  { name: "arbitrumSepoliaTestnet", chainId: 421614, symbol: "ETH", rpcEnvKey: "ARBITRUM_SEPOLIA_TESTNET_RPC", defaultRpc: "https://sepolia-rollup.arbitrum.io/rpc", envSuffix: "ARBITRUM_SEPOLIA_TESTNET" },
  { name: "optimismMainnet", chainId: 10, symbol: "ETH", rpcEnvKey: "OPTIMISM_MAINNET_RPC", defaultRpc: "https://mainnet.optimism.io", envSuffix: "OPTIMISM_MAINNET" },
  { name: "optimismSepoliaTestnet", chainId: 11155420, symbol: "ETH", rpcEnvKey: "OPTIMISM_SEPOLIA_TESTNET_RPC", defaultRpc: "https://sepolia.optimism.io", envSuffix: "OPTIMISM_SEPOLIA_TESTNET" },
  { name: "baseMainnet", chainId: 8453, symbol: "ETH", rpcEnvKey: "BASE_MAINNET_RPC", defaultRpc: "https://mainnet.base.org", envSuffix: "BASE_MAINNET" },
  { name: "baseSepoliaTestnet", chainId: 84532, symbol: "ETH", rpcEnvKey: "BASE_SEPOLIA_TESTNET_RPC", defaultRpc: "https://sepolia.base.org", envSuffix: "BASE_SEPOLIA_TESTNET" },
  { name: "avalancheMainnet", chainId: 43114, symbol: "AVAX", rpcEnvKey: "AVALANCHE_MAINNET_RPC", defaultRpc: "https://api.avax.network/ext/bc/C/rpc", envSuffix: "AVALANCHE_MAINNET" },
  { name: "avalancheFujiTestnet", chainId: 43113, symbol: "AVAX", rpcEnvKey: "AVALANCHE_FUJI_TESTNET_RPC", defaultRpc: "https://api.avax-test.network/ext/bc/C/rpc", envSuffix: "AVALANCHE_FUJI_TESTNET" },
];

const TESTNET_NAMES = new Set<NetworkName>([
  "bscTestnet",
  "ethereumSepoliaTestnet",
  "polygonAmoyTestnet",
  "arbitrumSepoliaTestnet",
  "optimismSepoliaTestnet",
  "baseSepoliaTestnet",
  "avalancheFujiTestnet",
]);

export const NETWORKS: NetworkConfig[] = NETWORK_DEFS.map((n) => ({
  ...n,
  isTestnet: TESTNET_NAMES.has(n.name),
}));

export function getNetwork(name: string): NetworkConfig {
  const net = NETWORKS.find((n) => n.name === name);
  if (!net) {
    throw new Error(`Unknown network "${name}"`);
  }
  return net;
}

export function getRpcUrl(network: NetworkConfig, env: NodeJS.ProcessEnv): string {
  return env[network.rpcEnvKey] || network.defaultRpc;
}

export function envKeyForNetwork(baseKey: string, network: NetworkConfig): string {
  return `${baseKey}_${network.envSuffix}`;
}
