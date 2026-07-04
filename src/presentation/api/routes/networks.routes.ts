import { Router, Request, Response, NextFunction } from "express";
import { walletService } from "../../../application/container";
import { AppError } from "../../../domain/errors";

export const networksRouter = Router();

networksRouter.get("/", (_req, res) => {
  res.json(walletService.listNetworks());
});

networksRouter.get("/:name/status", (req, res) => {
  res.json(walletService.getEnvStatus(req.params.name));
});

export function errorHandler(err: unknown, _req: Request, res: Response, _next: NextFunction) {
  if (err instanceof AppError) {
    res.status(err.statusCode).json({ error: err.message });
    return;
  }
  const message = err instanceof Error ? err.message : "Internal server error";
  console.error(err);
  res.status(500).json({ error: message });
}
