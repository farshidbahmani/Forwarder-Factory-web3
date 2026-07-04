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
    it("should return a predictable address before deploy", async () => {
      const predicted = await factory.getAddress(USER_ID);
      expect(predicted).to.be.properAddress;
      expect(await ethers.provider.getCode(predicted)).to.equal("0x");
    });

    it("different userIds should produce different addresses", async () => {
      const addr1 = await factory.getAddress(1n);
      const addr2 = await factory.getAddress(2n);
      expect(addr1).to.not.equal(addr2);
    });
  });

  // ────────────────────────────────────────────
  describe("deployAndSweepNative", () => {
    it("should sweep native token to the mother wallet", async () => {
      const userWallet = await factory.getAddress(USER_ID);
      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("1") });

      const before = await ethers.provider.getBalance(motherWallet.address);
      await factory.connect(relayer).deployAndSweepNative(USER_ID);
      const after  = await ethers.provider.getBalance(motherWallet.address);

      expect(after - before).to.equal(ethers.parseEther("1"));
    });

    it("second sweep should work without redeploying", async () => {
      const userWallet = await factory.getAddress(USER_ID);

      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("1") });
      await factory.connect(relayer).deployAndSweepNative(USER_ID);

      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("0.5") });
      await expect(
        factory.connect(relayer).deployAndSweepNative(USER_ID)
      ).to.not.emit(factory, "WalletDeployed");
    });

    it("non-relayer should not be able to sweep", async () => {
      await expect(
        factory.connect(attacker).deployAndSweepNative(USER_ID)
      ).to.be.revertedWith("Factory: not relayer");
    });
  });

  // ────────────────────────────────────────────
  describe("deployAndSweepToken", () => {
    it("should sweep BEP20 tokens to the mother wallet", async () => {
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
    it("Owner should be able to emergency withdraw native token", async () => {
      const userWallet = await factory.getAddress(USER_ID);
      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("1") });
      await factory.connect(relayer).deployWallet(USER_ID);

      const before = await ethers.provider.getBalance(owner.address);
      await factory.connect(owner).emergencyWithdrawNative(USER_ID, owner.address);
      const after  = await ethers.provider.getBalance(owner.address);

      expect(after).to.be.gt(before); // some gas is spent, but funds arrive
    });

    it("non-Owner should not be able to emergency withdraw", async () => {
      const userWallet = await factory.getAddress(USER_ID);
      await user.sendTransaction({ to: userWallet, value: ethers.parseEther("1") });
      await factory.connect(relayer).deployWallet(USER_ID);

      await expect(
        factory.connect(attacker).emergencyWithdrawNative(USER_ID, attacker.address)
      ).to.be.revertedWith("Factory: not owner");
    });
  });

  // ────────────────────────────────────────────
  describe("Timelock - Mother Wallet", () => {
    it("immediate motherWallet change should fail", async () => {
      await factory.connect(owner).requestMotherWalletChange(attacker.address);
      await expect(
        factory.connect(owner).applyMotherWalletChange()
      ).to.be.revertedWith("Factory: timelock active");
    });

    it("should apply after 48 hours", async () => {
      await factory.connect(owner).requestMotherWalletChange(attacker.address);

      // Simulate 48 hours passing on the local network
      await ethers.provider.send("evm_increaseTime", [48 * 60 * 60 + 1]);
      await ethers.provider.send("evm_mine", []);

      await factory.connect(owner).applyMotherWalletChange();
      expect(await factory.motherWallet()).to.equal(attacker.address);
    });

    it("cancelling a pending change should work", async () => {
      await factory.connect(owner).requestMotherWalletChange(attacker.address);
      await factory.connect(owner).cancelMotherWalletChange();
      expect(await factory.pendingMotherWallet()).to.equal(ethers.ZeroAddress);
    });
  });

  // ────────────────────────────────────────────
  describe("updateRelayer", () => {
    it("only owner should be able to change the relayer", async () => {
      await expect(
        factory.connect(attacker).updateRelayer(attacker.address)
      ).to.be.revertedWith("Factory: not owner");

      await factory.connect(owner).updateRelayer(attacker.address);
      expect(await factory.relayer()).to.equal(attacker.address);
    });
  });
});
