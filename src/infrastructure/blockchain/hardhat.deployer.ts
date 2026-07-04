import { execSync } from "child_process";
import { ethers } from "ethers";
import { DeployResult } from "../../domain/contract";
import { AppError } from "../../domain/errors";
import { envKeyForNetwork, getNetwork } from "../../domain/network";
import { EnvRepository } from "../env/env.repository";
import { ProviderFactory } from "./provider.factory";

export class HardhatDeployer {
  constructor(
    private readonly envRepo: EnvRepository,
    private readonly providerFactory: ProviderFactory,
  ) {}

  compile(): void {
    execSync("npx hardhat compile", { stdio: "pipe", cwd: process.cwd() });
  }

  async deploy(networkName: string, verify = true): Promise<DeployResult> {
    const network = getNetwork(networkName);
    const motherWallet = this.envRepo.getForNetwork("MOTHER_WALLET", network.envSuffix);
    const relayer = this.envRepo.getForNetwork("RELAYER_ADDRESS", network.envSuffix);

    if (!motherWallet || !relayer) {
      throw new AppError(
        `Set MOTHER_WALLET_${network.envSuffix} and RELAYER_ADDRESS_${network.envSuffix} in .env`,
      );
    }

    this.compile();

    const signer = this.providerFactory.getSigner(networkName, "deployer");
    const balance = await signer.provider!.getBalance(signer.address);

    if (balance === 0n) {
      throw new AppError(
        `Deployer ${signer.address} has 0 ${network.symbol}. Fund it before deploying.`,
      );
    }

    const Factory = new ethers.ContractFactory(
      this.providerFactory.getFactoryAbi(),
      this.providerFactory.getFactoryBytecode(),
      signer,
    );

    const factory = await Factory.deploy(motherWallet, relayer);
    await factory.waitForDeployment();

    const factoryAddress = await factory.getAddress();
    const implementationAddress = await factory.getFunction("implementation")();

    const envKey = envKeyForNetwork("FACTORY_ADDRESS", network);
    this.envRepo.setMany({ [envKey]: factoryAddress });

    let verified = false;
    let verificationMessage: string | undefined;

    if (verify) {
      try {
        execSync(
          `npx hardhat verify --network ${networkName} ${factoryAddress} "${motherWallet}" "${relayer}"`,
          { stdio: "pipe", cwd: process.cwd(), timeout: 120_000 },
        );
        verified = true;
      } catch (e: unknown) {
        const msg = e instanceof Error ? e.message : String(e);
        verificationMessage = msg.includes("Already Verified")
          ? "Already verified"
          : msg.slice(0, 500);
      }
    }

    return {
      network: networkName,
      factoryAddress,
      implementationAddress,
      deployerAddress: signer.address,
      deployerBalance: ethers.formatEther(balance),
      symbol: network.symbol,
      verified,
      verificationMessage,
      envKey,
    };
  }
}
