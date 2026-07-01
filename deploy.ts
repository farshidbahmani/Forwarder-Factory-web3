import { ethers, run, network } from "hardhat";

async function main() {
  const MOTHER_WALLET = process.env.MOTHER_WALLET;
  const RELAYER      = process.env.RELAYER_ADDRESS;

  if (!MOTHER_WALLET || !RELAYER) {
    throw new Error("MOTHER_WALLET و RELAYER_ADDRESS رو در .env تنظیم کن");
  }

  console.log(`\nDeploy روی شبکه: ${network.name}`);
  console.log(`Mother Wallet : ${MOTHER_WALLET}`);
  console.log(`Relayer       : ${RELAYER}\n`);

  const [deployer] = await ethers.getSigners();
  console.log(`Deployer      : ${deployer.address}`);
  const balance = await ethers.provider.getBalance(deployer.address);
  console.log(`Balance       : ${ethers.formatEther(balance)} BNB\n`);

  // ── Deploy ──────────────────────────────────
  console.log("در حال deploy کردن ForwarderFactory...");
  const Factory = await ethers.getContractFactory("ForwarderFactory");
  const factory = await Factory.deploy(MOTHER_WALLET, RELAYER);
  await factory.waitForDeployment();

  const factoryAddress        = await factory.getAddress();
  const implementationAddress = await factory.implementation();

  console.log(`✅ ForwarderFactory : ${factoryAddress}`);
  console.log(`✅ Implementation   : ${implementationAddress}\n`);

  // ── Verify روی BscScan (فقط برای testnet/mainnet) ──
  if (network.name !== "hardhat" && network.name !== "localhost") {
    console.log("در حال verify کردن روی BscScan (30 ثانیه صبر کن...)");
    await new Promise((r) => setTimeout(r, 30_000)); // صبر تا indexer آماده بشه

    try {
      await run("verify:verify", {
        address: factoryAddress,
        constructorArguments: [MOTHER_WALLET, RELAYER],
      });
      console.log("✅ Factory verified روی BscScan");
    } catch (e: any) {
      if (e.message.includes("Already Verified")) {
        console.log("ℹ️  قبلاً verify شده بود");
      } else {
        console.warn("⚠️  Verify ناموفق:", e.message);
      }
    }
  }

  // ── خلاصه نهایی ──────────────────────────────
  console.log("\n══════════════════════════════════════════");
  console.log("این مقادیر رو در .env پروژه Backend ذخیره کن:");
  console.log(`FACTORY_ADDRESS=${factoryAddress}`);
  console.log("══════════════════════════════════════════\n");
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
