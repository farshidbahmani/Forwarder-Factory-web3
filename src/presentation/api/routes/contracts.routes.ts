import { Router } from "express";
import { contractService } from "../../../application/container";
import { AppError } from "../../../domain/errors";
import { NETWORKS } from "../../../domain/network";

export const contractsRouter = Router();

function parseNetworkParam(value: string): string {
  if (value.startsWith("{") && value.endsWith("}")) {
    const names = NETWORKS.map((n) => n.name).join(", ");
    throw new AppError(
      `Invalid network "${value}". Use a real network key, e.g. bscTestnet. Supported: ${names}`,
    );
  }
  return value;
}

contractsRouter.get("/functions", (_req, res) => {
  res.json(contractService.listFunctions());
});

contractsRouter.get("/:network/info", async (req, res, next) => {
  try {
    const network = parseNetworkParam(req.params.network);
    res.json(await contractService.getFactoryInfo(network));
  } catch (e) {
    next(e);
  }
});

contractsRouter.post("/call", async (req, res, next) => {
  try {
    const { network, functionName, args = {} } = req.body as {
      network: string;
      functionName: string;
      args?: Record<string, string>;
    };
    if (!network || !functionName) {
      res.status(400).json({ error: "network and functionName are required" });
      return;
    }
    const result = await contractService.call(network, functionName, args);
    res.json(result);
  } catch (e) {
    next(e);
  }
});
