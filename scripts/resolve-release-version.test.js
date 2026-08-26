const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { execFileSync, spawnSync } = require("node:child_process");
const { it } = require("node:test");

const repository = path.resolve(__dirname, "..");
const resolver = path.join(repository, "scripts", "resolve-release-version.js");

function makeTempPackage(version = "1.2.3") {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-release-version-"));
  const packageJson = path.join(root, "package.json");
  fs.writeFileSync(
    packageJson,
    `${JSON.stringify(
      {
        name: "fanloop-cli",
        version,
        publishConfig: { registry: "https://registry.npmjs.org" },
      },
      null,
      2,
    )}\n`,
  );
  return { root, packageJson };
}

function fakeNpm(existingVersions) {
  const bin = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-fake-npm-"));
  const npmPath = path.join(bin, "npm");
  fs.writeFileSync(
    npmPath,
    `#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" != "view" ]]; then
  echo "unexpected npm command: $*" >&2
  exit 2
fi
if [[ "$3" == "versions" && "$4" == "--json" && "$5" == "--prefer-online" ]]; then
  echo '${JSON.stringify(existingVersions)}'
  exit 0
fi
echo "unexpected npm view: $*" >&2
exit 2
`,
  );
  fs.chmodSync(npmPath, 0o755);
  return bin;
}

it("selects the first unpublished patch version without changing package.json", () => {
  const { packageJson } = makeTempPackage("1.2.3");
  const bin = fakeNpm(["1.2.3", "1.2.4"]);
  const output = execFileSync(process.execPath, [resolver, packageJson], {
    encoding: "utf8",
    env: { ...process.env, PATH: `${bin}:${process.env.PATH}` },
  });

  assert.equal(output.trim(), "1.2.5");
  assert.equal(JSON.parse(fs.readFileSync(packageJson, "utf8")).version, "1.2.3");
});

it("seeds from the latest published patch when it outpaces the frozen baseline", () => {
  const { packageJson } = makeTempPackage("1.2.3");
  const bin = fakeNpm(["1.2.3", "1.2.4", "1.2.5", "1.2.6", "1.2.7", "1.2.8", "1.2.9"]);
  const output = execFileSync(process.execPath, [resolver, packageJson], {
    encoding: "utf8",
    env: { ...process.env, PATH: `${bin}:${process.env.PATH}` },
  });

  assert.equal(output.trim(), "1.2.10");
  assert.equal(JSON.parse(fs.readFileSync(packageJson, "utf8")).version, "1.2.3");
});

it("writes the resolved version when requested", () => {
  const { packageJson } = makeTempPackage("1.2.3");

  execFileSync(process.execPath, [resolver, "--write", packageJson, "1.2.5"]);

  assert.equal(JSON.parse(fs.readFileSync(packageJson, "utf8")).version, "1.2.5");
});

it("fails closed when npm returns an unknown registry error", () => {
  const { packageJson } = makeTempPackage("1.2.3");
  const bin = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-fake-npm-error-"));
  const npmPath = path.join(bin, "npm");
  fs.writeFileSync(
    npmPath,
    `#!/usr/bin/env bash
echo "registry temporarily unavailable" >&2
exit 1
`,
  );
  fs.chmodSync(npmPath, 0o755);

  const result = spawnSync(process.execPath, [resolver, packageJson], {
    encoding: "utf8",
    env: { ...process.env, PATH: `${bin}:${process.env.PATH}` },
  });

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /unable to list/);
});
