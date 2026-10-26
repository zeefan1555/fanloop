'use strict';

const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');
const {execFileSync} = require('child_process');
const test = require('node:test');
const {installRelease} = require('./scripts/install');
const {releaseAssets, skillRoots} = require('./test-fixtures');

let native;
function nativeBinary() {
  if (!native) {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-native-migration-'));
    const binary = path.join(root, 'fanloop');
    execFileSync('go', ['build', '-buildvcs=false', '-o', binary, '.'], {cwd: path.resolve(__dirname, '../..'), stdio: 'pipe'});
    native = fs.readFileSync(binary);
  }
  return native;
}

function managed(root) {
  const current = path.join(root, 'current');
  fs.mkdirSync(path.join(root, 'releases', '1.2.2'), {recursive: true});
  fs.mkdirSync(current);
  for (const [name, target] of Object.entries({bin: 'release/bin', entrypoints: 'release/entrypoints', 'release.json': 'release/release.json', release: '../releases/1.2.2'})) {
    fs.symlinkSync(target, path.join(current, name));
  }
  return fs.lstatSync(current).ino;
}

test('npm migrates managed current and retains its original inode and links', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-managed-'));
  const inode = managed(root);
  const {release} = releaseAssets('1.2.3', (stage) => {
    fs.writeFileSync(path.join(stage, 'bin', 'fanloop'), nativeBinary());
    const manifestPath = path.join(stage, 'release.json');
    const manifest = JSON.parse(fs.readFileSync(manifestPath));
    manifest.cli.binary_sha256 = require('./scripts/layout').fileDigest(path.join(stage, 'bin', 'fanloop'));
    fs.writeFileSync(manifestPath, JSON.stringify(manifest));
  }, {platform: {os: process.platform, arch: process.arch === 'arm64' ? 'arm64' : 'amd64'}});
  await installRelease({version: '1.2.3', platform: {os: process.platform, arch: process.arch === 'arm64' ? 'arm64' : 'amd64'}, dataHome: root, releaseDir: release, skillRoots: skillRoots(root), env: {FANLOOP_NO_AUTO_CONFIG: '1'}});
  assert.equal(fs.readlinkSync(path.join(root, 'current')), 'releases/1.2.3');
  const backups = fs.readdirSync(root).filter((name) => name.startsWith('current.backup-'));
  assert.equal(backups.length, 1);
  assert.equal(fs.lstatSync(path.join(root, backups[0])).ino, inode);
  assert.equal(fs.readlinkSync(path.join(root, backups[0], 'release')), '../releases/1.2.2');
});

test('npm refuses invalid managed current and preserves it on upgrade failure', async (t) => {
  for (const variant of ['unknown', 'escape', 'archive', 'skill', 'activation']) {
    await t.test(variant, async () => {
      const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-managed-failure-'));
      const inode = managed(root);
      if (variant === 'unknown') fs.writeFileSync(path.join(root, 'current', 'user-data'), 'keep');
      if (variant === 'escape') {
        fs.unlinkSync(path.join(root, 'current', 'release'));
        fs.symlinkSync('../../outside', path.join(root, 'current', 'release'));
      }
      const {release, archiveName} = releaseAssets('1.2.3');
      if (variant === 'archive') fs.appendFileSync(path.join(release, archiveName), 'corrupt');
      const roots = skillRoots(root);
      if (variant === 'skill') fs.writeFileSync(path.join(roots[0], 'fanloop-workflow'), 'user-data');
      const expected = variant === 'archive' ? /release validation/ : variant === 'skill' ? /non-Fanloop/ : variant === 'activation' ? /activation/ : /current/;
      await assert.rejects(() => installRelease({version: '1.2.3', platform: {os: 'linux', arch: 'amd64'}, dataHome: root, releaseDir: release, skillRoots: roots, env: {FANLOOP_NO_AUTO_CONFIG: '1'}}), expected);
      assert.equal(fs.lstatSync(path.join(root, 'current')).ino, inode);
      assert.equal(fs.lstatSync(path.join(root, 'current')).isDirectory(), true);
      assert.equal(fs.readdirSync(root).some((name) => name.startsWith('current.backup-')), false);
    });
  }
});

test('same-version npm repair restores active bytes and preserves user files after Skill failure', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-repair-failure-'));
  const inode = managed(root);
  const old = path.join(root, 'releases', '1.2.2');
  require('./test-fixtures').writeValidTree(old, '1.2.2');
  fs.writeFileSync(path.join(old, 'user-marker'), 'keep');
  const binaryBefore = fs.readFileSync(path.join(old, 'bin/fanloop'));
  const releaseInode = fs.lstatSync(old).ino;
  const {release} = releaseAssets('1.2.2', (stage) => {
    fs.writeFileSync(path.join(stage, 'bin/fanloop'), nativeBinary());
    const file = path.join(stage, 'release.json');
    const manifest = JSON.parse(fs.readFileSync(file));
    manifest.cli.binary_sha256 = require('./scripts/layout').fileDigest(path.join(stage, 'bin/fanloop'));
    fs.writeFileSync(file, JSON.stringify(manifest));
  });
  const roots = skillRoots(root);
  fs.writeFileSync(path.join(roots[0], 'fanloop-workflow'), 'user Skill');
  await assert.rejects(() => installRelease({version: '1.2.2', platform: {os: 'linux', arch: 'amd64'}, dataHome: root, releaseDir: release, skillRoots: roots, env: {FANLOOP_NO_AUTO_CONFIG: '1'}}), /non-Fanloop/);
  assert.equal(fs.lstatSync(path.join(root, 'current')).ino, inode);
  assert.equal(fs.lstatSync(old).ino, releaseInode);
  assert.deepEqual(fs.readFileSync(path.join(old, 'bin/fanloop')), binaryBefore);
  assert.equal(fs.readFileSync(path.join(old, 'user-marker'), 'utf8'), 'keep');
});
