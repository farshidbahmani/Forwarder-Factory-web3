import { createApp } from "./app";
import { monitorService } from "../../application/container";

const PORT = Number(process.env.PORT) || 3000;

const app = createApp();
app.listen(PORT, () => {
  console.log(`Forwarder Factory web app running at http://localhost:${PORT}`);

  // هر شبکه را جداگانه start کنید:
  void monitorService.start("bscTestnet");
});
