import { ethers } from "ethers";
import { DepositEvent } from "../../domain/monitor";

const TRANSFER_TOPIC = ethers.id("Transfer(address,address,uint256)");

export type BlockListenerOptions = {
  networkName: string;
  provider: ethers.JsonRpcProvider;
  addressToUserId: Map<string, string>;
  onDeposit: (deposit: DepositEvent) => Promise<void>;
};

export class BlockListener {
  private running = false;
  private lastBlock?: number;
  private readonly processedKeys = new Set<string>();
  private processing = false;
  private pendingBlock?: number;

  constructor(private readonly options: BlockListenerOptions) {}

  getLastBlock(): number | undefined {
    return this.lastBlock;
  }

  isRunning(): boolean {
    return this.running;
  }

  async start(): Promise<void> {
    if (this.running) return;
    this.running = true;

    const { provider } = this.options;
    const current = await provider.getBlockNumber();
    this.lastBlock = current;

    provider.on("block", (blockNumber) => {
      void this.enqueueBlock(blockNumber);
    });
  }

  stop(): void {
    if (!this.running) return;
    this.running = false;
    this.options.provider.removeAllListeners("block");
  }

  private async enqueueBlock(blockNumber: number): Promise<void> {
    this.pendingBlock = blockNumber;
    if (this.processing) return;

    this.processing = true;
    try {
      while (this.pendingBlock !== undefined) {
        const next = this.pendingBlock;
        this.pendingBlock = undefined;
        await this.processBlock(next);
        this.lastBlock = next;
      }
    } finally {
      this.processing = false;
    }
  }

  private async processBlock(blockNumber: number): Promise<void> {
    const { provider, networkName, addressToUserId, onDeposit } = this.options;
    const watched = new Set(addressToUserId.keys());
    if (watched.size === 0) return;

    const block = await provider.getBlock(blockNumber, true);
    if (!block) return;

    for (const tx of block.prefetchedTransactions ?? []) {
      if (!tx.to || tx.value === 0n) continue;
      const to = ethers.getAddress(tx.to);
      const userId = addressToUserId.get(to);
      if (!userId) continue;

      const key = `${tx.hash}:native`;
      if (this.processedKeys.has(key)) continue;
      this.processedKeys.add(key);

      await onDeposit({
        network: networkName,
        userId,
        type: "native",
        txHash: tx.hash,
        blockNumber,
      });
    }

    const logs = await provider.getLogs({
      fromBlock: blockNumber,
      toBlock: blockNumber,
      topics: [TRANSFER_TOPIC],
    });

    for (const log of logs) {
      if (!log.topics[2]) continue;
      const to = ethers.getAddress(ethers.dataSlice(log.topics[2], 12));
      const userId = addressToUserId.get(to);
      if (!userId) continue;

      const key = `${log.transactionHash}:${log.address}:token`;
      if (this.processedKeys.has(key)) continue;
      this.processedKeys.add(key);

      await onDeposit({
        network: networkName,
        userId,
        type: "token",
        token: log.address,
        txHash: log.transactionHash,
        blockNumber,
      });
    }
  }
}
