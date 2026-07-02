import { ethers, run, network } from "hardhat";

// Native currency symbol per network — used only for display
const NATIVE_SYMBOL: Record<string, string> = {
  bscMainnet:      "BNB",
  bscTestnet:      "tBNB",
  opBNBMainnet:           "BNB",
  ethereumMainnet:        "ETH",
  sepoliaTestnet:         "ETH",
  polygonMainnet:         "MATIC",
  polygonAmoyTestnet:     "MATIC",
  arbitrumMainnet:        "ETH",
  arbitrumSepoliaTestnet: "ETH",
  optimismMainnet:        "ETH",
  optimismSepoliaTestnet: "ETH",
  baseMainnet:            "ETH",
  baseSepoliaTestnet:     "ETH",
  avalancheMainnet:       "AVAX",
  avalancheFujiTestnet:   "AVAX",
  hardhat:         "ETH",
};

async function main() {
  const MOTHER_WALLET = process.env.MOTHER_WALLET;
  const RELAYER       = process.env.RELAYER_ADDRESS;

  if (!MOTHER_WALLET || !RELAYER) {
    throw new Error("Set MOTHER_WALLET and RELAYER_ADDRESS in .env before deploying");
  }

  const symbol = NATIVE_SYMBOL[network.name] ?? "ETH";

  console.log(`\nDeploying to network : ${network.name}`);
  console.log(`Mother Wallet        : ${MOTHER_WALLET}`);
  console.log(`Relayer              : ${RELAYER}\n`);

  const [deployer] = await ethers.getSigners();
  const balance    = await ethers.provider.getBalance(deployer.address);
  console.log(`Deployer : ${deployer.address}`);
  console.log(`Balance  : ${ethers.formatEther(balance)} ${symbol}\n`);

  // ── Deploy ──────────────────────────────────────────────────
  console.log("Deploying ForwarderFactory...");
  const Factory = await ethers.getContractFactory("ForwarderFactory");
  const factory = await Factory.deploy(MOTHER_WALLET, RELAYER);
  await factory.waitForDeployment();

  const factoryAddress        = await factory.getAddress();
  const implementationAddress = await factory.implementation();

  console.log(`✅ ForwarderFactory : ${factoryAddress}`);
  console.log(`✅ Implementation   : ${implementationAddress}\n`);

  // ── Verify on block explorer (skip for local networks) ──────
  const isLocal = network.name === "hardhat" || network.name === "localhost";
  if (!isLocal) {
    console.log("Verifying on block explorer (waiting 30s for indexer)...");
    await new Promise((r) => setTimeout(r, 30_000));

    try {
      await run("verify:verify", {
        address: factoryAddress,
        constructorArguments: [MOTHER_WALLET, RELAYER],
      });
      console.log("✅ Verified on block explorer");
    } catch (e: any) {
      if (e.message.includes("Already Verified")) {
        console.log("ℹ️  Contract was already verified");
      } else {
        console.warn("⚠️  Verification failed:", e.message);
      }
    }
  }

  // ── Summary ─────────────────────────────────────────────────
  const envKey = `FACTORY_ADDRESS_${network.name.toUpperCase()}`;
  console.log("\n══════════════════════════════════════════");
  console.log("Save this in your backend .env:");
  console.log(`${envKey}=${factoryAddress}`);
  console.log("══════════════════════════════════════════\n");
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
