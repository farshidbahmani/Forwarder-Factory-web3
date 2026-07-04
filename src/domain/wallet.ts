export type WalletRole = "deployer" | "relayer" | "mother";

export type GeneratedWallet = {
  address: string;
  privateKey: string;
  mnemonic?: string;
};

export type NetworkWallets = {
  network: string;
  deployer: GeneratedWallet;
  relayer: GeneratedWallet;
  mother: GeneratedWallet;
};

export type WalletBalance = {
  network: string;
  chainId: number;
  symbol: string;
  address: string;
  balance: string;
};

export type EnvSnippet = {
  network: string;
  lines: string[];
};
