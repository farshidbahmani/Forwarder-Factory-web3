import { ethers } from "ethers";
import * as dotenv from "dotenv";
import * as fs from "fs";
import * as path from "path";

const NETWORKS = [
  { name: "bscMainnet",      chainId: 56,      symbol: "BNB",   url: process.env.BSC_MAINNET_RPC        || "https://bsc-dataseed1.bnbchain.org" },
  { name: "bscTestnet",      chainId: 97,      symbol: "tBNB",  url: process.env.BSC_TESTNET_RPC        || "https://data-seed-prebsc-1-s1.bnbchain.org:8545" },
  { name: "opBNB",           chainId: 204,     symbol: "BNB",   url: process.env.OPBNB_RPC              || "https://opbnb-mainnet-rpc.bnbchain.org" },
  { name: "ethereum",        chainId: 1,       symbol: "ETH",   url: process.env.ETHEREUM_RPC           || "https://eth.llamarpc.com" },
  { name: "sepolia",         chainId: 11155111, symbol: "ETH",  url: process.env.SEPOLIA_RPC            || "https://rpc.sepolia.org" },
  { name: "polygon",         chainId: 137,     symbol: "MATIC", url: process.env.POLYGON_RPC            || "https://polygon-rpc.com" },
  { name: "polygonAmoy",     chainId: 80002,   symbol: "MATIC", url: process.env.POLYGON_AMOY_RPC       || "https://rpc-amoy.polygon.technology" },
  { name: "arbitrum",        chainId: 42161,   symbol: "ETH",   url: process.env.ARBITRUM_RPC           || "https://arb1.arbitrum.io/rpc" },
  { name: "arbitrumSepolia", chainId: 421614,  symbol: "ETH",   url: process.env.ARBITRUM_SEPOLIA_RPC   || "https://sepolia-rollup.arbitrum.io/rpc" },
  { name: "optimism",        chainId: 10,      symbol: "ETH",   url: process.env.OPTIMISM_RPC           || "https://mainnet.optimism.io" },
  { name: "optimismSepolia", chainId: 11155420, symbol: "ETH",  url: process.env.OPTIMISM_SEPOLIA_RPC   || "https://sepolia.optimism.io" },
  { name: "base",            chainId: 8453,    symbol: "ETH",   url: process.env.BASE_RPC               || "https://mainnet.base.org" },
  { name: "baseSepolia",     chainId: 84532,   symbol: "ETH",   url: process.env.BASE_SEPOLIA_RPC       || "https://sepolia.base.org" },
  { name: "avalanche",       chainId: 43114,   symbol: "AVAX",  url: process.env.AVALANCHE_RPC          || "https://api.avax.network/ext/bc/C/rpc" },
  { name: "avalancheFuji",   chainId: 43113,   symbol: "AVAX",  url: process.env.AVALANCHE_FUJI_RPC     || "https://api.avax-test.network/ext/bc/C/rpc" },
] as const;

type NetworkName = (typeof NETWORKS)[number]["name"];

type NetworkWallets = {
  deployer: ethers.HDNodeWallet;
  relayer: ethers.HDNodeWallet;
  mother: ethers.HDNodeWallet;
};

function toEnvSuffix(networkName: string): string {
  return networkName
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1_$2")
    .replace(/([a-z])([A-Z])/g, "$1_$2")
    .toUpperCase();
}

function getNetwork(name: string) {
  const net = NETWORKS.find((n) => n.name === name);
  if (!net) {
    const available = NETWORKS.map((n) => n.name).join(", ");
    throw new Error(`Unknown network "${name}". Available: ${available}`);
  }
  return net;
}

function parseArgs() {
  const args = process.argv.slice(2);
  const networkArg = args.find((a) => a.startsWith("--network="))?.split("=")[1];
  return {
    network: networkArg as NetworkName | undefined,
    allNetworks: args.includes("--all-networks"),
    check: args.includes("--check"),
    save: args.includes("--save"),
    list: args.includes("--list"),
  };
}

function createNetworkWallets(): NetworkWallets {
  return {
    deployer: ethers.Wallet.createRandom(),
    relayer: ethers.Wallet.createRandom(),
    mother: ethers.Wallet.createRandom(),
  };
}

function envLinesForNetwork(networkName: string, wallets: NetworkWallets): string[] {
  const suffix = toEnvSuffix(networkName);
  return [
    `# ${networkName}`,
    `DEPLOYER_PRIVATE_KEY_${suffix}=${wallets.deployer.privateKey}`,
    `RELAYER_ADDRESS_${suffix}=${wallets.relayer.address}`,
    `MOTHER_WALLET_${suffix}=${wallets.mother.address}`,
    "",
  ];
}

