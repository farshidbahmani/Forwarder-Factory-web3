import { ethers } from "ethers";
import { getNetwork, getRpcUrl, NetworkConfig } from "../../domain/network";
import { EnvRepository } from "../env/env.repository";
import { AppError } from "../../domain/errors";
import ForwarderFactoryArtifact from "../../../artifacts/contracts/ForwarderFactory.sol/ForwarderFactory.json";

export class ProviderFactory {
  constructor(private readonly envRepo: EnvRepository) {}

  getProvider(networkName: string): { provider: ethers.JsonRpcProvider; network: NetworkConfig } {
    const network = getNetwork(networkName);
    const url = getRpcUrl(network, process.env);
    return { provider: new ethers.JsonRpcProvider(url), network };
  }

  getSigner(networkName: string, role: "deployer" | "relayer" | "owner"): ethers.Wallet {
    const { provider, network } = this.getProvider(networkName);
    const roleKey =
      role === "relayer"
        ? "RELAYER_PRIVATE_KEY"
        : "DEPLOYER_PRIVATE_KEY";

    const privateKey = this.envRepo.getForNetwork(roleKey, network.envSuffix);
    if (!privateKey || privateKey === "0x...") {
      throw new AppError(
        `Missing ${roleKey}_${network.envSuffix} (or global ${roleKey}) in .env`,
      );
    }
    return new ethers.Wallet(privateKey, provider);
  }

  getFactoryContract(networkName: string, factoryAddress: string, signer?: ethers.Signer) {
    const { provider } = this.getProvider(networkName);
    return new ethers.Contract(
      factoryAddress,
      ForwarderFactoryArtifact.abi,
      signer ?? provider,
    );
  }

  getFactoryBytecode(): string {
    return ForwarderFactoryArtifact.bytecode;
  }

  getFactoryAbi() {
    return ForwarderFactoryArtifact.abi;
  }
}
