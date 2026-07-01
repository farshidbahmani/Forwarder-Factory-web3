import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";
import * as dotenv from "dotenv";
dotenv.config();

const DEPLOYER_KEY = process.env.DEPLOYER_PRIVATE_KEY;
const accounts     = DEPLOYER_KEY ? [DEPLOYER_KEY] : [];

const config: HardhatUserConfig = {
  solidity: {
    version: "0.8.24",
    settings: {
      optimizer: { enabled: true, runs: 200 },
    },
  },

  networks: {
    // ── Local ───────────────────────────────────────────────
    hardhat: {},

    // ── BNB Chain ───────────────────────────────────────────
    bscMainnet: {
      url: process.env.BSC_MAINNET_RPC || "https://bsc-dataseed1.bnbchain.org",
      chainId: 56,
      accounts,
    },
    bscTestnet: {
      url: process.env.BSC_TESTNET_RPC || "https://data-seed-prebsc-1-s1.bnbchain.org:8545",
      chainId: 97,
      accounts,
    },
    opBNB: {
      url: process.env.OPBNB_RPC || "https://opbnb-mainnet-rpc.bnbchain.org",
      chainId: 204,
      accounts,
    },

    // ── Ethereum ────────────────────────────────────────────
    ethereum: {
      url: process.env.ETHEREUM_RPC || "https://eth.llamarpc.com",
      chainId: 1,
      accounts,
    },
    sepolia: {
      url: process.env.SEPOLIA_RPC || "https://rpc.sepolia.org",
      chainId: 11155111,
      accounts,
    },

    // ── Polygon ─────────────────────────────────────────────
    polygon: {
      url: process.env.POLYGON_RPC || "https://polygon-rpc.com",
      chainId: 137,
      accounts,
    },
    polygonAmoy: {
      url: process.env.POLYGON_AMOY_RPC || "https://rpc-amoy.polygon.technology",
      chainId: 80002,
      accounts,
    },

    // ── Arbitrum ────────────────────────────────────────────
    arbitrum: {
      url: process.env.ARBITRUM_RPC || "https://arb1.arbitrum.io/rpc",
      chainId: 42161,
      accounts,
    },
    arbitrumSepolia: {
      url: process.env.ARBITRUM_SEPOLIA_RPC || "https://sepolia-rollup.arbitrum.io/rpc",
      chainId: 421614,
      accounts,
    },

    // ── Optimism ────────────────────────────────────────────
    optimism: {
      url: process.env.OPTIMISM_RPC || "https://mainnet.optimism.io",
      chainId: 10,
      accounts,
    },
    optimismSepolia: {
      url: process.env.OPTIMISM_SEPOLIA_RPC || "https://sepolia.optimism.io",
      chainId: 11155420,
      accounts,
    },

    // ── Base ────────────────────────────────────────────────
    base: {
      url: process.env.BASE_RPC || "https://mainnet.base.org",
      chainId: 8453,
      accounts,
    },
    baseSepolia: {
      url: process.env.BASE_SEPOLIA_RPC || "https://sepolia.base.org",
      chainId: 84532,
      accounts,
    },

    // ── Avalanche ───────────────────────────────────────────
    avalanche: {
      url: process.env.AVALANCHE_RPC || "https://api.avax.network/ext/bc/C/rpc",
      chainId: 43114,
      accounts,
    },
    avalancheFuji: {
      url: process.env.AVALANCHE_FUJI_RPC || "https://api.avax-test.network/ext/bc/C/rpc",
      chainId: 43113,
      accounts,
    },
  },

  // ── Block explorer API keys for contract verification ────
  etherscan: {
    apiKey: {
      // BNB Chain
      bsc:             process.env.BSCSCAN_API_KEY    || "",
      bscTestnet:      process.env.BSCSCAN_API_KEY    || "",
      // Ethereum
      mainnet:         process.env.ETHERSCAN_API_KEY  || "",
      sepolia:         process.env.ETHERSCAN_API_KEY  || "",
      // Polygon
      polygon:         process.env.POLYGONSCAN_API_KEY || "",
      polygonAmoy:     process.env.POLYGONSCAN_API_KEY || "",
      // Arbitrum
      arbitrumOne:     process.env.ARBISCAN_API_KEY   || "",
      arbitrumSepolia: process.env.ARBISCAN_API_KEY   || "",
      // Optimism
      optimisticEthereum: process.env.OPTIMISM_API_KEY || "",
      optimismSepolia:    process.env.OPTIMISM_API_KEY || "",
      // Base
      base:            process.env.BASESCAN_API_KEY   || "",
      baseSepolia:     process.env.BASESCAN_API_KEY   || "",
      // Avalanche
      avalanche:       process.env.SNOWTRACE_API_KEY  || "",
      avalancheFuji:   process.env.SNOWTRACE_API_KEY  || "",
    },
  },
};

export default config;
