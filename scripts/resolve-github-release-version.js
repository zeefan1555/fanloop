#!/usr/bin/env node
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const REPOSITORY = "zeefan1555/fanloop";

function parseStableSemver(version) {
  const match = /^(\d+)\.(\d+)\.(\d+)$/.exec(version);
  if (!match) throw new Error(`auto-increment only supports stable semver versions, got ${version}`);
  return { major: Number(match[1]), minor: Number(match[2]), patch: Number(match[3]) };
}

function latestReleasePatch(major, minor) {
  const result = spawnSync("gh", [
    "release", "list", "--repo", REPOSITORY, "--limit", "1000", "--json", "tagName",
  ], { encoding: "utf8" });
  if (result.status !== 0) {
    const output = `${result.stdout || ""}\n${result.stderr || ""}`.trim();
    throw new Error(`unable to list GitHub Releases for ${REPOSITORY}: ${output}`);
  }
  const releases = JSON.parse(result.stdout);
  let highest = -1;
  for (const release of releases) {
    const match = /^v(\d+)\.(\d+)\.(\d+)$/.exec(String(release.tagName));
    if (match && Number(match[1]) === major && Number(match[2]) === minor) {
      highest = Math.max(highest, Number(match[3]));
    }
  }
  return highest;
}

function selectReleaseVersion(packageJsonPath) {
  const metadata = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));
  const base = parseStableSemver(metadata.version || "");
  const releasedPatch = latestReleasePatch(base.major, base.minor);
  return `${base.major}.${base.minor}.${Math.max(base.patch, releasedPatch + 1)}`;
}

try {
  if (process.argv.length !== 3) throw new Error("usage: resolve-github-release-version.js <package.json>");
  process.stdout.write(`${selectReleaseVersion(path.resolve(process.argv[2]))}\n`);
} catch (error) {
  console.error(error.message);
  process.exit(1);
}
