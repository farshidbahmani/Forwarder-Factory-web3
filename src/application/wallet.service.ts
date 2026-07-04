import { ethers } from "ethers";
import { EnvSnippet, NetworkWallets, WalletBalance } from "../domain/wallet";
import { envKeyForNetwork, getNetwork, NETWORKS } from "../domain/network";
import { AppError } from "../domain/errors";
import { EnvRepository } from "../infrastructure/env/env.repository";
import { ProviderFactory } from "../infrastructure/blockchain/provider.factory";

export class WalletService {
  constructor(
    private readonly envRepo: EnvRepository,
    private readonly providerFactory: ProviderFactory,
  ) {}

  listNetworks() {
    return NETWORKS;
  }

  private toWallet(w: ethers.HDNodeWallet) {
    return {
      address: w.address,
      privateKey: w.privateKey,
      mnemonic: w.mnemonic?.phrase,
    };
  }

  generateForNetwork(networkName: string): NetworkWallets {
    getNetwork(networkName);
    return {
      network: networkName,
      deployer: this.toWallet(ethers.Wallet.createRandom()),
      relayer: this.toWallet(ethers.Wallet.createRandom()),
      mother: this.toWallet(ethers.Wallet.createRandom()),
    };
  }

  toEnvSnippet(wallets: NetworkWallets): EnvSnippet {
    const network = getNetwork(wallets.network);
    const lines = [
      `# ${wallets.network}`,
      `${envKeyForNetwork("DEPLOYER_PRIVATE_KEY", network)}=${wallets.deployer.privateKey}`,
      `${envKeyForNetwork("RELAYER_PRIVATE_KEY", network)}=${wallets.relayer.privateKey}`,
      `${envKeyForNetwork("RELAYER_ADDRESS", network)}=${wallets.relayer.address}`,
      `${envKeyForNetwork("MOTHER_WALLET", network)}=${wallets.mother.address}`,
    ];
    return { network: wallets.network, lines };
  }

  toEnvText(wallets: NetworkWallets): string {
    return `${this.toEnvSnippet(wallets).lines.join("\n")}\n`;
  }

  async checkBalance(networkName: string, address: string): Promise<WalletBalance> {
    const network = getNetwork(networkName);
    if (!ethers.isAddress(address)) {
      throw new AppError("Invalid address");
    }

    const { provider } = this.providerFactory.getProvider(networkName);
    const balance = await provider.getBalance(address);

    return {
      network: networkName,
      chainId: network.chainId,
      symbol: network.symbol,
      address,
      balance: ethers.formatEther(balance),
    };
  }

  getEnvStatus(networkName: string) {
    const network = getNetwork(networkName);
    return {
      network: networkName,
      deployerKey: Boolean(this.envRepo.getForNetwork("DEPLOYER_PRIVATE_KEY", network.envSuffix)),
      relayerKey: Boolean(this.envRepo.getForNetwork("RELAYER_PRIVATE_KEY", network.envSuffix)),
      relayerAddress: this.envRepo.getForNetwork("RELAYER_ADDRESS", network.envSuffix) ?? null,
      motherWallet: this.envRepo.getForNetwork("MOTHER_WALLET", network.envSuffix) ?? null,
      factoryAddress: this.envRepo.get(envKeyForNetwork("FACTORY_ADDRESS", network)) ?? null,
    };
  }
}
