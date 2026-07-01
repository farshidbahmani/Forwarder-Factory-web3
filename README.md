# BNB Forwarder — Multi-Chain Deposit Wallet System

A smart contract system for cryptocurrency exchanges that gives each user a dedicated deposit address. Deposited funds are automatically swept to a central mother wallet — without users needing to pay gas.

## How It Works

```
User registers
  └── getAddress(userId) → deterministic address (no deployment yet)

User deposits BNB or BEP20 token to their address
  └── funds sit on the predicted address

Backend detects the deposit
  └── relayer calls deployAndSweepBNB(userId) or deployAndSweepToken(userId, token)
      └── Forwarder wallet is deployed (first time only)
      └── Funds are swept to the mother wallet in the same transaction
```

The user **never pays gas**. The relayer pays gas for the sweep transaction.

---

## Contracts

| Contract | Description |
|---|---|
| `Forwarder.sol` | Each user's deposit wallet (Minimal Proxy clone) |
| `ForwarderFactory.sol` | Deploys and manages user wallets |
| `MockBEP20.sol` | Test token — local testing only, never deploy to mainnet |

### Security Features

- **onlyFactory** — only the Factory can trigger sweeps on Forwarder wallets
- **nonReentrant** — prevents reentrancy attacks on all sweep functions
- **48h Timelock** — motherWallet changes require a 48-hour delay before taking effect
- **Emergency Withdrawal** — owner can directly rescue funds if the Factory has a bug
- **Relayer rotation** — relayer key can be changed without redeploying user wallets

### Roles

| Role | Responsibility | Recommended setup |
|---|---|---|
| `owner` | Update motherWallet, relayer, ownership | Multisig (e.g. Safe) |
| `relayer` | Pay gas for deploy + sweep transactions | Hot wallet on backend server |
| `motherWallet` | Receive all swept funds | Cold wallet or Multisig |

---

## Supported Networks

| Network | Chain ID | Testnet |
|---|---|---|
| BNB Chain | 56 | BSC Testnet (97) |
| opBNB | 204 | — |
| Ethereum | 1 | Sepolia (11155111) |
| Polygon | 137 | Amoy (80002) |
| Arbitrum | 42161 | Arbitrum Sepolia (421614) |
| Optimism | 10 | Optimism Sepolia (11155420) |
| Base | 8453 | Base Sepolia (84532) |
| Avalanche C-Chain | 43114 | Fuji (43113) |

---

## Project Structure

```
bnb-forwarder/
├── contracts/
│   ├── Forwarder.sol          # User wallet implementation
│   ├── ForwarderFactory.sol   # Factory + admin logic
│   └── MockBEP20.sol          # Test token (local only)
├── scripts/
│   ├── deploy.ts              # Deploy Factory to any network
│   └── wallet.ts              # CLI tools for wallet management
├── test/
│   └── factory.test.ts        # Hardhat tests
├── hardhat.config.ts
├── tsconfig.json
├── package.json
└── .env.example
```

---

## Setup

**1. Install dependencies**
```bash
npm install
```

**2. Create your `.env` file**
```bash
cp .env.example .env
# Fill in your values
```

**3. Compile contracts**
```bash
npm run compile
```

**4. Run tests (local network)**
```bash
npm test
```

---

## Deploy

Always deploy to testnet first and verify everything works before mainnet.

**Testnet:**
```bash
npm run deploy:bscTestnet
npm run deploy:sepolia
npm run deploy:polygonAmoy
npm run deploy:arbitrumSepolia
npm run deploy:optimismSepolia
npm run deploy:baseSepolia
npm run deploy:avalancheFuji
```

**Mainnet (after testnet verification):**
```bash
npm run deploy:bscMainnet
npm run deploy:ethereum
npm run deploy:polygon
npm run deploy:arbitrum
npm run deploy:optimism
npm run deploy:base
npm run deploy:avalanche
```

After each deploy, save the Factory address in your backend `.env`:
```
FACTORY_ADDRESS_BSC=0x...
FACTORY_ADDRESS_POLYGON=0x...
```

---

## Wallet CLI

```bash
# Print Factory info (owner, relayer, motherWallet)
npx hardhat run scripts/wallet.ts --network bscTestnet

# Get deposit address for user 1
npx hardhat run scripts/wallet.ts --network bscTestnet -- address 1

# Get addresses for multiple users at once
npx hardhat run scripts/wallet.ts --network bscTestnet -- batch 1 2 3 4 5

# Sweep BNB for user 1
npx hardhat run scripts/wallet.ts --network bscTestnet -- sweepBNB 1

# Sweep a BEP20 token for user 1
npx hardhat run scripts/wallet.ts --network bscTestnet -- sweepToken 1 0xTokenAddress
```

---

## Admin Operations

### Rotate the relayer key
```solidity
// Call on ForwarderFactory — owner only
updateRelayer(newRelayerAddress)
```

### Change the mother wallet (48h timelock)
```solidity
// Step 1: Request the change
requestMotherWalletChange(newMotherWalletAddress)

// Step 2: Wait 48 hours, then apply
applyMotherWalletChange()

// Cancel at any time before applying
cancelMotherWalletChange()
```

### Emergency withdrawal (if sweep is broken)
```solidity
// Rescue BNB from a user's wallet directly
emergencyWithdrawBNB(userId, destinationAddress)

// Rescue a BEP20 token from a user's wallet directly
emergencyWithdrawToken(userId, tokenAddress, destinationAddress)
```

---

## Before Going to Mainnet

- [ ] All tests passing (`npm test`)
- [ ] Full end-to-end test on testnet (deposit + sweep for BNB and at least one token)
- [ ] Contract verified on block explorer
- [ ] Owner transferred to a Multisig (e.g. [Safe](https://safe.global))
- [ ] Mother wallet is a cold wallet or Multisig
- [ ] Relayer wallet has sufficient native token balance for gas
- [ ] Monitoring and alerts set up for relayer balance and failed sweeps
- [ ] Security audit completed (at minimum run [Slither](https://github.com/crytic/slither))