function printNetworkWallets(networkName: string, wallets: NetworkWallets) {
  const net = getNetwork(networkName);
  console.log(`\n══════════════════════════════════════════`);
  console.log(`  ${net.name} (chainId ${net.chainId}, ${net.symbol})`);
  console.log(`══════════════════════════════════════════`);

  const roles: [string, ethers.HDNodeWallet][] = [
    ["Deployer", wallets.deployer],
    ["Relayer", wallets.relayer],
    ["Mother Wallet", wallets.mother],
  ];

  for (const [label, wallet] of roles) {
    console.log(`\n── ${label}`);
    console.log(`Address     : ${wallet.address}`);
    console.log(`Private Key : ${wallet.privateKey}`);
    if (wallet.mnemonic?.phrase) {
      console.log(`Mnemonic    : ${wallet.mnemonic.phrase}`);
    }
  }
}

async function checkNetworkBalance(networkName: string) {
  dotenv.config();
  const net = getNetwork(networkName);
  const suffix = toEnvSuffix(networkName);
  const key = process.env[`DEPLOYER_PRIVATE_KEY_${suffix}`];

  if (!key || key === "0x...") {
    throw new Error(`Set DEPLOYER_PRIVATE_KEY_${suffix} in .env to check ${networkName}`);
  }

  const wallet = new ethers.Wallet(key);
  const provider = new ethers.JsonRpcProvider(net.url);
  const balance = await provider.getBalance(wallet.address);

  console.log(
    `${net.name.padEnd(18)} ${wallet.address}  ${ethers.formatEther(balance).padStart(14)} ${net.symbol}`
  );
}

function printUsage() {
  console.log("\nUsage:");
  console.log("  npm run wallet:create -- --network=bscTestnet   # wallets for one network");
  console.log("  npm run wallet:all-networks                   # wallets for all 15 networks");
  console.log("  npm run wallet:all-networks -- --save         # also append to wallets.env");
  console.log("  npm run wallet:check -- --network=bscTestnet  # check balance on one network");
  console.log("  npm run wallet:check -- --all-networks        # check all saved network wallets");
  console.log("  npm run wallet:list                           # list supported networks\n");
}

async function main() {
  const { network, allNetworks, check, save, list } = parseArgs();

  if (list || (!network && !allNetworks && !check)) {
    console.log("\nSupported networks:");
    for (const net of NETWORKS) {
      console.log(`  • ${net.name} (chainId ${net.chainId}, ${net.symbol})`);
    }
    printUsage();
    if (!list && !network && !allNetworks && !check) return;
    if (list) return;
  }

  if (check) {
    console.log("\n══════════════════════════════════════════");
    console.log("  Balance Check");
    console.log("══════════════════════════════════════════\n");

    const targets = allNetworks ? NETWORKS.map((n) => n.name) : network ? [network] : [];
    if (targets.length === 0) {
      throw new Error("Use --network=<name> or --all-networks with --check");
    }

    for (const name of targets) {
      try {
        await checkNetworkBalance(name);
      } catch (e: any) {
        console.log(`${name.padEnd(18)} skipped — ${e.message}`);
      }
    }
    console.log("");
    return;
  }

  const targets = allNetworks ? NETWORKS.map((n) => n.name) : network ? [network] : [];
  if (targets.length === 0) {
    throw new Error("Use --network=<name> or --all-networks");
  }

  console.log("\n══════════════════════════════════════════");
  console.log("  Per-Network Wallet Generator");
  console.log("══════════════════════════════════════════");
  console.log("\nEach network gets its own deployer, relayer, and mother wallet.");

  const envLines: string[] = [
    "# Auto-generated per-network wallets — store securely, never commit to git",
    `# Generated at ${new Date().toISOString()}`,
    "",
  ];

  for (const name of targets) {
    const wallets = createNetworkWallets();
    printNetworkWallets(name, wallets);
    envLines.push(...envLinesForNetwork(name, wallets));
  }

  console.log("\n══════════════════════════════════════════");
  console.log("  .env snippet");
  console.log("══════════════════════════════════════════\n");
  console.log(envLines.join("\n"));
  console.log("══════════════════════════════════════════");
  console.log("\n⚠️  Store private keys securely. Never commit them to git.");

  if (save) {
    const outPath = path.join(__dirname, "..", "wallets.env");
    fs.writeFileSync(outPath, envLines.join("\n"), { flag: "a" });
    console.log(`\n✅ Appended to ${outPath}`);
    console.log("   Add wallets.env to .gitignore if not already there.\n");
  } else {
    console.log("\nTip: add --save to append output to wallets.env\n");
  }
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
