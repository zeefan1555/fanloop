const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { it } = require("node:test");

const launcher = path.join(__dirname, "run.js");

it("updates through the latest official npm installer outside the caller project", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "commonloop-launcher-update-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const bin = path.join(root, "bin");
  const log = path.join(root, "npx.log");
  const caller = path.join(root, "same-package-project");
  fs.mkdirSync(bin);
  fs.mkdirSync(caller);
  fs.writeFileSync(path.join(caller, "package.json"), JSON.stringify({ name: "@zeefan1555/commonloop-cli" }));
  fs.writeFileSync(
    path.join(bin, "npx"),
    "#!/bin/sh\n[ ! -f package.json ] || { printf 'npx ran in caller project\\n' >&2; exit 9; }\nprintf '%s\\n' \"$COMMONLOOP_UPDATE_FORWARD_ONLY|$*\" >\"$COMMONLOOP_UPDATE_TEST_LOG\"\nprintf 'Commonloop 9.9.9 installed successfully\\n'\n",
    { mode: 0o755 },
  );

  const result = spawnSync(process.execPath, [launcher, "update"], {
    cwd: caller,
    encoding: "utf8",
    env: {
      ...process.env,
      PATH: `${bin}:${process.env.PATH}`,
      COMMONLOOP_DATA_HOME: path.join(root, "missing-installation"),
      COMMONLOOP_UPDATE_TEST_LOG: log,
    },
  });

  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, "Commonloop 9.9.9 installed successfully\n");
  assert.equal(result.stderr, "");
  assert.equal(
    fs.readFileSync(log, "utf8"),
    "1|--yes --prefer-online @zeefan1555/commonloop-cli@latest install\n",
  );
});

it("returns the latest installer failure", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "commonloop-launcher-update-failure-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const bin = path.join(root, "bin");
  fs.mkdirSync(bin);
  fs.writeFileSync(path.join(bin, "npx"), "#!/bin/sh\nprintf 'install failed\\n' >&2\nexit 7\n", { mode: 0o755 });

  const result = spawnSync(process.execPath, [launcher, "update"], {
    encoding: "utf8",
    env: {
      ...process.env,
      PATH: `${bin}:${process.env.PATH}`,
      COMMONLOOP_DATA_HOME: path.join(root, "missing-installation"),
    },
  });

  assert.equal(result.status, 7);
  assert.equal(result.stdout, "");
  assert.equal(result.stderr, "install failed\n");
});
