const assert = require("node:assert/strict");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { execFileSync } = require("child_process");
const { describe, it } = require("node:test");

const { assertMatchedVersion, checksum, dataHome, describeInstall, keepsInstalledRelease, selectedAsset, validateArchivePath, validateArchive } = require("./install.js");

function installedAt(version) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-installed-"));
  fs.mkdirSync(path.join(root, "current"));
  fs.writeFileSync(path.join(root, "current", "release.json"), JSON.stringify({ release_version: version }));
  return root;
}

describe("Fanloop npm installer", () => {
  it("selects the exact current platform asset", () => {
    const platform = { darwin: "darwin", linux: "linux" }[process.platform];
    const arch = { x64: "amd64", arm64: "arm64" }[process.arch];
    const wanted = { os: platform, arch, file: "wanted.tar.xz" };
    assert.equal(selectedAsset({ assets: [{ os: "other", arch }, wanted] }), wanted);
  });

  it("computes the archive SHA-256 used by release.json", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-checksum-"));
    const file = path.join(directory, "asset.tar.xz");
    fs.writeFileSync(file, "fanloop\n");
    assert.equal(checksum(file), "sha256:81dd68c75e5c5059482343c06d0c7e3b5dd1d299d171ff60723af589573f5d81");
  });

  it("uses an explicit data root without reading a project directory", () => {
    assert.equal(dataHome({ FANLOOP_DATA_HOME: "/tmp/fanloop-data" }), "/tmp/fanloop-data");
  });

  it("stores matched releases under ~/.fanloop by default", () => {
    assert.equal(dataHome({}), path.join(os.homedir(), ".fanloop"));
  });

  it("keeps a newer installed Release when update resolves an older candidate", () => {
    const root = installedAt("0.1.66");
    const env = { FANLOOP_DATA_HOME: root, FANLOOP_UPDATE_FORWARD_ONLY: "1" };
    assert.equal(keepsInstalledRelease(env, "0.1.65"), "0.1.66");
    assert.equal(keepsInstalledRelease(env, "0.1.66"), "");
    assert.equal(keepsInstalledRelease(env, "0.1.67"), "");
  });

  it("reports a kept Release apart from a fresh install", () => {
    assert.equal(describeInstall({ version: "0.1.66", kept: true }), "Fanloop 0.1.66 is already up to date");
    assert.equal(describeInstall({ version: "0.1.66", kept: false }), "Fanloop 0.1.66 installed successfully");
  });

  it("installs an older Release when the caller is not update", () => {
    const root = installedAt("0.1.66");
    assert.equal(keepsInstalledRelease({ FANLOOP_DATA_HOME: root }, "0.1.65"), "");
  });

  it("installs when either side is missing or not stable semver", () => {
    const missing = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-absent-"));
    const forwardOnly = { FANLOOP_UPDATE_FORWARD_ONLY: "1" };
    assert.equal(keepsInstalledRelease({ ...forwardOnly, FANLOOP_DATA_HOME: missing }, "0.1.65"), "");
    assert.equal(keepsInstalledRelease({ ...forwardOnly, FANLOOP_DATA_HOME: installedAt("dev") }, "0.1.65"), "");
    assert.equal(keepsInstalledRelease({ ...forwardOnly, FANLOOP_DATA_HOME: installedAt("0.1.66") }, "0.2.0-rc.1"), "");
  });

  it("rejects a Release manifest that does not match the npm launcher", () => {
    assert.throws(
      () => assertMatchedVersion({ release_version: "1.2.4", cli: { version: "1.2.4" } }, "1.2.3"),
      /does not match npm launcher 1\.2\.3/,
    );
  });

  it("rejects links before extracting a Release archive", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "fanloop-archive-"));
    const source = path.join(directory, "source");
    fs.mkdirSync(path.join(source, "bin"), { recursive: true });
    fs.writeFileSync(path.join(source, "target"), "not a binary\n");
    fs.symlinkSync("../target", path.join(source, "bin", "fanloop"));
    const archive = path.join(directory, "release.tar.xz");
    execFileSync("tar", ["-cJf", archive, "-C", source, "."]);
    assert.throws(() => validateArchive(archive), /unsupported archive entry type/);
  });

  it("rejects an archive path outside the staging directory", () => {
    assert.throws(() => validateArchivePath("../outside"), /unsafe archive path/);
    assert.doesNotThrow(() => validateArchivePath("skills/fanloop-workflow/common/ai-test/SKILL.md"));
  });
});
