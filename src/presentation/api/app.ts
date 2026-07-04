import express, { Request } from "express";
import cors from "cors";
import path from "path";
import swaggerUi from "swagger-ui-express";
import { buildOpenApiSpec } from "./openapi";
import { networksRouter, errorHandler } from "./routes/networks.routes";
import { walletsRouter } from "./routes/wallets.routes";
import { deployRouter } from "./routes/deploy.routes";
import { contractsRouter } from "./routes/contracts.routes";
import { monitorRouter } from "./routes/monitor.routes";

function requestOrigin(req: Request): string {
  const host = req.get("host");
  const proto = req.get("x-forwarded-proto") ?? req.protocol;
  return `${proto}://${host}`;
}

const swaggerUiOptions: swaggerUi.SwaggerUiOptions = {
  customSiteTitle: "Forwarder Factory API",
  customCssUrl: "/docs-assets/sidebar.css",
  customJs: "/docs-assets/sidebar.js",
  swaggerOptions: {
    url: "/api/openapi.json",
    deepLinking: true,
    docExpansion: "list",
    filter: false,
    tryItOutEnabled: true,
    tagsSorter: "alpha",
    validatorUrl: null,
  },
};

export function createApp() {
  const app = express();
  app.use(cors({ origin: true }));
  app.use(express.json());

  app.get("/api/health", (_req, res) => {
    res.json({ status: "ok" });
  });

  app.use("/api/networks", networksRouter);
  app.use("/api/wallets", walletsRouter);
  app.use("/api/deploy", deployRouter);
  app.use("/api/contracts", contractsRouter);
  app.use("/api/monitor", monitorRouter);

  app.get("/api/openapi.json", (req, res) => {
    res.json(buildOpenApiSpec(requestOrigin(req)));
  });

  app.use("/api", (_req, res) => {
    res.status(404).json({ error: "Not found" });
  });

  const docsPath = path.join(__dirname, "docs");
  app.use("/docs-assets", express.static(docsPath));

  app.use("/", swaggerUi.serve, swaggerUi.setup(undefined, swaggerUiOptions));

  app.use(errorHandler);
  return app;
}
