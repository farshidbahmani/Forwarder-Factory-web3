const openApiSpecBase = {
  openapi: "3.0.3",
  info: {
    title: "Forwarder Factory API",
    version: "1.0.0",
    description:
      "REST API for multi-chain wallet generation, ForwarderFactory deployment, on-chain contract calls, and deposit monitoring.",
  },
  tags: [
    { name: "Health", description: "Service health" },
    { name: "Networks", description: "Supported blockchain networks" },
    { name: "Wallets", description: "Wallet generation and balance lookup" },
    { name: "Deploy", description: "Contract compile and deploy" },
    { name: "Contracts", description: "ForwarderFactory read/write calls" },
    { name: "Monitor", description: "Block monitoring and auto-sweep" },
  ],
  paths: {
    "/api/health": {
      get: {
        tags: ["Health"],
        operationId: "getHealth",
        summary: "Health check",
        responses: {
          "200": {
            description: "Service is running",
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  properties: { status: { type: "string", example: "ok" } },
                },
              },
            },
          },
        },
      },
    },
    "/api/networks": {
      get: {
        tags: ["Networks"],
        operationId: "listNetworks",
        summary: "List all supported networks",
        responses: {
          "200": {
            description: "Network list",
            content: { "application/json": { schema: { type: "array", items: { $ref: "#/components/schemas/Network" } } } },
          },
        },
      },
    },
    "/api/networks/{name}/status": {
      get: {
        tags: ["Networks"],
        operationId: "getNetworkStatus",
        summary: "Env config status for a network",
        parameters: [{ $ref: "#/components/parameters/networkName" }],
        responses: {
          "200": {
            description: "Config status",
            content: { "application/json": { schema: { $ref: "#/components/schemas/NetworkStatus" } } },
          },
        },
      },
    },
    "/api/wallets/generate": {
      get: {
        tags: ["Wallets"],
        operationId: "generateWallets",
        summary: "Generate wallets for one network",
        description: "Creates deployer, relayer, and mother wallets. Use ?format=env for plain-text .env copy.",
        parameters: [
          { $ref: "#/components/parameters/networkQuery" },
          {
            name: "format",
            in: "query",
            required: false,
            schema: { type: "string", enum: ["json", "env"], default: "json" },
            description: "Set to env to get ready-to-paste plain text",
          },
        ],
        responses: {
          "200": {
            description: "Generated wallets (JSON) or .env text (format=env)",
            content: {
              "application/json": { schema: { $ref: "#/components/schemas/GenerateWalletsResponse" } },
              "text/plain": { schema: { type: "string" } },
            },
          },
          "400": { $ref: "#/components/responses/BadRequest" },
        },
      },
    },
    "/api/wallets/balance": {
      get: {
        tags: ["Wallets"],
        operationId: "getWalletBalance",
        summary: "Get native token balance for an address",
        parameters: [
          { $ref: "#/components/parameters/networkQuery" },
          {
            name: "address",
            in: "query",
            required: true,
            schema: { type: "string", example: "0xDD281B850B8a32F2dca05f5058b6656d32C2998f" },
          },
        ],
        responses: {
          "200": {
            description: "Balance",
            content: { "application/json": { schema: { $ref: "#/components/schemas/WalletBalance" } } },
          },
          "400": { $ref: "#/components/responses/BadRequest" },
        },
      },
    },
    "/api/wallets/status": {
      get: {
        tags: ["Wallets"],
        operationId: "getWalletStatus",
        summary: "Env config status for one network",
        parameters: [{ $ref: "#/components/parameters/networkQuery" }],
        responses: {
          "200": {
            description: "Config status",
            content: { "application/json": { schema: { $ref: "#/components/schemas/NetworkStatus" } } },
          },
          "400": { $ref: "#/components/responses/BadRequest" },
        },
      },
    },
    "/api/deploy/compile": {
      post: {
        tags: ["Deploy"],
        operationId: "compileContracts",
        summary: "Compile contracts",
        responses: {
          "200": {
            description: "Compiled",
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  properties: { compiled: { type: "boolean", example: true } },
                },
              },
            },
          },
        },
      },
    },
    "/api/deploy": {
      post: {
        tags: ["Deploy"],
        operationId: "deployFactory",
        summary: "Deploy ForwarderFactory",
        requestBody: {
          required: true,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/DeployRequest" },
            },
          },
        },
        responses: {
          "200": {
            description: "Deploy result",
            content: { "application/json": { schema: { $ref: "#/components/schemas/DeployResult" } } },
          },
          "400": { $ref: "#/components/responses/BadRequest" },
        },
      },
    },
    "/api/contracts/functions": {
      get: {
        tags: ["Contracts"],
        operationId: "listContractFunctions",
        summary: "List callable ForwarderFactory functions",
        responses: {
          "200": {
            description: "Function definitions",
            content: { "application/json": { schema: { type: "array", items: { $ref: "#/components/schemas/ContractFunction" } } } },
          },
        },
      },
    },
    "/api/contracts/{network}/info": {
      get: {
        tags: ["Contracts"],
        operationId: "getFactoryInfo",
        summary: "Get deployed factory info",
        parameters: [{ $ref: "#/components/parameters/networkPath" }],
        responses: {
          "200": {
            description: "Factory info",
            content: { "application/json": { schema: { $ref: "#/components/schemas/FactoryInfo" } } },
          },
          "400": { $ref: "#/components/responses/BadRequest" },
        },
      },
    },
    "/api/contracts/call": {
      post: {
        tags: ["Contracts"],
        operationId: "callContractFunction",
        summary: "Call a ForwarderFactory function",
        requestBody: {
          required: true,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/ContractCallRequest" },
            },
          },
        },
        responses: {
          "200": {
            description: "Call result",
            content: { "application/json": { schema: { $ref: "#/components/schemas/ContractCallResult" } } },
          },
          "400": { $ref: "#/components/responses/BadRequest" },
        },
      },
    },
    "/api/monitor/wallets": {
      get: {
        tags: ["Monitor"],
        operationId: "listMonitoredWallets",
        summary: "List monitored user wallets",
        responses: {
          "200": {
            description: "Monitored wallets",
            content: {
              "application/json": {
                schema: { type: "array", items: { $ref: "#/components/schemas/MonitoredWallet" } },
              },
            },
          },
        },
      },
    },
    "/api/monitor/status": {
      get: {
        tags: ["Monitor"],
        operationId: "getMonitorStatus",
        summary: "Monitor status for all or one network",
        parameters: [
          {
            name: "network",
            in: "query",
            required: false,
            schema: { type: "string", example: "bscTestnet" },
          },
        ],
        responses: {
          "200": {
            description: "Monitor status",
            content: {
              "application/json": {
                schema: {
                  oneOf: [
                    { type: "array", items: { $ref: "#/components/schemas/NetworkMonitorStatus" } },
                    { $ref: "#/components/schemas/NetworkMonitorStatus" },
                  ],
                },
              },
            },
          },
        },
      },
    },
    "/api/monitor/addresses": {
      get: {
        tags: ["Monitor"],
        operationId: "getMonitoredAddresses",
        summary: "Resolve monitored wallet addresses on a network",
        parameters: [{ $ref: "#/components/parameters/networkQuery" }],
        responses: {
          "200": {
            description: "Resolved addresses",
            content: { "application/json": { schema: { $ref: "#/components/schemas/MonitoredAddressesResponse" } } },
          },
          "400": { $ref: "#/components/responses/BadRequest" },
        },
      },
    },
    "/api/monitor/start": {
      post: {
        tags: ["Monitor"],
        operationId: "startMonitor",
        summary: "Start block monitoring on one network",
        requestBody: {
          required: true,
          content: {
            "application/json": {
              schema: {
                type: "object",
                required: ["network"],
                properties: {
                  network: { type: "string", example: "bscTestnet" },
                },
              },
            },
          },
        },
        responses: {
          "200": {
            description: "Monitor started",
            content: { "application/json": { schema: { type: "object" } } },
          },
          "400": { $ref: "#/components/responses/BadRequest" },
        },
      },
    },
    "/api/monitor/stop": {
      post: {
        tags: ["Monitor"],
        operationId: "stopMonitor",
        summary: "Stop block monitoring on one network",
        requestBody: {
          required: true,
          content: {
            "application/json": {
              schema: {
                type: "object",
                required: ["network"],
                properties: {
                  network: { type: "string", example: "bscTestnet" },
                },
              },
            },
          },
        },
        responses: {
          "200": {
            description: "Monitor stopped",
            content: { "application/json": { schema: { type: "object" } } },
          },
        },
      },
    },
  },
  components: {
    parameters: {
      networkName: {
        name: "name",
        in: "path",
        required: true,
        schema: { type: "string", example: "bscTestnet" },
      },
      networkPath: {
        name: "network",
        in: "path",
        required: true,
        schema: { type: "string", example: "bscTestnet" },
      },
      networkQuery: {
        name: "network",
        in: "query",
        required: true,
        schema: { type: "string", example: "bscTestnet" },
      },
    },
    responses: {
      BadRequest: {
        description: "Bad request",
        content: {
          "application/json": {
            schema: { $ref: "#/components/schemas/Error" },
          },
        },
      },
    },
    schemas: {
      Error: {
        type: "object",
        properties: { error: { type: "string" } },
      },
      Network: {
        type: "object",
        properties: {
          name: { type: "string", example: "bscTestnet" },
          chainId: { type: "integer", example: 97 },
          symbol: { type: "string", example: "tBNB" },
          envSuffix: { type: "string", example: "BSC_TESTNET" },
          isTestnet: { type: "boolean", example: true },
        },
      },
      NetworkStatus: {
        type: "object",
        properties: {
          network: { type: "string" },
          deployerKey: { type: "boolean" },
          relayerKey: { type: "boolean" },
          relayerAddress: { type: "string", nullable: true },
          motherWallet: { type: "string", nullable: true },
          factoryAddress: { type: "string", nullable: true },
        },
      },
      GeneratedWallet: {
        type: "object",
        properties: {
          address: { type: "string" },
          privateKey: { type: "string" },
          mnemonic: { type: "string" },
        },
      },
      GenerateWalletsResponse: {
        type: "object",
        properties: {
          wallets: { $ref: "#/components/schemas/NetworkWallets" },
          snippet: {
            type: "object",
            properties: {
              network: { type: "string" },
              lines: {
                type: "array",
                items: { type: "string" },
                description: "One .env line per item — use ?format=env for ready-to-paste plain text",
              },
            },
          },
        },
      },
      NetworkWallets: {
        type: "object",
        properties: {
          network: { type: "string" },
          deployer: { $ref: "#/components/schemas/GeneratedWallet" },
          relayer: { $ref: "#/components/schemas/GeneratedWallet" },
          mother: { $ref: "#/components/schemas/GeneratedWallet" },
        },
      },
      WalletBalance: {
        type: "object",
        properties: {
          network: { type: "string" },
          chainId: { type: "integer" },
          symbol: { type: "string" },
          address: { type: "string" },
          balance: { type: "string", example: "1.5" },
        },
      },
      DeployRequest: {
        type: "object",
        required: ["network"],
        properties: {
          network: { type: "string", example: "bscTestnet" },
          verify: { type: "boolean", default: true },
        },
      },
      DeployResult: {
        type: "object",
        properties: {
          network: { type: "string" },
          factoryAddress: { type: "string" },
          implementationAddress: { type: "string" },
          deployerAddress: { type: "string" },
          deployerBalance: { type: "string" },
          symbol: { type: "string" },
          verified: { type: "boolean" },
          verificationMessage: { type: "string" },
          envKey: { type: "string" },
        },
      },
      ContractFunction: {
        type: "object",
        properties: {
          name: { type: "string" },
          label: { type: "string" },
          description: { type: "string" },
          type: { type: "string", enum: ["read", "write"] },
          role: { type: "string", enum: ["any", "owner", "relayer"] },
          inputs: { type: "array", items: { type: "object" } },
        },
      },
      FactoryInfo: {
        type: "object",
        properties: {
          factoryAddress: { type: "string" },
          motherWallet: { type: "string" },
          relayer: { type: "string" },
          owner: { type: "string" },
          implementation: { type: "string" },
        },
      },
      ContractCallRequest: {
        type: "object",
        required: ["network", "functionName"],
        properties: {
          network: { type: "string", example: "bscTestnet" },
          functionName: { type: "string", example: "getAddress" },
          args: {
            type: "object",
            additionalProperties: { type: "string" },
            example: { userId: "12345" },
          },
        },
      },
      ContractCallResult: {
        type: "object",
        properties: {
          functionName: { type: "string" },
          type: { type: "string", enum: ["read", "write"] },
          result: {},
          txHash: { type: "string" },
          blockNumber: { type: "integer" },
          gasUsed: { type: "string" },
        },
      },
      MonitoredWallet: {
        type: "object",
        properties: {
          userId: { type: "string", example: "1" },
          label: { type: "string", example: "user-1" },
        },
      },
      MonitoredAddressesResponse: {
        type: "object",
        properties: {
          network: { type: "string" },
          addresses: {
            type: "array",
            items: {
              type: "object",
              properties: {
                userId: { type: "string" },
                address: { type: "string" },
                label: { type: "string" },
              },
            },
          },
        },
      },
      NetworkMonitorStatus: {
        type: "object",
        properties: {
          network: { type: "string" },
          running: { type: "boolean" },
          lastBlock: { type: "integer" },
          watchedAddresses: { type: "array", items: { type: "object" } },
          recentSweeps: { type: "array", items: { type: "object" } },
        },
      },
    },
  },
} as const;

export function buildOpenApiSpec(serverUrl: string) {
  return {
    ...openApiSpecBase,
    servers: [{ url: serverUrl, description: "Current server" }],
  };
}

/** @deprecated use buildOpenApiSpec() — kept for tests/tools that import a static spec */
export const openApiSpec = buildOpenApiSpec("http://localhost:3000");
