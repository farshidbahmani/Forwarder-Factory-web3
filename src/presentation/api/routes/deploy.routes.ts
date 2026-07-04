import { Router } from "express";
import { deployService } from "../../../application/container";

export const deployRouter = Router();

deployRouter.post("/compile", (_req, res, next) => {
  try {
    deployService.compile();
    res.json({ compiled: true });
  } catch (e) {
    next(e);
  }
});

deployRouter.post("/", async (req, res, next) => {
  try {
    const { network, verify = true } = req.body as { network: string; verify?: boolean };
    if (!network) {
      res.status(400).json({ error: "network is required" });
      return;
    }
    const result = await deployService.deploy(network, verify);
    res.json(result);
  } catch (e) {
    next(e);
  }
});
