export type MonitoredWallet = {
  userId: string;
  label?: string;
};

/** والت‌هایی که مانیتور می‌شوند — فعلاً آرایه ثابت */
export const MONITORED_WALLETS: MonitoredWallet[] = [
  { userId: "1", label: "user-1" },
  { userId: "2", label: "user-2" },
];

export type DepositType = "native" | "token";

export type DepositEvent = {
  network: string;
  userId: string;
  type: DepositType;
  txHash: string;
  blockNumber: number;
  token?: string;
};

export type SweepResult = {
  userId: string;
  type: DepositType;
  txHash: string;
  sweepTxHash?: string;
  error?: string;
};

export type NetworkMonitorStatus = {
  network: string;
  running: boolean;
  lastBlock?: number;
  watchedAddresses: { userId: string; address: string; label?: string }[];
  recentSweeps: SweepResult[];
};
