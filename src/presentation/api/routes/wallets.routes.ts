import { Router } from "express";
import { walletService } from "../../../application/container";
import { AppError } from "../../../domain/errors";

export const walletsRouter = Router();

function requireQueryParam(value: unknown, name: string): string {
  if (!value || typeof value !== "string") {
    throw new AppError(`Query param ?${name}= is required`);
  }
  return value;
}

walletsRouter.get("/generate", (req, res, next) => {
  try {
    const network = requireQueryParam(req.query.network, "network");
    const wallets = walletService.generateForNetwork(network);
    const snippet = walletService.toEnvSnippet(wallets);

    if (req.query.format === "env") {
      res.type("text/plain; charset=utf-8").send(walletService.toEnvText(wallets));
      return;
    }

    res.json({ wallets, snippet });
  } catch (e) {
    next(e);
  }
});

walletsRouter.get("/balance", async (req, res, next) => {
  try {
    const network = requireQueryParam(req.query.network, "network");
    const address = requireQueryParam(req.query.address, "address");
    res.json(await walletService.checkBalance(network, address));
  } catch (e) {
    next(e);
  }
});

walletsRouter.get("/status", (req, res, next) => {
  try {
    const network = requireQueryParam(req.query.network, "network");
    res.json(walletService.getEnvStatus(network));
  } catch (e) {
    next(e);
  }
});
