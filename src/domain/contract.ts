export type ContractCallType = "read" | "write";

export type ContractParamType = "uint256" | "address" | "none";

export type ContractParam = {
  name: string;
  type: ContractParamType;
  label: string;
  placeholder?: string;
};

export type ContractRole = "any" | "owner" | "relayer";

export type ContractFunctionDef = {
  name: string;
  label: string;
  description: string;
  type: ContractCallType;
  role: ContractRole;
  inputs: ContractParam[];
};

export const FORWARDER_FACTORY_FUNCTIONS: ContractFunctionDef[] = [
  {
    name: "getAddress",
    label: "Get User Wallet Address",
    description: "Predict deterministic deposit address for a userId (no deployment needed).",
    type: "read",
    role: "any",
    inputs: [{ name: "userId", type: "uint256", label: "User ID", placeholder: "12345" }],
  },
  {
    name: "implementation",
    label: "Implementation Address",
    description: "Forwarder implementation contract used for clones.",
    type: "read",
    role: "any",
    inputs: [],
  },
  {
    name: "motherWallet",
    label: "Mother Wallet",
    description: "Address that receives swept funds.",
    type: "read",
    role: "any",
    inputs: [],
  },
  {
    name: "relayer",
    label: "Relayer",
    description: "Address authorized to deploy and sweep.",
    type: "read",
    role: "any",
    inputs: [],
  },
  {
    name: "owner",
    label: "Owner",
    description: "Factory owner (admin).",
    type: "read",
    role: "any",
    inputs: [],
  },
  {
    name: "TIMELOCK_DELAY",
    label: "Timelock Delay",
    description: "Seconds required before mother wallet change.",
    type: "read",
    role: "any",
    inputs: [],
  },
  {
    name: "pendingMotherWallet",
    label: "Pending Mother Wallet",
    description: "Mother wallet awaiting timelock.",
    type: "read",
    role: "any",
    inputs: [],
  },
  {
    name: "motherWalletUnlockTime",
    label: "Mother Wallet Unlock Time",
    description: "Unix timestamp when pending change can be applied.",
    type: "read",
    role: "any",
    inputs: [],
  },
  {
    name: "deployWallet",
    label: "Deploy Wallet",
    description: "Deploy user forwarder wallet (relayer only).",
    type: "write",
    role: "relayer",
    inputs: [{ name: "userId", type: "uint256", label: "User ID", placeholder: "12345" }],
  },
  {
    name: "deployAndSweepNative",
    label: "Deploy & Sweep Native",
    description: "Deploy wallet and sweep native token to mother wallet.",
    type: "write",
    role: "relayer",
    inputs: [{ name: "userId", type: "uint256", label: "User ID", placeholder: "12345" }],
  },
  {
    name: "deployAndSweepToken",
    label: "Deploy & Sweep Token",
    description: "Deploy wallet and sweep ERC20 token to mother wallet.",
    type: "write",
    role: "relayer",
    inputs: [
      { name: "userId", type: "uint256", label: "User ID", placeholder: "12345" },
      { name: "token", type: "address", label: "Token Address", placeholder: "0x..." },
    ],
  },
  {
    name: "emergencyWithdrawNative",
    label: "Emergency Withdraw Native",
    description: "Owner rescues native token from a user wallet.",
    type: "write",
    role: "owner",
    inputs: [
      { name: "userId", type: "uint256", label: "User ID", placeholder: "12345" },
      { name: "to", type: "address", label: "Destination", placeholder: "0x..." },
    ],
  },
  {
    name: "emergencyWithdrawToken",
    label: "Emergency Withdraw Token",
    description: "Owner rescues ERC20 from a user wallet.",
    type: "write",
    role: "owner",
    inputs: [
      { name: "userId", type: "uint256", label: "User ID", placeholder: "12345" },
      { name: "token", type: "address", label: "Token Address", placeholder: "0x..." },
      { name: "to", type: "address", label: "Destination", placeholder: "0x..." },
    ],
  },
  {
    name: "requestMotherWalletChange",
    label: "Request Mother Wallet Change",
    description: "Start 48h timelock for mother wallet update.",
    type: "write",
    role: "owner",
    inputs: [{ name: "newMotherWallet", type: "address", label: "New Mother Wallet", placeholder: "0x..." }],
  },
  {
    name: "applyMotherWalletChange",
    label: "Apply Mother Wallet Change",
    description: "Apply pending mother wallet after timelock.",
    type: "write",
    role: "owner",
    inputs: [],
  },
  {
    name: "cancelMotherWalletChange",
    label: "Cancel Mother Wallet Change",
    description: "Cancel pending mother wallet change.",
    type: "write",
    role: "owner",
    inputs: [],
  },
  {
    name: "updateRelayer",
    label: "Update Relayer",
    description: "Set new relayer address.",
    type: "write",
    role: "owner",
    inputs: [{ name: "newRelayer", type: "address", label: "New Relayer", placeholder: "0x..." }],
  },
  {
    name: "transferOwnership",
    label: "Transfer Ownership",
    description: "Transfer factory ownership (e.g. to multisig).",
    type: "write",
    role: "owner",
    inputs: [{ name: "newOwner", type: "address", label: "New Owner", placeholder: "0x..." }],
  },
];

export type DeployResult = {
  network: string;
  factoryAddress: string;
  implementationAddress: string;
  deployerAddress: string;
  deployerBalance: string;
  symbol: string;
  verified: boolean;
  verificationMessage?: string;
  envKey: string;
};

export type ContractCallResult = {
  functionName: string;
  type: ContractCallType;
  result?: unknown;
  txHash?: string;
  blockNumber?: number;
  gasUsed?: string;
};
