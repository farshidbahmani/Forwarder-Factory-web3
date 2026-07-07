# Forwarder Factory

Multi-chain deposit wallet system for exchanges, payment gateways, and custodial platforms. Each user gets a **unique, deterministic deposit address** without managing thousands of private keys. Deposits are detected on-chain and **automatically swept** to a central treasury wallet.

Smart contracts are built with **Foundry**; the backend is **Go** with a REST API and Swagger UI.

---

## The Problem

Platforms that accept crypto deposits face a painful trade-off:

| Approach | Drawback |
|----------|----------|
| One shared hot wallet per chain | Cannot attribute deposits to individual users |
| One private key per user | Key management at scale is expensive and risky |
| HD wallet derivation off-chain | Requires secure signing infrastructure per address |

**Forwarder Factory** solves this with **CREATE2 minimal proxies**:

1. Call `getAddress(userId)` — get a unique deposit address **before** deploying anything on-chain.
2. User sends native coin (BNB, ETH, TRX, …) or tokens (ERC-20, TRC-20) to that address.
3. The backend **monitor** detects the deposit and calls `deployAndSweepNative` / `deployAndSweepToken`.
4. Funds land in the **mother wallet** — your treasury (ideally a multisig).

No per-user private keys exist. Addresses are mathematically derived from `userId` and the factory contract. Wallets are deployed lazily (only when a sweep is needed), saving gas.

---

## How It Works

```
┌─────────────┐     deposit      ┌──────────────────┐    sweep tx     ┌───────────────┐
│    User     │ ───────────────► │ Forwarder (clone)│ ────────────────► │ Mother Wallet │
│  (anyone)   │   BNB/USDT/TRX   │  per userId      │   via Relayer   │  (treasury)   │
└─────────────┘                  └────────┬─────────┘                 └───────────────┘
                                          │ onlyFactory
                                          ▼
                                 ┌──────────────────┐
                                 │ ForwarderFactory │
                                 │  owner / relayer │
                                 └──────────────────┘
```

### Roles

| Role | Key holder | On-chain powers |
|------|------------|-----------------|
| **Owner** | Cold wallet / multisig | Transfer ownership (two-step), timelocked mother-wallet change, emergency rescue, update relayer |
| **Relayer** | Backend server hot wallet | Deploy user wallets, sweep native & tokens |
| **Mother wallet** | Multisig (recommended) | Receives all swept funds — no signing required |
| **Deployer** | One-time setup | Deploys the factory contract |

### Core contracts

- **`ForwarderFactory`** (EVM) — deploys EIP-1167 clones via OpenZeppelin `Clones`, predicts addresses with CREATE2.
- **`ForwarderFactoryTron`** (Tron TVM) — same logic; uses `TronClones` for correct `0x41`-prefixed address prediction.
- **`Forwarder`** — per-user minimal proxy. Only the factory can sweep. Funds always go to `motherWallet`.

### Backend services

- **Wallet API** — generate deployer/relayer/mother keys, check balances, env status.
- **Deploy API** — compile & deploy factory via Foundry, verify on block explorers.
- **Contract API** — read factory state, call any factory function.
- **Monitor** — watch registered deposit addresses; auto-sweep on incoming transfers.

---

## Security Model

### On-chain protections

| Mechanism | What it prevents |
|-----------|------------------|
| **`onlyFactory` on Forwarder** | Random addresses cannot drain user wallets |
| **`onlyRelayer` on deploy/sweep** | Only the authorized backend can move funds (to mother wallet) |
| **`_requireForwarderWallet`** | Sweeps rejected for foreign or undeployed contracts |
| **`ReentrancyGuard`** | Reentrancy during token/native transfers |
| **`SafeERC20`** | Non-standard ERC-20 transfer failures |
| **Two-step ownership transfer** | Accidental permanent loss of factory ownership |
| **48-hour timelock on mother wallet** | Sudden redirect of treasury funds |
| **Emergency withdraw → motherWallet only** | Compromised owner cannot send rescued funds to an arbitrary address |
| **Immutable implementation (EVM)** | Clone logic cannot be swapped after deploy |

### Known limitations

- **ERC-777 tokens** trigger `tokensReceived` hooks on transfer. `nonReentrant` mitigates reentrancy, but ERC-777 tokens should be reviewed before whitelisting.
- **Relayer key compromise** lets an attacker sweep funds — but only **to** the mother wallet, not to their own address. Still disruptive (DoS, griefing) and should be rotated via `updateRelayer`.
- **Owner key compromise** enables emergency withdraws and (after 48 h) mother-wallet redirection. **Use a multisig as owner on mainnet.**

### Backend / operational security

