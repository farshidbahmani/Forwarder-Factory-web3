import { Router } from "express";
import { monitorService } from "../../../application/container";

export const monitorRouter = Router();

monitorRouter.get("/wallets", (_req, res) => {
  res.json(monitorService.listMonitoredWallets());
});

monitorRouter.get("/status", (req, res, next) => {
  try {
    const network = req.query.network as string | undefined;
    if (network) {
      res.json(monitorService.getStatus(network));
      return;
    }
    res.json(monitorService.listRunning());
  } catch (e) {
    next(e);
  }
});

monitorRouter.get("/addresses", async (req, res, next) => {
  try {
    const network = req.query.network as string;
    if (!network) {
      res.status(400).json({ error: "network query param is required" });
      return;
    }
    const addresses = await monitorService.resolveAddresses(network);
    res.json({ network, addresses });
  } catch (e) {
    next(e);
  }
});

monitorRouter.post("/start", async (req, res, next) => {
  try {
    const { network } = req.body as { network?: string };
    if (!network) {
      res.status(400).json({ error: "network is required" });
      return;
    }
    const result = await monitorService.start(network);
    res.json(result);
  } catch (e) {
    next(e);
  }
});

monitorRouter.post("/stop", (req, res, next) => {
  try {
    const { network } = req.body as { network?: string };
    if (!network) {
      res.status(400).json({ error: "network is required" });
      return;
    }
    res.json(monitorService.stop(network));
  } catch (e) {
    next(e);
  }
});
