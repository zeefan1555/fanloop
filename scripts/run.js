#!/usr/bin/env node

const fs = require("fs");
const os = require("os");
const path = require("path");
const { spawnSync } = require("child_process");
const { dataHome, describeInstall, install } = require("./install.js");
const launcherVersion = require("../package.json").version;

function installLauncher(env = process.env) {
  const args = ["install", "--global", "--install-links", "--ignore-scripts", "--no-audit", "--no-fund", path.resolve(__dirname, "..")];
  const command = env.npm_execpath ? process.execPath : "npm";
  if (env.npm_execpath) args.unshift(env.npm_execpath);
  const result = spawnSync(command, args, {
    encoding: "utf8",
    env,
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error(result.stderr.trim() || `persistent npm launcher install exited ${result.status}`);
  }
}

function exitWithChild(result) {
  if (result.error) {
    console.error(result.error.message);
    process.exit(1);
  }
  if (result.signal) process.kill(process.pid, result.signal);
  process.exit(result.status ?? 1);
}

if (process.argv[2] === "install") {
  try {
    installLauncher();
    console.log(describeInstall(install(process.env, launcherVersion)));
  } catch (error) {
    console.error(`Fanloop install failed: ${error.message}`);
    process.exit(1);
  }
} else if (process.argv.length === 3 && process.argv[2] === "update") {
  let result;
  try {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-update-"));
    try {
      result = spawnSync("npx", [
        "--yes",
        "--prefer-online",
        "--package=fanloop-cli@latest",
        "--",
        "fanloop",
        "install",
      ], {
        cwd: directory,
        env: { ...process.env, NPM_CONFIG_REGISTRY: "https://registry.npmjs.org", FANLOOP_UPDATE_FORWARD_ONLY: "1" },
        stdio: "inherit",
      });
    } finally {
      fs.rmSync(directory, { recursive: true, force: true });
    }
  } catch (error) {
    console.error(`Fanloop update failed: ${error.message}`);
    process.exit(1);
  }
  exitWithChild(result);
} else {
  const binary = path.join(dataHome(), "current", "bin", "fanloop");
  if (!fs.existsSync(binary)) {
    console.error("Fanloop is not installed. Run: npx --yes --prefer-online --package=fanloop-cli@latest -- fanloop install");
    process.exit(1);
  }
  const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
  exitWithChild(result);
}
