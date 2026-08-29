const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { execFileSync, spawnSync } = require("node:child_process");
const { it } = require("node:test");

const resolver = path.join(__dirname, "resolve-github-release-version.js");

function fixture(version, releases) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-release-version-"));
  const bin = path.join(root, "bin");
  const packageJson = path.join(root, "package.json");
  fs.mkdirSync(bin);
  fs.writeFileSync(packageJson, JSON.stringify({ version }));
  fs.writeFileSync(path.join(bin, "gh"), `#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "release list --repo zeefan1555/fanloop --limit 1000 --json tagName" ]]
printf '%s\n' '${JSON.stringify(releases.map((tagName) => ({ tagName })))}'
`, { mode: 0o755 });
  return { root, packageJson, path: `${bin}:${process.env.PATH}` };
}

it("uses the frozen baseline when GitHub has no Release", (t) => {
  const value = fixture("1.2.3", []);
  t.after(() => fs.rmSync(value.root, { recursive: true, force: true }));
  const output = execFileSync(process.execPath, [resolver, value.packageJson], {
    encoding: "utf8", env: { ...process.env, PATH: value.path },
  });
  assert.equal(output.trim(), "1.2.3");
});

it("selects the first unused patch across draft and published Releases", (t) => {
  const value = fixture("1.2.3", ["v1.2.3", "v1.2.9", "v2.0.0", "invalid"]);
  t.after(() => fs.rmSync(value.root, { recursive: true, force: true }));
  const output = execFileSync(process.execPath, [resolver, value.packageJson], {
    encoding: "utf8", env: { ...process.env, PATH: value.path },
  });
  assert.equal(output.trim(), "1.2.10");
});

it("fails closed when GitHub cannot list Releases", (t) => {
  const value = fixture("1.2.3", []);
  t.after(() => fs.rmSync(value.root, { recursive: true, force: true }));
  fs.writeFileSync(path.join(value.root, "bin", "gh"), "#!/bin/sh\necho unavailable >&2\nexit 1\n", { mode: 0o755 });
  const result = spawnSync(process.execPath, [resolver, value.packageJson], {
    encoding: "utf8", env: { ...process.env, PATH: value.path },
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /unable to list GitHub Releases.*unavailable/);
});