> **The API has no built-in authentication.** Any client that can reach the server can deploy contracts, call factory functions, and start monitors. In production:
>
> - Run behind a VPN, reverse proxy, or API gateway with auth.
> - Never expose the server to the public internet without protection.
> - Store `RELAYER_PRIVATE_KEY` and `DEPLOYER_PRIVATE_KEY` in a secrets manager, not in plain `.env` on shared hosts.
> - Use a **multisig** for `MOTHER_WALLET` and factory **owner** on mainnet.
> - Fund the relayer with just enough gas for sweeps; it never holds user deposits long-term.

---

## Quick Start

```bash
# Prerequisites: Go 1.22+, Foundry (forge/cast)
curl -L https://foundry.paradigm.xyz | bash && foundryup

forge install          # first clone only — installs lib/ deps
cp .env.example .env   # fill in keys and RPC URLs

forge build            # compile contracts
forge test             # run contract tests
make dev               # API with hot reload (or: go run ./cmd/server)
```

- API: `http://localhost:3000`
- Swagger UI: `http://localhost:3000/`
- OpenAPI spec: `http://localhost:3000/api/openapi.json`

---

## Setup Workflow

### 1. Generate wallets

```bash
curl "http://localhost:3000/api/wallets/generate?network=bnbTestnet&format=env"
```

Creates deployer, relayer, and mother wallets. Paste the output into `.env`.

### 2. Deploy factory

```bash
curl -X POST http://localhost:3000/api/deploy \
  -H "Content-Type: application/json" \
  -d '{"network": "bnbTestnet", "verify": true}'
```

Set the returned `FACTORY_ADDRESS` in `.env`. On Tron, use `"completeSetup": true` for the two-step factory + implementation deploy.

### 3. Get a user's deposit address

```bash
curl -X POST http://localhost:3000/api/contracts/call \
  -H "Content-Type: application/json" \
  -d '{"network":"bnbTestnet","functionName":"getAddress","args":{"userId":"42"}}'
```

Show this address to the user. No on-chain deployment is needed yet.

### 4. Register wallets & start monitoring

```bash
# Register user deposit addresses to watch
curl -X POST http://localhost:3000/api/monitor/wallets \
  -H "Content-Type: application/json" \
  -d '{
    "network": "bnbTestnet",
    "setting": {
      "minNativeBalance": 0.001,
      "tokens": [{"token": "0x...", "minTokenBalance": 1}]
    },
    "wallets": {
      "42": "0x<PredictedAddressFromGetAddress>"
    }
  }'

# Start the block listener
curl -X POST http://localhost:3000/api/monitor/start \
  -H "Content-Type: application/json" \
  -d '{"network": "bnbTestnet"}'
```

When a deposit meets the minimum threshold, the monitor calls `deployAndSweepNative` or `deployAndSweepToken` automatically.

Alternatively, start monitors in `cmd/server/main.go`:

```go
go func() {
    if _, err := app.Monitor.Start(context.Background(), "bnbTestnet"); err != nil {
        log.Printf("[monitor] bnbTestnet: %v", err)
    }
}()
```

---

## Supported Networks

| Network key | Chain | Type |
|-------------|-------|------|
| `bnbMainnet`, `bnbTestnet`, `opBNBMainnet` | BNB Chain | EVM |
| `ethereumMainnet`, `ethereumSepoliaTestnet` | Ethereum | EVM |
| `polygonMainnet`, `polygonAmoyTestnet` | Polygon | EVM |
| `arbitrumMainnet`, `arbitrumSepoliaTestnet` | Arbitrum | EVM |
| `optimismMainnet`, `optimismSepoliaTestnet` | Optimism | EVM |
| `baseMainnet`, `baseSepoliaTestnet` | Base | EVM |
| `avalancheMainnet`, `avalancheFujiTestnet` | Avalanche | EVM |
| `tronMainnet`, `tronShasta` | Tron | TVM (gRPC, base58 `T...` addresses) |

### Tron-specific notes

- Deploy **`ForwarderFactoryTron`**, not the EVM factory. Use `POST /api/deploy` with `"completeSetup": true`.
- Tron uses gRPC endpoints (`grpc.trongrid.io:50051`), not HTTP JSON-RPC.
- Address prediction uses the TVM-correct `0x41` CREATE2 prefix via `TronClones`.
- Monitoring supports native **TRX** and **TRC-20** tokens.

> **Migration warning:** After upgrading to `ForwarderFactoryTron`, redeploy on each Tron network and update `TRON_*_FACTORY_ADDRESS`. Deposit addresses from the old factory are **not** compatible.

---

## Environment Variables

Per-network keys follow the pattern `{NETWORK_SUFFIX}_{KEY}`:

