#!/usr/bin/env node

const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { execFileSync, spawnSync } = require("child_process");

const PLATFORM = { darwin: "darwin", linux: "linux" }[process.platform];
const ARCH = { x64: "amd64", arm64: "arm64" }[process.arch];

function dataHome(env = process.env) {
  if (env.COMMONLOOP_DATA_HOME) return path.resolve(env.COMMONLOOP_DATA_HOME);
  return path.join(os.homedir(), ".commonloop");
}

function stableSemver(value) {
  const match = /^(\d+)\.(\d+)\.(\d+)$/.exec(typeof value === "string" ? value : "");
  return match ? [Number(match[1]), Number(match[2]), Number(match[3])] : null;
}

function installedRelease(env) {
  try {
    const manifest = JSON.parse(fs.readFileSync(path.join(dataHome(env), "current", "release.json"), "utf8"));
    return typeof manifest.release_version === "string" ? manifest.release_version : "";
  } catch {
    return "";
  }
}

// `commonloop update` only moves forward; direct `commonloop install` stays unconditional so the
// documented rollback entry keeps working. Equal versions still install to preserve local repair.
function keepsInstalledRelease(env, candidateVersion) {
  if (!env.COMMONLOOP_UPDATE_FORWARD_ONLY) return "";
  const installed = installedRelease(env);
  const current = stableSemver(installed);
  const candidate = stableSemver(candidateVersion);
  if (!current || !candidate) return "";
  for (let index = 0; index < current.length; index += 1) {
    if (candidate[index] !== current[index]) return candidate[index] < current[index] ? installed : "";
  }
  return "";
}

function checksum(file) {
  const hash = crypto.createHash("sha256");
  const descriptor = fs.openSync(file, "r");
  try {
    const buffer = Buffer.alloc(64 * 1024);
    let count;
    while ((count = fs.readSync(descriptor, buffer, 0, buffer.length, null)) > 0) {
      hash.update(buffer.subarray(0, count));
    }
  } finally {
    fs.closeSync(descriptor);
  }
  return `sha256:${hash.digest("hex")}`;
}

function selectedAsset(manifest) {
  if (!PLATFORM || !ARCH) throw new Error(`unsupported platform ${process.platform}-${process.arch}`);
  const asset = manifest.assets.find((item) => item.os === PLATFORM && item.arch === ARCH);
  if (!asset) throw new Error(`release has no asset for ${PLATFORM}-${ARCH}`);
  return asset;
}

function assertMatchedVersion(manifest, launcherVersion) {
  if (manifest.release_version !== launcherVersion || manifest.cli?.version !== launcherVersion) {
    throw new Error(`Release ${manifest.release_version || "unknown"} does not match npm launcher ${launcherVersion}`);
  }
}

function validateArchivePath(listedName) {
  const name = listedName.replace(/^\.\//, "").replace(/\/$/, "");
  if (name && (name.includes("\\") || path.posix.isAbsolute(name) || path.posix.normalize(name) !== name || name === ".." || name.startsWith("../"))) {
    throw new Error(`unsafe archive path: ${listedName}`);
  }
  return name;
}

function validateArchive(archive) {
  const options = { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] };
  const names = execFileSync("tar", ["-tf", archive], options).split(/\r?\n/).filter(Boolean);
  const details = execFileSync("tar", ["-tvf", archive], options).split(/\r?\n/).filter(Boolean);
  if (names.length === 0 || names.length !== details.length) {
    throw new Error("archive listing is ambiguous or empty");
  }
  for (let index = 0; index < names.length; index += 1) {
    validateArchivePath(names[index]);
    if (!"-d".includes(details[index][0])) {
      throw new Error(`unsupported archive entry type: ${names[index]}`);
    }
  }
  if (!names.some((name) => name.replace(/^\.\//, "").replace(/\/$/, "") === "bin/commonloop")) {
    throw new Error("archive does not contain bin/commonloop");
  }
}

function install(env = process.env, launcherVersion = "") {
  const manifestPath = env.COMMONLOOP_RELEASE_MANIFEST || path.join(__dirname, "..", "release.json");
  const manifestContent = fs.readFileSync(manifestPath);
  const manifest = JSON.parse(manifestContent);
  if (launcherVersion) assertMatchedVersion(manifest, launcherVersion);
  const kept = keepsInstalledRelease(env, manifest.release_version);
  if (kept) return { version: kept, kept: true };
  const asset = selectedAsset(manifest);
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "commonloop-install-"));
  try {
    const archive = path.join(temporary, path.basename(asset.file));
    if (env.COMMONLOOP_RELEASE_ARCHIVE) {
      fs.copyFileSync(path.resolve(env.COMMONLOOP_RELEASE_ARCHIVE), archive);
    } else {
      if (path.basename(asset.file) !== asset.file) throw new Error(`unsafe release asset: ${asset.file}`);
      fs.copyFileSync(path.join(__dirname, "..", "releases", asset.file), archive);
    }
    const actual = checksum(archive);
    if (actual !== asset.sha256) {
      throw new Error(`archive checksum mismatch: expected ${asset.sha256}, got ${actual}`);
    }

    const staging = path.join(temporary, "release");
    fs.mkdirSync(staging);
    validateArchive(archive);
    execFileSync("tar", ["-xf", archive, "-C", staging], { stdio: ["ignore", "ignore", "pipe"] });
    fs.writeFileSync(path.join(staging, "release.json"), manifestContent);
    const binary = path.join(staging, "bin", "commonloop");
    fs.chmodSync(binary, 0o755);

    const root = dataHome(env);
    const codexSkills = path.resolve(env.COMMONLOOP_CODEX_SKILLS_ROOT || path.join(os.homedir(), ".codex", "skills"));
    const agentSkills = path.resolve(env.COMMONLOOP_AGENT_SKILLS_ROOT || path.join(os.homedir(), ".agents", "skills"));
    const traeSkills = path.resolve(env.COMMONLOOP_TRAE_SKILLS_ROOT || path.join(os.homedir(), ".trae", "skills"));
    const claudeSkills = path.resolve(env.COMMONLOOP_CLAUDE_SKILLS_ROOT || path.join(os.homedir(), ".claude", "skills"));
    const result = spawnSync(binary, [
      "__install", "--source", staging, "--data-root", root,
      "--codex-skills-root", codexSkills, "--agent-skills-root", agentSkills,
      "--trae-skills-root", traeSkills, "--claude-skills-root", claudeSkills, "--replace-invalid",
    ], { encoding: "utf8" });
    if (result.status !== 0) {
      throw new Error(result.stderr.trim() || `installer exited ${result.status}`);
    }
    return { version: manifest.release_version, kept: false };
  } finally {
    fs.rmSync(temporary, { recursive: true, force: true });
  }
}

function describeInstall(result) {
  return result.kept
    ? `Commonloop ${result.version} is already up to date`
    : `Commonloop ${result.version} installed successfully`;
}

if (require.main === module) {
  try {
    console.log(describeInstall(install()));
  } catch (error) {
    console.error(`Commonloop install failed: ${error.message}`);
    process.exit(1);
  }
}

module.exports = { assertMatchedVersion, checksum, dataHome, describeInstall, installedRelease, keepsInstalledRelease, selectedAsset, validateArchivePath, validateArchive, install };
