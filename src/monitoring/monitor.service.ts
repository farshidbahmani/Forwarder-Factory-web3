import { MONITORED_WALLETS, NetworkMonitorStatus } from "../domain/monitor";
import { getNetwork } from "../domain/network";
import { AppError } from "../domain/errors";
import { EnvRepository } from "../infrastructure/env/env.repository";
import { ProviderFactory } from "../infrastructure/blockchain/provider.factory";
import { ContractService } from "../application/contract.service";
import { NetworkMonitor } from "./network.monitor";

export class MonitorService {
  private readonly monitors = new Map<string, NetworkMonitor>();

  constructor(
    private readonly envRepo: EnvRepository,
    private readonly providerFactory: ProviderFactory,
    private readonly contractService: ContractService,
  ) {}

  listMonitoredWallets() {
    return MONITORED_WALLETS;
  }

  isRunning(networkName: string): boolean {
    return this.monitors.get(networkName)?.isRunning() ?? false;
  }

  getStatus(networkName: string): NetworkMonitorStatus {
    this.assertNetwork(networkName);
    return this.monitors.get(networkName)?.getStatus() ?? {
      network: networkName,
      running: false,
      watchedAddresses: [],
      recentSweeps: [],
    };
  }

  listRunning(): NetworkMonitorStatus[] {
    return [...this.monitors.values()]
      .filter((monitor) => monitor.isRunning())
      .map((monitor) => monitor.getStatus());
  }

  async resolveAddresses(networkName: string) {
    this.assertNetwork(networkName);
    return this.getOrCreateMonitor(networkName).resolveAddresses();
  }

  async start(networkName: string): Promise<NetworkMonitorStatus> {
    this.assertNetwork(networkName);
    return this.getOrCreateMonitor(networkName).start();
  }

  stop(networkName: string): NetworkMonitorStatus {
    this.assertNetwork(networkName);
    const monitor = this.monitors.get(networkName);
    if (!monitor) {
      return {
        network: networkName,
        running: false,
        watchedAddresses: [],
        recentSweeps: [],
      };
    }

    const status = monitor.stop();
    this.monitors.delete(networkName);
    return status;
  }

  private getOrCreateMonitor(networkName: string): NetworkMonitor {
    let monitor = this.monitors.get(networkName);
    if (!monitor) {
      monitor = new NetworkMonitor(networkName, {
        envRepo: this.envRepo,
        providerFactory: this.providerFactory,
        contractService: this.contractService,
      });
      this.monitors.set(networkName, monitor);
    }
    return monitor;
  }

  private assertNetwork(networkName: string): void {
    try {
      getNetwork(networkName);
    } catch {
      throw new AppError(`Unknown network: ${networkName}`);
    }
  }
}
