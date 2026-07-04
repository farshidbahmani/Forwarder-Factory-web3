# BNB Forwarder — Multi-Chain Deposit Wallet System

A smart contract system for cryptocurrency exchanges that gives each user a dedicated deposit address. Deposited funds are automatically swept to a central mother wallet — without users needing to pay gas.

## How It Works

```
User registers
  └── getAddress(userId) → deterministic address (no deployment yet)

User deposits BNB or BEP20 token to their address
  └── funds sit on the predicted address

Backend detects the deposit
  └── relayer calls deployAndSweepNative(userId) or deployAndSweepToken(userId, token)
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
| `ethereumSepoliaTestnet` | __Testnet__ | 11155111 |
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
forwarder-factory/
├── contracts/                 # Solidity smart contracts
├── src/
│   ├── domain/                # Entities, network config, contract metadata
│   ├── application/           # Use cases (wallet, deploy, contract)
│   ├── infrastructure/        # Env repo, ethers/hardhat gateways
│   └── presentation/
│       └── api/               # Express REST API
├── scripts/                   # Legacy CLI scripts (optional)
├── test/
├── hardhat.config.ts
└── package.json
```

### Clean Architecture Layers

| Layer | Responsibility |
|---|---|
| **Domain** | Network definitions, wallet types, contract function catalog |
| **Application** | Business logic — wallet generation, deploy orchestration, contract calls |
| **Infrastructure** | `.env` persistence, ethers providers, Hardhat compile/verify |
| **Presentation** | Express REST API |

---

## Setup

**1. Install dependencies**
```bash
npm install
```

**2. Create your `.env` file**
```bash
cp .env.example .env
# Fill in your values (or generate via API)
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

## API Server

```bash
npm run dev
# or
npm start
```

Base URL: http://localhost:3000

**API docs (Swagger UI):** http://localhost:3000  
OpenAPI JSON: http://localhost:3000/api/openapi.json

### Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/health` | Health check |
| GET | `/api/networks` | List all networks |
| GET | `/api/networks/:name/status` | Env config status for a network |
| GET | `/api/wallets/generate?network=` | Generate wallets for one network |
| GET | `/api/wallets/balance?network=&address=` | Native balance for an address |
| GET | `/api/wallets/status?network=` | Env config status for one network |
| POST | `/api/deploy/compile` | Compile contracts |
| POST | `/api/deploy` | Deploy factory (`{ network, verify? }`) |
| GET | `/api/contracts/functions` | List callable contract functions |
| GET | `/api/contracts/:network/info` | Deployed factory info |
| POST | `/api/contracts/call` | Call contract (`{ network, functionName, args }`) |

### Examples

```bash
# Generate wallets for BSC Testnet
curl "http://localhost:3000/api/wallets/generate?network=bscTestnet"

# Check balance for an address
curl "http://localhost:3000/api/wallets/balance?network=bscTestnet&address=0xDD281B850B8a32F2dca05f5058b6656d32C2998f"

# Deploy factory
curl -X POST http://localhost:3000/api/deploy \
  -H "Content-Type: application/json" \
  -d '{"network":"bscTestnet","verify":true}'

# Read getAddress
curl -X POST http://localhost:3000/api/contracts/call \
  -H "Content-Type: application/json" \
  -d '{"network":"bscTestnet","functionName":"getAddress","args":{"userId":"12345"}}'
```

### Per-network `.env` keys

Each network uses its own suffix, e.g. for BSC Testnet:

```bash
DEPLOYER_PRIVATE_KEY_BSC_TESTNET=0x...
RELAYER_PRIVATE_KEY_BSC_TESTNET=0x...
RELAYER_ADDRESS_BSC_TESTNET=0x...
MOTHER_WALLET_BSC_TESTNET=0x...
FACTORY_ADDRESS_BSC_TESTNET=0x...
```

---

## Deploy (API or CLI)

Always deploy to testnet first and verify everything works before mainnet.

**API:** `POST /api/deploy` with `{ "network": "bscTestnet" }`

**CLI (legacy):**
```bash
npm run deploy:bscTestnet
npm run deploy:ethereumSepoliaTestnet
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

## Wallet Management (API or CLI)

**API:** `GET /api/wallets/generate?network=bscTestnet`

**CLI (legacy):**

```bash
# List supported networks
npm run wallet:list

# Generate wallets for a single network
npm run wallet:create -- --network=bscTestnet

# Generate wallets for all networks at once
npm run wallet:all-networks

# Check deployer balance
npm run wallet:check -- --network=bscTestnet
npm run wallet:check -- --all-networks
```

Each network gets its own env keys, for example for BSC Testnet:

```bash
DEPLOYER_PRIVATE_KEY_BSC_TESTNET=0x...
RELAYER_PRIVATE_KEY_BSC_TESTNET=0x...
RELAYER_ADDRESS_BSC_TESTNET=0x...
MOTHER_WALLET_BSC_TESTNET=0x...
```

---

## Contract Calls (API)

Use `POST /api/contracts/call` for read/write functions.

- **Read:** `getAddress`, `motherWallet`, `relayer`, timelock state, etc.
- **Write (relayer):** `deployWallet`, `deployAndSweepNative`, `deployAndSweepToken`
- **Write (owner):** `updateRelayer`, `transferOwnership`, emergency withdrawals, mother wallet timelock

List all functions: `GET /api/contracts/functions`

---

## Admin Operations (on-chain reference)

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
// Rescue native token from a user's wallet directly
emergencyWithdrawNative(userId, destinationAddress)

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
