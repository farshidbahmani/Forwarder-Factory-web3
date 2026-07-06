# BNB Forwarder — Multi-Chain Deposit Wallet System

Smart contract deposit wallets with a Go backend API. Contracts are built with **Foundry**; the server is **Go**.

## Quick Start

```bash
# prerequisites: Go 1.22+, Foundry (forge/cast)
curl -L https://foundry.paradigm.xyz | bash && foundryup

forge install   # first clone only — installs lib/ deps
cp .env.example .env

forge build     # compile contracts
forge test      # run contract tests
make dev        # API with hot reload (or: go run ./cmd/server)
```

API: http://localhost:3000 — Swagger UI at `/`

---

## Project Structure

```
forwarder-factory/
├── cmd/server/           # Go API entry point
├── internal/             # Go services
├── contracts/            # Solidity
├── test/                 # Forge tests (Solidity)
├── lib/                  # forge-std, openzeppelin (submodules)
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
| Run API server | `make dev` (hot reload) or `go run ./cmd/server` |
| Build server binary | `make build` |

Deploy & verify via API: `POST /api/deploy` with `{ "network": "bnbTestnet", "verify": true }`

---

## Supported Networks

`bnbTestnet`, `bnbMainnet`, `ethereumSepoliaTestnet`, `ethereumMainnet`, `polygonAmoyTestnet`, `polygonMainnet`, `arbitrumSepoliaTestnet`, `arbitrumMainnet`, `optimismSepoliaTestnet`, `optimismMainnet`, `baseSepoliaTestnet`, `baseMainnet`, `avalancheFujiTestnet`, `avalancheMainnet`, `opBNBMainnet`, **`tronMainnet`**, **`tronShasta`**

Tron networks use gRPC (not HTTP JSON-RPC) and base58 addresses (`T...`). Deploy **`ForwarderFactoryTron`** (not the EVM `ForwarderFactory`) — it uses TVM-correct CREATE2 address prediction (`0x41` prefix). Monitoring supports native **TRX** and **TRC20** tokens.

> **Important:** After upgrading to `ForwarderFactoryTron`, redeploy on each Tron network (`POST /api/deploy`) and update `TRON_*_FACTORY_ADDRESS`. Deposit addresses from the old factory are **not** compatible; funds sent to old predicted addresses cannot be swept.

---

## `.env` keys (per network)

```bash
BNB_TESTNET_DEPLOYER_PRIVATE_KEY=0x...
BNB_TESTNET_DEPLOYER_ADDRESS=0x...
BNB_TESTNET_RELAYER_PRIVATE_KEY=0x...
BNB_TESTNET_RELAYER_ADDRESS=0x...
BNB_TESTNET_MOTHER_PRIVATE_KEY=0x...
BNB_TESTNET_MOTHER_WALLET=0x...
BNB_TESTNET_FACTORY_ADDRESS=0x...
BNB_TESTNET_RPC=https://...
BSCSCAN_API_KEY=...   # for verify on BSC
```

---

## API Endpoints

| Method | Path |
|--------|------|
| GET | `/api/health` |
| GET | `/api/networks` |
| GET | `/api/wallets/generate?network=` |
| POST | `/api/deploy` |
| POST | `/api/contracts/call` |
| POST | `/api/monitor/start` |
| … | see Swagger at `/` |

---

## Monitoring

In `cmd/server/main.go`:

```go
app.Monitor.Start(context.Background(), "bnbTestnet")
```

Or `POST /api/monitor/start` with `{ "network": "bnbTestnet" }`.
