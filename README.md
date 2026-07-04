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
go run ./cmd/server
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
| Run API server | `go run ./cmd/server` or `make run` |
| Build server binary | `make build` |

Deploy & verify via API: `POST /api/deploy` with `{ "network": "bscTestnet", "verify": true }`

---

## Supported Networks

`bscTestnet`, `bscMainnet`, `ethereumSepoliaTestnet`, `ethereumMainnet`, `polygonAmoyTestnet`, `polygonMainnet`, `arbitrumSepoliaTestnet`, `arbitrumMainnet`, `optimismSepoliaTestnet`, `optimismMainnet`, `baseSepoliaTestnet`, `baseMainnet`, `avalancheFujiTestnet`, `avalancheMainnet`, `opBNBMainnet`

---

## `.env` keys (per network)

```bash
DEPLOYER_PRIVATE_KEY_BSC_TESTNET=0x...
RELAYER_PRIVATE_KEY_BSC_TESTNET=0x...
RELAYER_ADDRESS_BSC_TESTNET=0x...
MOTHER_WALLET_BSC_TESTNET=0x...
FACTORY_ADDRESS_BSC_TESTNET=0x...
BSC_TESTNET_RPC=https://...
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
app.Monitor.Start(context.Background(), "bscTestnet")
```

Or `POST /api/monitor/start` with `{ "network": "bscTestnet" }`.
