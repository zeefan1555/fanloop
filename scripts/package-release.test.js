const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const { it } = require("node:test");

const { packageRelease } = require("./package-release.js");

it("packs every platform release archive into the npm package", (t) => {
  const dist = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-package-release-"));
  t.after(() => fs.rmSync(dist, { recursive: true, force: true }));
  const version = "1.2.3";
  const assets = [
    "fanloop-1.2.3-darwin-amd64.tar.xz",
    "fanloop-1.2.3-darwin-arm64.tar.xz",
    "fanloop-1.2.3-linux-amd64.tar.xz",
    "fanloop-1.2.3-linux-arm64.tar.xz",
  ];
  fs.writeFileSync(path.join(dist, "release.json"), JSON.stringify({
    release_version: version,
    cli: { version },
    assets: assets.map((file) => ({ file })),
  }));
  for (const file of assets) fs.writeFileSync(path.join(dist, file), file);

  const artifact = packageRelease(version, dist);
  const entries = execFileSync("tar", ["-tzf", artifact], { encoding: "utf8" }).trim().split("\n");
  for (const file of assets) assert.ok(entries.includes(`package/releases/${file}`), file);
  const metadata = JSON.parse(execFileSync("tar", ["-xOf", artifact, "package/package.json"], { encoding: "utf8" }));
  assert.equal(metadata.name, "@zeefan1555/fanloop-cli");
  assert.equal(metadata.publishConfig.registry, "https://npm.pkg.github.com");
  assert.equal(metadata.repository.url, "https://github.com/zeefan1555/fanloop.git");
});
