import { expect } from "chai";
import { ethers } from "hardhat";
import { ForwarderFactory, MockBEP20 } from "../typechain-types";
import { HardhatEthersSigner } from "@nomicfoundation/hardhat-ethers/signers";

describe("ForwarderFactory", () => {
  let factory:     ForwarderFactory;
  let token:       MockBEP20;
  let owner:       HardhatEthersSigner;
  let relayer:     HardhatEthersSigner;
  let motherWallet:HardhatEthersSigner;
  let attacker:    HardhatEthersSigner;
  let user:        HardhatEthersSigner;

  const USER_ID = 1n;

  beforeEach(async () => {
    [owner, relayer, motherWallet, attacker, user] = await ethers.getSigners();

    factory = await ethers.deployContract("ForwarderFactory", [
      motherWallet.address,
      relayer.address,
    ]);

    token = await ethers.deployContract("MockBEP20", ["Mock USDT", "mUSDT"]);
  });

  // ────────────────────────────────────────────
  describe("getAddress", () => {
    it("آدرس کاربر قبل از deploy باید قابل پیش‌بینی باشد", async () => {
      const predicted = await factory.getAddress(USER_ID);
      expect(predicted).to.be.properAddress;
      expect(await ethers.provider.getCode(predicted)).to.equal("0x");
    });

    it("دو userId متفاوت باید آدرس متفاوت بدن", async () => {
      const addr1 = await factory.getAddress(1n);
      const addr2 = await factory.getAddress(2n);
      expect(addr1).to.not.equal(addr2);
    });
  });

  // ────────────────────────────────────────────
  describe("deployAndSweepBNB", () => {
    it("باید BNB را به مادر والت sweep کند", async () => {
      const userWallet = await factory.getAddress(USER_ID);
      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("1") });

      const before = await ethers.provider.getBalance(motherWallet.address);
      await factory.connect(relayer).deployAndSweepBNB(USER_ID);
      const after  = await ethers.provider.getBalance(motherWallet.address);

      expect(after - before).to.equal(ethers.parseEther("1"));
    });

    it("دومین sweep باید بدون deploy مجدد کار کند", async () => {
      const userWallet = await factory.getAddress(USER_ID);

      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("1") });
      await factory.connect(relayer).deployAndSweepBNB(USER_ID);

      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("0.5") });
      await expect(
        factory.connect(relayer).deployAndSweepBNB(USER_ID)
      ).to.not.emit(factory, "WalletDeployed");
    });

    it("غیر relayer نباید بتواند sweep کند", async () => {
      await expect(
        factory.connect(attacker).deployAndSweepBNB(USER_ID)
      ).to.be.revertedWith("Factory: not relayer");
    });
  });

  // ────────────────────────────────────────────
  describe("deployAndSweepToken", () => {
    it("باید توکن BEP20 را به مادر والت sweep کند", async () => {
      const userWallet = await factory.getAddress(USER_ID);
      await token.mint(userWallet, ethers.parseUnits("100", 18));

      const before = await token.balanceOf(motherWallet.address);
      await factory.connect(relayer).deployAndSweepToken(USER_ID, await token.getAddress());
      const after  = await token.balanceOf(motherWallet.address);

      expect(after - before).to.equal(ethers.parseUnits("100", 18));
    });
  });

  // ────────────────────────────────────────────
  describe("emergencyWithdraw", () => {
    it("Owner باید بتواند BNB را اضطراری برداشت کند", async () => {
      const userWallet = await factory.getAddress(USER_ID);
      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("1") });
      await factory.connect(relayer).deployWallet(USER_ID);

      const before = await ethers.provider.getBalance(owner.address);
      await factory.connect(owner).emergencyWithdrawBNB(USER_ID, owner.address);
      const after  = await ethers.provider.getBalance(owner.address);

      expect(after).to.be.gt(before); // کمی گس کم می‌شه ولی پول می‌رسه
    });

    it("غیر Owner نباید emergency withdraw کند", async () => {
      const userWallet = await factory.getAddress(USER_ID);
      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("1") });
      await factory.connect(relayer).deployWallet(USER_ID);

      await expect(
        factory.connect(attacker).emergencyWithdrawBNB(USER_ID, attacker.address)
      ).to.be.revertedWith("Factory: not owner");
    });
  });

  // ────────────────────────────────────────────
  describe("Timelock - Mother Wallet", () => {
    it("تغییر فوری motherWallet باید ناموفق باشد", async () => {
      await factory.connect(owner).requestMotherWalletChange(attacker.address);
      await expect(
        factory.connect(owner).applyMotherWalletChange()
      ).to.be.revertedWith("Factory: timelock active");
    });

    it("بعد از ۴۸ ساعت باید اعمال شود", async () => {
      await factory.connect(owner).requestMotherWalletChange(attacker.address);

      // شبیه‌سازی گذشت ۴۸ ساعت روی شبکه local
      await ethers.provider.send("evm_increaseTime", [48 * 60 * 60 + 1]);
      await ethers.provider.send("evm_mine", []);

      await factory.connect(owner).applyMotherWalletChange();
      expect(await factory.motherWallet()).to.equal(attacker.address);
    });

    it("لغو تغییر باید کار کند", async () => {
      await factory.connect(owner).requestMotherWalletChange(attacker.address);
      await factory.connect(owner).cancelMotherWalletChange();
      expect(await factory.pendingMotherWallet()).to.equal(ethers.ZeroAddress);
    });
  });

  // ────────────────────────────────────────────
  describe("updateRelayer", () => {
    it("فقط owner بتونه relayer رو عوض کنه", async () => {
      await expect(
        factory.connect(attacker).updateRelayer(attacker.address)
      ).to.be.revertedWith("Factory: not owner");

      await factory.connect(owner).updateRelayer(attacker.address);
      expect(await factory.relayer()).to.equal(attacker.address);
    });
  });
});
