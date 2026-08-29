#!/usr/bin/env node
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const DEFAULT_REGISTRY = "https://registry.npmjs.org/";

function readJson(file) {
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

function writePackageVersion(packageJsonPath, version) {
  const metadata = readJson(packageJsonPath);
  metadata.version = version;
  fs.writeFileSync(packageJsonPath, `${JSON.stringify(metadata, null, 2)}\n`);
}

function parseStableSemver(version) {
  const match = /^(\d+)\.(\d+)\.(\d+)$/.exec(version);
  if (!match) {
    throw new Error(`auto-increment only supports stable semver versions, got ${version}`);
  }
  return {
    major: Number(match[1]),
    minor: Number(match[2]),
    patch: Number(match[3]),
  };
}

function latestPublishedPatch(packageName, major, minor, registry) {
  const result = spawnSync("npm", ["view", packageName, "versions", "--json", "--prefer-online"], {
    encoding: "utf8",
    env: {
      ...process.env,
      NPM_CONFIG_REGISTRY: registry,
    },
  });
  if (result.status !== 0) {
    const output = `${result.stdout || ""}\n${result.stderr || ""}`;
    if (/\bE404\b|not found|No match found/i.test(output)) {
      return -1;
    }
    throw new Error(`unable to list ${packageName} versions from ${registry}: ${output.trim()}`);
  }

  const parsed = JSON.parse(result.stdout);
  const versions = Array.isArray(parsed) ? parsed : [parsed];
  let highest = -1;
  for (const version of versions) {
    const match = /^(\d+)\.(\d+)\.(\d+)$/.exec(String(version));
    if (match && Number(match[1]) === major && Number(match[2]) === minor) {
      highest = Math.max(highest, Number(match[3]));
    }
  }
  return highest;
}

function selectReleaseVersion(packageJsonPath) {
  const metadata = readJson(packageJsonPath);
  if (!metadata.name) {
    throw new Error("package.json must include name");
  }
  if (!metadata.version) {
    throw new Error("package.json must include version");
  }

  const base = parseStableSemver(metadata.version);
  const registry = process.env.NPM_CONFIG_REGISTRY || metadata.publishConfig?.registry || DEFAULT_REGISTRY;
  const publishedPatch = latestPublishedPatch(metadata.name, base.major, base.minor, registry);
  return `${base.major}.${base.minor}.${Math.max(base.patch, publishedPatch + 1)}`;
}

function main(argv) {
  const [modeOrPath, maybePath, maybeVersion] = argv;
  if (modeOrPath === "--write") {
    if (!maybePath || !maybeVersion) {
      throw new Error("usage: resolve-release-version.js --write <package.json> <version>");
    }
    parseStableSemver(maybeVersion);
    writePackageVersion(path.resolve(maybePath), maybeVersion);
    return;
  }

  if (!modeOrPath || maybePath) {
    throw new Error("usage: resolve-release-version.js <package.json>");
  }
  process.stdout.write(`${selectReleaseVersion(path.resolve(modeOrPath))}\n`);
}

try {
  main(process.argv.slice(2));
} catch (error) {
  console.error(error.message);
  process.exit(1);
}
