import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";
import * as dotenv from "dotenv";
import { NETWORKS } from "./src/domain/network";

dotenv.config();

function getAccountsForNetwork(networkKey: string): string[] {
  const network = NETWORKS.find((n) => n.name === networkKey);
  if (!network) return [];
  const perNetworkKey = process.env[`DEPLOYER_PRIVATE_KEY_${network.envSuffix}`];
  const key = perNetworkKey || process.env.DEPLOYER_PRIVATE_KEY;
  return key ? [key] : [];
}

const networkEntries = Object.fromEntries(
  NETWORKS.map((n) => [
    n.name,
    {
      url: process.env[n.rpcEnvKey] || n.defaultRpc,
      chainId: n.chainId,
      accounts: getAccountsForNetwork(n.name),
    },
  ]),
);

const config: HardhatUserConfig = {
  solidity: {
    version: "0.8.24",
    settings: {
      optimizer: { enabled: true, runs: 200 },
    },
  },

  networks: {
    hardhat: {},
    ...networkEntries,
  },

  etherscan: {
    apiKey: {
      bsc:             process.env.BSCSCAN_API_KEY    || "",
      bscTestnet:      process.env.BSCSCAN_API_KEY    || "",
      mainnet:         process.env.ETHERSCAN_API_KEY  || "",
      sepolia:         process.env.ETHERSCAN_API_KEY  || "",
      polygon:         process.env.POLYGONSCAN_API_KEY || "",
      polygonAmoy:     process.env.POLYGONSCAN_API_KEY || "",
      arbitrumOne:     process.env.ARBISCAN_API_KEY   || "",
      arbitrumSepolia: process.env.ARBISCAN_API_KEY   || "",
      optimisticEthereum: process.env.OPTIMISM_API_KEY || "",
      optimismSepolia:    process.env.OPTIMISM_API_KEY || "",
      base:            process.env.BASESCAN_API_KEY   || "",
      baseSepolia:     process.env.BASESCAN_API_KEY   || "",
      avalanche:       process.env.SNOWTRACE_API_KEY  || "",
      avalancheFuji:   process.env.SNOWTRACE_API_KEY  || "",
    },
  },
};

export default config;
