import * as fs from "fs";
import * as path from "path";

export class EnvRepository {
  constructor(private readonly envPath = path.join(process.cwd(), ".env")) {}

  read(): Record<string, string> {
    if (!fs.existsSync(this.envPath)) return {};
    const content = fs.readFileSync(this.envPath, "utf8");
    const result: Record<string, string> = {};
    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;
      const eq = trimmed.indexOf("=");
      if (eq === -1) continue;
      const key = trimmed.slice(0, eq).trim();
      const value = trimmed.slice(eq + 1).trim();
      result[key] = value;
    }
    return result;
  }

  get(key: string): string | undefined {
    return process.env[key] ?? this.read()[key];
  }

  getForNetwork(baseKey: string, envSuffix: string): string | undefined {
    return this.get(`${baseKey}_${envSuffix}`) ?? this.get(baseKey);
  }

  append(lines: string[]): void {
    const block = [
      "",
      `# Appended ${new Date().toISOString()}`,
      ...lines,
      "",
    ].join("\n");
    fs.appendFileSync(this.envPath, block, "utf8");
    this.reload();
  }

  setMany(entries: Record<string, string>): void {
    const existing = this.read();
    const merged = { ...existing, ...entries };
    const lines = Object.entries(merged).map(([k, v]) => `${k}=${v}`);
    fs.writeFileSync(this.envPath, `${lines.join("\n")}\n`, "utf8");
    this.reload();
  }

  reload(): void {
    const parsed = this.read();
    for (const [key, value] of Object.entries(parsed)) {
      process.env[key] = value;
    }
  }
}