```bash
# Example: BNB Testnet
BNB_TESTNET_DEPLOYER_PRIVATE_KEY=0x...
BNB_TESTNET_DEPLOYER_ADDRESS=0x...
BNB_TESTNET_RELAYER_PRIVATE_KEY=0x...
BNB_TESTNET_RELAYER_ADDRESS=0x...
BNB_TESTNET_MOTHER_WALLET=0x...        # multisig recommended
BNB_TESTNET_FACTORY_ADDRESS=0x...      # set after deploy
BNB_TESTNET_RPC=https://...

# Global fallbacks (used when per-network keys are absent)
DEPLOYER_PRIVATE_KEY=0x...
RELAYER_PRIVATE_KEY=0x...
MOTHER_WALLET=0x...

# Block explorer API keys (for contract verification)
BSCSCAN_API_KEY=...
ETHERSCAN_API_KEY=...
# ... see .env.example for full list
```

See [`.env.example`](.env.example) for all RPC URLs, Tron gRPC endpoints, and explorer keys.

---

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Health check |
| `GET` | `/api/networks` | List supported networks |
| `GET` | `/api/networks/{name}/status` | Env configuration status |
| `GET` | `/api/wallets/generate?network=` | Generate deployer/relayer/mother wallets |
| `GET` | `/api/wallets/balance?network=&address=` | Check native balance |
| `POST` | `/api/deploy` | Deploy factory to a network |
| `POST` | `/api/deploy/compile` | Compile contracts (`forge build`) |
| `GET` | `/api/contracts/functions` | List callable factory functions |
| `GET` | `/api/contracts/{network}/info` | Read on-chain factory state |
| `POST` | `/api/contracts/call` | Call a factory function (read or write) |
| `GET` | `/api/monitor/wallets` | View registered wallets |
| `POST` | `/api/monitor/wallets` | Add/update wallets |
| `PUT` | `/api/monitor/wallets` | Replace all wallets for a network |
| `DELETE` | `/api/monitor/wallets` | Remove wallets by ID or address |
| `GET` | `/api/monitor/status` | Monitor status (all or `?network=`) |
| `GET` | `/api/monitor/addresses?network=` | Resolve watched addresses |
| `POST` | `/api/monitor/start` | Start deposit listener |
| `POST` | `/api/monitor/stop` | Stop deposit listener |

Full interactive docs at `/` (Swagger UI).

---

## Monitoring Details

The monitor maintains an in-memory registry of `walletId → depositAddress` pairs per network.

**Deposit detection:**
- **EVM** — subscribes to new blocks; checks native transfers and ERC-20 `Transfer` logs for watched addresses.
- **Tron** — polls via gRPC; detects TRX and TRC-20 transfers.

**Sweep logic:**
- Deposits below `minNativeBalance` or `minTokenBalance` are ignored (dust protection).
- Token deposits for tokens not listed in `setting.tokens` are ignored.
- On match, calls `deployAndSweepNative(userId)` or `deployAndSweepToken(userId, token)`.
- Recent sweep results (tx hashes, errors) are exposed via `/api/monitor/status`.

The registry is **in-memory** — wallet registrations are lost on server restart. Re-push wallets after restart, or integrate persistence in your deployment.

---

## Project Structure

```
forwarder-factory/
├── cmd/server/              # Go API entry point
├── internal/
│   ├── blockchain/          # EVM RPC client, Foundry integration
│   ├── contract/            # Factory ABI calls (read & write)
│   ├── deploy/              # Contract deployment & verification
│   ├── env/                 # .env loader
│   ├── factoryabi/          # Function definitions for API
│   ├── httpapi/             # Chi router, Swagger UI
│   ├── monitor/             # Deposit listener & auto-sweep
│   ├── network/             # Network definitions
│   ├── openapi/             # OpenAPI spec builder
│   ├── tron/                # Tron gRPC client, deploy, listener
│   └── wallet/              # Key generation, balance checks
├── contracts/
│   ├── Forwarder.sol        # Per-user minimal proxy
│   ├── ForwarderFactory.sol # EVM factory
│   ├── ForwarderFactoryTron.sol
│   └── tron/TronClones.sol  # TVM CREATE2 address prediction
├── test/                    # Forge tests (Solidity)
├── web/docs/                # Swagger UI assets
├── foundry.toml
├── go.mod
└── Makefile
```

---

## Commands

| Task | Command |
|------|---------|
| Compile contracts | `forge build` or `make compile` |
| Contract tests | `forge test` or `make test` |
| Run API (hot reload) | `make dev` |
| Run API | `make run` or `go run ./cmd/server` |
| Build server binary | `make build` |

Verbose tests: `forge test -vvv`

---

## License

MIT — see individual contract SPDX headers.
