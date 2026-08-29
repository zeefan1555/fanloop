const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { it } = require("node:test");

const installer = path.join(__dirname, "install-github.sh");

it("downloads latest and exact GitHub Release assets and protects an unmanaged launcher", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-github-install-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const assets = path.join(root, "assets");
  const bin = path.join(root, "bin");
  const launchers = path.join(root, "launchers");
  const log = path.join(root, "gh.log");
  fs.mkdirSync(assets);
  fs.mkdirSync(bin);
  const platform = { darwin: "darwin", linux: "linux" }[process.platform];
  const architecture = { x64: "amd64", arm64: "arm64" }[process.arch];
  fs.writeFileSync(path.join(assets, "release.json"), "{}\n");
  fs.writeFileSync(path.join(assets, "fanloop-install.js"), "// fake\n");
  fs.copyFileSync(path.join(__dirname, "fanloop-launcher.sh"), path.join(assets, "fanloop-launcher.sh"));
  fs.writeFileSync(path.join(assets, `fanloop-1.2.3-${platform}-${architecture}.tar.xz`), "fake\n");
  fs.writeFileSync(path.join(bin, "node"), "#!/bin/sh\n[ -f \"$FANLOOP_RELEASE_MANIFEST\" ] && [ -f \"$FANLOOP_RELEASE_ARCHIVE\" ]\nprintf 'Fanloop 1.2.3 installed successfully\\n'\n", { mode: 0o755 });
  fs.writeFileSync(path.join(bin, "gh"), `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$FANLOOP_GH_TEST_LOG"
if [[ "$1 $2" == "auth status" ]]; then exit 0; fi
destination=""
while [[ "$#" -gt 0 ]]; do
  if [[ "$1" == "-D" ]]; then destination="$2"; break; fi
  shift
done
[[ -n "$destination" ]]
cp "$FANLOOP_GH_TEST_ASSETS"/* "$destination/"
`, { mode: 0o755 });

  const env = {
    ...process.env,
    PATH: `${bin}:${process.env.PATH}`,
    FANLOOP_BIN_DIR: launchers,
    FANLOOP_GH_TEST_ASSETS: assets,
    FANLOOP_GH_TEST_LOG: log,
  };
  const result = spawnSync("bash", [installer], { encoding: "utf8", env });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, "Fanloop 1.2.3 installed successfully\n");
  assert.match(fs.readFileSync(log, "utf8"), /release download -R zeefan1555\/fanloop .*fanloop-install\.js/);
  assert.ok(fs.statSync(path.join(launchers, "fanloop")).mode & 0o100);

  const exact = spawnSync("bash", [installer], {
    encoding: "utf8", env: { ...env, FANLOOP_RELEASE_TAG: "v1.2.3" },
  });
  assert.equal(exact.status, 0, exact.stderr);
  assert.match(fs.readFileSync(log, "utf8"), /release download v1\.2\.3 -R zeefan1555\/fanloop/);

  fs.writeFileSync(path.join(launchers, "fanloop"), "user command\n");
  const refused = spawnSync("bash", [installer], { encoding: "utf8", env });
  assert.notEqual(refused.status, 0);
  assert.match(refused.stderr, /refusing to replace unmanaged launcher/);
  assert.equal(fs.readFileSync(path.join(launchers, "fanloop"), "utf8"), "user command\n");
});
