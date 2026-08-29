const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { it } = require("node:test");

const launcher = path.join(__dirname, "fanloop-launcher.sh");

it("updates through the latest GitHub Release installer", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-launcher-update-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const bin = path.join(root, "bin");
  const installer = path.join(root, "installer.sh");
  const log = path.join(root, "gh.log");
  fs.mkdirSync(bin);
  fs.writeFileSync(installer, "#!/usr/bin/env bash\n[[ \"$FANLOOP_UPDATE_FORWARD_ONLY\" == 1 ]]\nprintf 'Fanloop 9.9.9 installed successfully\\n'\n", { mode: 0o755 });
  fs.writeFileSync(path.join(bin, "gh"), `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >"$FANLOOP_GH_TEST_LOG"
[[ "$1 $2 $3 $4 $5 $6 $7" == "release download -R zeefan1555/fanloop -p fanloop-install.sh -O" ]]
cp "$FANLOOP_GH_TEST_INSTALLER" "$8"
`, { mode: 0o755 });

  const result = spawnSync("bash", [launcher, "update"], {
    encoding: "utf8",
    env: {
      ...process.env,
      PATH: `${bin}:${process.env.PATH}`,
      FANLOOP_GH_TEST_INSTALLER: installer,
      FANLOOP_GH_TEST_LOG: log,
    },
  });

  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, "Fanloop 9.9.9 installed successfully\n");
  assert.match(fs.readFileSync(log, "utf8"), /^release download -R zeefan1555\/fanloop -p fanloop-install\.sh -O /);
});

it("executes the current installed payload for ordinary commands", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-launcher-payload-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const binary = path.join(root, "current", "bin", "fanloop");
  fs.mkdirSync(path.dirname(binary), { recursive: true });
  fs.writeFileSync(binary, "#!/bin/sh\nprintf '%s\\n' \"$*\"\n", { mode: 0o755 });

  const result = spawnSync("bash", [launcher, "version", "--json"], {
    encoding: "utf8", env: { ...process.env, FANLOOP_DATA_HOME: root },
  });

  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, "version --json\n");
});
