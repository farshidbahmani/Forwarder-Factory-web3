import { ethers } from "ethers";
import {
  DepositEvent,
  MONITORED_WALLETS,
  NetworkMonitorStatus,
  SweepResult,
} from "../domain/monitor";
import { AppError } from "../domain/errors";
import { envKeyForNetwork, getNetwork, NetworkConfig } from "../domain/network";
import { EnvRepository } from "../infrastructure/env/env.repository";
import { ProviderFactory } from "../infrastructure/blockchain/provider.factory";
import { BlockListener } from "../infrastructure/blockchain/block.listener";
import { ContractService } from "../application/contract.service";

const MAX_RECENT_SWEEPS = 50;

export type NetworkMonitorDeps = {
  envRepo: EnvRepository;
  providerFactory: ProviderFactory;
  contractService: ContractService;
};

export class NetworkMonitor {
  private listener?: BlockListener;
  private provider?: ethers.JsonRpcProvider;
  private watchedAddresses: { userId: string; address: string; label?: string }[] = [];
  private readonly recentSweeps: SweepResult[] = [];

  constructor(
    readonly networkName: string,
    private readonly deps: NetworkMonitorDeps,
  ) {}

  get network(): NetworkConfig {
    return getNetwork(this.networkName);
  }

  isRunning(): boolean {
    return this.listener?.isRunning() ?? false;
  }

  getStatus(): NetworkMonitorStatus {
    return {
      network: this.networkName,
      running: this.isRunning(),
      lastBlock: this.listener?.getLastBlock(),
      watchedAddresses: this.watchedAddresses,
      recentSweeps: this.recentSweeps,
    };
  }

  async resolveAddresses() {
    const watched: { userId: string; address: string; label?: string }[] = [];

    for (const wallet of MONITORED_WALLETS) {
      const result = await this.deps.contractService.call(this.networkName, "getAddress", {
        userId: wallet.userId,
      });
      watched.push({
        userId: wallet.userId,
        address: ethers.getAddress(result.result as string),
        label: wallet.label,
      });
    }

    return watched;
  }

  async start(): Promise<NetworkMonitorStatus> {
    if (this.isRunning()) return this.getStatus();

    this.assertFactoryConfigured();

    this.watchedAddresses = await this.resolveAddresses();
    const addressToUserId = new Map(
      this.watchedAddresses.map((w) => [ethers.getAddress(w.address), w.userId]),
    );

    const { provider } = this.deps.providerFactory.getProvider(this.networkName);
    this.provider = provider;

    this.listener = new BlockListener({
      networkName: this.networkName,
      provider,
      addressToUserId,
      onDeposit: (deposit) => this.handleDeposit(deposit),
    });

    await this.listener.start();

    console.log(
      `[monitor:${this.networkName}] started — watching ${this.watchedAddresses.length} wallet(s)`,
    );

    return this.getStatus();
  }

  stop(): NetworkMonitorStatus {
    if (!this.isRunning()) return this.getStatus();

    this.listener!.stop();
    this.listener = undefined;
    this.provider = undefined;

    console.log(`[monitor:${this.networkName}] stopped`);

    return this.getStatus();
  }

  private assertFactoryConfigured(): void {
    const factoryKey = envKeyForNetwork("FACTORY_ADDRESS", this.network);
    if (!this.deps.envRepo.get(factoryKey)) {
      throw new AppError(`No factory deployed. Set ${factoryKey} in .env`);
    }
  }

  private async handleDeposit(deposit: DepositEvent): Promise<void> {
    const sweep: SweepResult = {
      userId: deposit.userId,
      type: deposit.type,
      txHash: deposit.txHash,
    };

    console.log(
      `[monitor:${this.networkName}] deposit: userId=${deposit.userId} ` +
        `type=${deposit.type} tx=${deposit.txHash}`,
    );

    try {
      if (deposit.type === "native") {
        const result = await this.deps.contractService.call(
          this.networkName,
          "deployAndSweepNative",
          { userId: deposit.userId },
        );
        sweep.sweepTxHash = result.txHash;
      } else {
        const result = await this.deps.contractService.call(
          this.networkName,
          "deployAndSweepToken",
          { userId: deposit.userId, token: deposit.token! },
        );
        sweep.sweepTxHash = result.txHash;
      }

      console.log(
        `[monitor:${this.networkName}] sweep sent: userId=${deposit.userId} ` +
          `sweepTx=${sweep.sweepTxHash}`,
      );
    } catch (e) {
      sweep.error = e instanceof Error ? e.message : String(e);
      console.error(
        `[monitor:${this.networkName}] sweep failed: userId=${deposit.userId} — ${sweep.error}`,
      );
    }

    this.recentSweeps.unshift(sweep);
    if (this.recentSweeps.length > MAX_RECENT_SWEEPS) {
      this.recentSweeps.length = MAX_RECENT_SWEEPS;
    }
  }
}
