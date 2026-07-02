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

| Network Key (Project Config) | Type | Chain ID |
|---|---|---|
| `bscMainnet` | __Mainnet__ | 56 |
| `bscTestnet` | __Testnet__ | 97 |
| `opBNBMainnet` | __Mainnet__ | 204 |
| `ethereumMainnet` | __Mainnet__ | 1 |
| `sepoliaTestnet` | __Testnet__ | 11155111 |
| `polygonMainnet` | __Mainnet__ | 137 |
| `polygonAmoyTestnet` | __Testnet__ | 80002 |
| `arbitrumMainnet` | __Mainnet__ | 42161 |
| `arbitrumSepoliaTestnet` | __Testnet__ | 421614 |
| `optimismMainnet` | __Mainnet__ | 10 |
| `optimismSepoliaTestnet` | __Testnet__ | 11155420 |
| `baseMainnet` | __Mainnet__ | 8453 |
| `baseSepoliaTestnet` | __Testnet__ | 84532 |
| `avalancheMainnet` | __Mainnet__ | 43114 |
| `avalancheFujiTestnet` | __Testnet__ | 43113 |

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
│   └── createWallet.ts        # CLI tools for per-network wallet generation
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
npm run deploy:sepoliaTestnet
npm run deploy:polygonAmoyTestnet
npm run deploy:arbitrumSepoliaTestnet
npm run deploy:optimismSepoliaTestnet
npm run deploy:baseSepoliaTestnet
npm run deploy:avalancheFujiTestnet
```

**Mainnet (after testnet verification):**
```bash
npm run deploy:bscMainnet
npm run deploy:opBNBMainnet
npm run deploy:ethereumMainnet
npm run deploy:polygonMainnet
npm run deploy:arbitrumMainnet
npm run deploy:optimismMainnet
npm run deploy:baseMainnet
npm run deploy:avalancheMainnet
```

After each deploy, save the Factory address in your backend `.env`:
```
FACTORY_ADDRESS_BSC=0x...
FACTORY_ADDRESS_POLYGON=0x...
```

---

## Wallet CLI (per network)

Use the wallet generator to create dedicated deployer/relayer/mother wallets for each network.

```bash
# List supported networks
npm run wallet:list

# Generate wallets for a single network
npm run wallet:create -- --network=bscTestnet
npm run wallet:create -- --network=ethereumMainnet

# Generate wallets for all networks at once
npm run wallet:all-networks

# Generate and also append .env-style output to wallets.env
npm run wallet:all-networks -- --save

# Check the deployer balance for one network
npm run wallet:check -- --network=bscTestnet
npm run wallet:check -- --network=arbitrumSepoliaTestnet

# Check deployer balances for all networks (uses DEPLOYER_PRIVATE_KEY_<NETWORK> from .env)
npm run wallet:check -- --all-networks
```

Each network gets its own env keys, for example for BSC Testnet:

```bash
DEPLOYER_PRIVATE_KEY_BSC_TESTNET=0x...
RELAYER_ADDRESS_BSC_TESTNET=0x...
MOTHER_WALLET_BSC_TESTNET=0x...
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
