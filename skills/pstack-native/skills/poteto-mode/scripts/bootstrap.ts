import { createHash } from "node:crypto";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  renameSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { homedir } from "node:os";
import { delimiter, isAbsolute, join } from "node:path";

const scriptsDirectory = import.meta.dir;

function currentInstallKey(): string {
  return createHash("sha256")
    .update(readFileSync(join(scriptsDirectory, "package.json")))
    .update("\0")
    .update(readFileSync(join(scriptsDirectory, "bun.lock")))
    .digest("hex");
}

export function ensureDependenciesInstalled(): void {
  const installKey = currentInstallKey();
  const cacheHome = process.env.XDG_CACHE_HOME;
  const cacheRoot = join(
    cacheHome && isAbsolute(cacheHome) ? cacheHome : join(homedir(), ".cache"),
    "poteto-mode-tools"
  );
  const installDirectory = join(cacheRoot, installKey);
  const nodeModulesDirectory = join(installDirectory, "node_modules");
  const ready = (directory: string): boolean =>
    existsSync(join(directory, "node_modules", "commander", "package.json")) &&
    existsSync(join(directory, ".install-key")) &&
    readFileSync(join(directory, ".install-key"), "utf8").trim() === installKey;

  if (!ready(installDirectory)) {
    mkdirSync(cacheRoot, { recursive: true });
    const temporary = mkdtempSync(join(cacheRoot, ".install-"));
    try {
      for (const name of ["package.json", "bun.lock"]) {
        copyFileSync(join(scriptsDirectory, name), join(temporary, name));
      }
      const result = Bun.spawnSync(
        [process.execPath, "install", "--production", "--frozen-lockfile"],
        { cwd: temporary }
      );
      if (result.exitCode !== 0) {
        process.stdout.write(result.stdout);
        process.stderr.write(result.stderr);
        throw new Error(`bun install exited with status ${result.exitCode}`);
      }
      writeFileSync(join(temporary, ".install-key"), `${installKey}\n`);
      if (!ready(temporary)) {
        throw new Error("bun install completed without installing commander");
      }
      // Publish only a complete installation; concurrent first runs may race.
      try {
        renameSync(temporary, installDirectory);
      } catch (error) {
        if (!ready(installDirectory)) throw error;
      }
    } finally {
      rmSync(temporary, { recursive: true, force: true });
    }
  }

  const modulePaths = (process.env.NODE_PATH ?? "").split(delimiter).filter(Boolean);
  if (modulePaths[0] === nodeModulesDirectory) return;

  const restarted = Bun.spawnSync([process.execPath, ...process.argv.slice(1)], {
    cwd: process.cwd(),
    env: { ...process.env, NODE_PATH: [nodeModulesDirectory, ...modulePaths].join(delimiter) },
    stdin: "inherit",
    stdout: "inherit",
    stderr: "inherit",
  });
  process.exit(restarted.exitCode ?? 1);
}
