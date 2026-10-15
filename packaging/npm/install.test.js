'use strict';

const assert = require('node:assert/strict');
const {execFileSync, spawnSync} = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const zlib = require('zlib');
const os = require('os');
const path = require('path');
const test = require('node:test');
const http = require('http');
const layout = require('./scripts/layout');
const {download, installRelease} = require('./scripts/install');

function writeValidTree(stage, version, bodies = {}) {
  fs.mkdirSync(path.join(stage, 'bin'), {recursive: true});
  fs.mkdirSync(path.join(stage, 'entrypoints', 'fanloop-workflow'), {recursive: true});
  fs.mkdirSync(path.join(stage, 'entrypoints', 'create-fanloop-workflow'), {recursive: true});
  fs.writeFileSync(path.join(stage, 'bin', 'fanloop'), bodies.binary || `#!/bin/sh\necho ${version}\n`, {mode: 0o755});
  fs.writeFileSync(path.join(stage, 'entrypoints', 'fanloop-workflow', 'SKILL.md'), bodies.workflow || '# workflow\n');
  fs.writeFileSync(path.join(stage, 'entrypoints', 'create-fanloop-workflow', 'SKILL.md'), bodies.author || '# author\n');
  const manifest = {
    schema_version: 4,
    release_version: version,
    cli: {version, binary_sha256: layout.fileDigest(path.join(stage, 'bin', 'fanloop'))},
    state_schema: {read_versions: [14], write_version: 14},
    skills: [
      {
        name: 'fanloop-workflow',
        version,
        path: 'entrypoints/fanloop-workflow',
        sha256: layout.directoryDigest(path.join(stage, 'entrypoints', 'fanloop-workflow')),
      },
      {
        name: 'create-fanloop-workflow',
        version,
        path: 'entrypoints/create-fanloop-workflow',
        sha256: layout.directoryDigest(path.join(stage, 'entrypoints', 'create-fanloop-workflow')),
      },
    ],
  };
  fs.writeFileSync(path.join(stage, 'release.json'), `${JSON.stringify(manifest)}\n`);
}

function skillRoots(parent) {
  return ['codex', 'agents', 'trae', 'claude'].map((name) => {
    const dir = path.join(parent, 'skills', name);
    fs.mkdirSync(dir, {recursive: true});
    return dir;
  });
}

function packStage(stage, archivePath, format) {
  const members = ['-C', stage, 'bin', 'entrypoints', 'release.json'];
  if (!format) {
    execFileSync('tar', ['-czf', archivePath, ...members]);
    return;
  }
  try {
    execFileSync('tar', [`--format=${format}`, '-czf', archivePath, ...members], {stdio: 'pipe'});
  } catch (err) {
    execFileSync('tar', ['--format', format, '-czf', archivePath, ...members]);
  }
}

function releaseAssets(version, prepare, options = {}) {
  const release = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-assets-'));
  const stage = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-stage-'));
  writeValidTree(stage, version);
  if (prepare) {
    prepare(stage);
  }
  const archiveName = `fanloop-${version}-linux-amd64.tar.gz`;
  packStage(stage, path.join(release, archiveName), options.format);
  const archiveBytes = fs.readFileSync(path.join(release, archiveName));
  const archiveHash = crypto.createHash('sha256').update(archiveBytes).digest('hex');
  const manifest = {
    schema_version: 1,
    version,
    tag: `v${version}`,
    repository: 'zeefan1555/fanloop',
    assets: [{name: archiveName, os: 'linux', arch: 'amd64', sha256: archiveHash}],
  };
  fs.writeFileSync(path.join(release, 'release.json'), `${JSON.stringify(manifest)}\n`);
  const manifestHash = crypto.createHash('sha256').update(fs.readFileSync(path.join(release, 'release.json'))).digest('hex');
  fs.writeFileSync(path.join(release, 'checksums.txt'), `${archiveHash}  ${archiveName}\n${manifestHash}  release.json\n`);
  return {release, archiveName};
}

test('selects darwin and linux amd64/arm64 only', () => {
  assert.deepEqual(layout.selectPlatform('linux', 'x64'), {os: 'linux', arch: 'amd64'});
  assert.deepEqual(layout.selectPlatform('linux', 'amd64'), {os: 'linux', arch: 'amd64'});
  assert.deepEqual(layout.selectPlatform('linux', 'arm64'), {os: 'linux', arch: 'arm64'});
  assert.deepEqual(layout.selectPlatform('darwin', 'x64'), {os: 'darwin', arch: 'amd64'});
  assert.deepEqual(layout.selectPlatform('darwin', 'arm64'), {os: 'darwin', arch: 'arm64'});
  assert.equal(layout.assetName('1.2.3', {os: 'linux', arch: 'amd64'}), 'fanloop-1.2.3-linux-amd64.tar.gz');
  for (const [platform, arch] of [['windows', 'x64'], ['linux', '386'], ['freebsd', 'arm64']]) {
    assert.throws(() => layout.selectPlatform(platform, arch), /unsupported platform/);
  }
});

test('rejects a checksum mismatch and a missing entry', () => {
  const body = Buffer.from('fanloop');
  const good = crypto.createHash('sha256').update(body).digest('hex');
  layout.verifyChecksum(`${good}  fanloop-1.2.3-linux-amd64.tar.gz\n`, 'fanloop-1.2.3-linux-amd64.tar.gz', body);
  const wrong = `${'0'.repeat(64)}  fanloop-1.2.3-linux-amd64.tar.gz\n`;
  assert.throws(() => layout.verifyChecksum(wrong, 'fanloop-1.2.3-linux-amd64.tar.gz', body), /sha256 mismatch/);
  assert.throws(() => layout.verifyChecksum(`${good}  other.tar.gz\n`, 'fanloop-1.2.3-linux-amd64.tar.gz', body), /no entry/);
  assert.throws(() => layout.parseChecksums('not-a-checksum\n'), /invalid checksums.txt/);
});

test('switches current atomically and leaves the previous release in place', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-switch-'));
  const install = (version) => {
    const dir = path.join(root, 'releases', version, 'bin');
    fs.mkdirSync(dir, {recursive: true});
    fs.writeFileSync(path.join(dir, 'fanloop'), `#!/bin/sh\necho ${version}\n`, {mode: 0o755});
    assert.equal(layout.switchCurrent(root, version), true);
    assert.equal(fs.readlinkSync(path.join(root, 'current')), path.join('releases', version));
  };
  install('1.2.3');
  install('1.2.4');
  assert.equal(fs.readlinkSync(path.join(root, 'current')), path.join('releases', '1.2.4'));
  assert.equal(fs.readFileSync(path.join(root, 'releases', '1.2.3', 'bin', 'fanloop'), 'utf8').includes('1.2.3'), true);
  assert.equal(layout.switchCurrent(root, '1.2.4'), false);
  fs.mkdirSync(path.join(root, 'releases', '9.9.9', 'bin'), {recursive: true});
  fs.rmSync(path.join(root, 'current'));
  fs.mkdirSync(path.join(root, 'current'));
  assert.throws(() => layout.switchCurrent(root, '9.9.9'), /not a symlink/);
  assert.equal(fs.lstatSync(path.join(root, 'current')).isDirectory(), true);
});

test('install verifies checksums before switching current', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-install-'));
  const version = '1.2.3';
  const {release, archiveName} = releaseAssets(version);
  const roots = skillRoots(root);
  const manifestHash = crypto.createHash('sha256').update(fs.readFileSync(path.join(release, 'release.json'))).digest('hex');
  const bad = path.join(release, 'checksums-bad.txt');
  fs.writeFileSync(bad, `${'ab'.repeat(32)}  ${archiveName}\n${manifestHash}  release.json\n`);
  await assert.rejects(
    () => installRelease({
      version,
      dataHome: root,
      platform: {os: 'linux', arch: 'amd64'},
      skillRoots: roots,
      readFile: async (name) => fs.promises.readFile(name === 'checksums.txt' ? bad : path.join(release, name)),
    }),
    /sha256 mismatch/,
  );
  assert.equal(fs.existsSync(path.join(root, 'current')), false);

  await installRelease({
    version,
    dataHome: root,
    platform: {os: 'linux', arch: 'amd64'},
    releaseDir: release,
    skillRoots: roots,
  });
  assert.equal(fs.readlinkSync(path.join(root, 'current')), path.join('releases', version));
  assert.equal(fs.readFileSync(path.join(root, 'current', 'bin', 'fanloop'), 'utf8').includes(version), true);
  for (const skillRoot of roots) {
    assert.equal(fs.readlinkSync(path.join(skillRoot, 'fanloop-workflow')), path.join(root, 'current', 'entrypoints', 'fanloop-workflow'));
    assert.equal(
      fs.readlinkSync(path.join(skillRoot, 'create-fanloop-workflow')),
      path.join(root, 'current', 'entrypoints', 'create-fanloop-workflow'),
    );
  }
});

test('rejects a relative FANLOOP_DATA_HOME before resolving it', () => {
  assert.throws(() => layout.dataHome({FANLOOP_DATA_HOME: 'relative/fanloop'}), /absolute/);
  const absolute = path.join(os.tmpdir(), 'fanloop-absolute');
  assert.equal(layout.dataHome({FANLOOP_DATA_HOME: absolute}), absolute);
});

test('does not switch current when the internal manifest or skill links are incomplete', async () => {
  const version = '1.2.3';
  const incomplete = releaseAssets(version, (stage) => {
    fs.writeFileSync(path.join(stage, 'release.json'), '{"schema_version":4,"release_version":"1.2.3"}\n');
  });
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-incomplete-'));
  await assert.rejects(
    () => installRelease({
      version,
      dataHome: root,
      platform: {os: 'linux', arch: 'amd64'},
      releaseDir: incomplete.release,
      skillRoots: skillRoots(root),
    }),
    /release\.json|binary checksum|schema/,
  );
  assert.equal(fs.existsSync(path.join(root, 'current')), false);
  assert.equal(fs.existsSync(path.join(root, 'releases', version)), false);

  const blocked = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-blocked-'));
  const roots = skillRoots(blocked);
  fs.writeFileSync(path.join(roots[0], 'fanloop-workflow'), 'user owned\n');
  const valid = releaseAssets(version);
  await assert.rejects(
    () => installRelease({
      version,
      dataHome: blocked,
      platform: {os: 'linux', arch: 'amd64'},
      releaseDir: valid.release,
      skillRoots: roots,
    }),
    /non-Fanloop path/,
  );
  assert.equal(fs.existsSync(path.join(blocked, 'current')), false);
  assert.equal(fs.readFileSync(path.join(roots[0], 'fanloop-workflow'), 'utf8'), 'user owned\n');
});

test('repairs tampered release contents when the same version is installed again', async () => {
  const version = '1.2.3';
  const {release} = releaseAssets(version);
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-repair-'));
  const roots = skillRoots(root);
  const options = {
    version,
    dataHome: root,
    platform: {os: 'linux', arch: 'amd64'},
    releaseDir: release,
    skillRoots: roots,
  };
  await installRelease(options);
  const skill = path.join(root, 'releases', version, 'entrypoints', 'fanloop-workflow', 'SKILL.md');
  fs.writeFileSync(skill, 'tampered\n');
  fs.writeFileSync(path.join(root, 'releases', version, 'extra.txt'), 'extra\n');
  await installRelease(options);
  assert.equal(fs.readFileSync(skill, 'utf8'), '# workflow\n');
  assert.equal(fs.existsSync(path.join(root, 'releases', version, 'extra.txt')), false);
  assert.equal(fs.readlinkSync(path.join(root, 'current')), path.join('releases', version));
});

function assertNoArchiveMetadata(root) {
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, {withFileTypes: true})) {
      assert.equal(entry.name.startsWith('._'), false, entry.name);
      assert.equal(entry.name === 'PaxHeader' || entry.name === 'PaxHeaders' || entry.name.startsWith('PaxHeaders.'), false, entry.name);
      if (entry.isDirectory()) {
        walk(path.join(dir, entry.name));
      }
    }
  };
  walk(root);
}

test('installs a pax archive and strips AppleDouble metadata', async () => {
  const version = '1.2.3';
  const {release} = releaseAssets(version, (stage) => {
    fs.writeFileSync(path.join(stage, 'entrypoints', 'fanloop-workflow', '._SKILL.md'), 'appledouble');
    fs.writeFileSync(path.join(stage, '._bin'), 'appledouble');
  }, {format: 'pax'});
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-pax-'));
  await installRelease({
    version,
    dataHome: root,
    platform: {os: 'linux', arch: 'amd64'},
    releaseDir: release,
    skillRoots: skillRoots(root),
  });
  const installed = path.join(root, 'releases', version);
  assert.equal(fs.readFileSync(path.join(installed, 'entrypoints', 'fanloop-workflow', 'SKILL.md'), 'utf8'), '# workflow\n');
  assert.equal(fs.existsSync(path.join(installed, 'entrypoints', 'fanloop-workflow', '._SKILL.md')), false);
  assertNoArchiveMetadata(installed);
  assert.equal(fs.readlinkSync(path.join(root, 'current')), path.join('releases', version));
});

test('rejects hard links and device entries before switching current', async () => {
  const version = '1.2.3';
  const linked = releaseAssets(version, (stage) => {
    fs.linkSync(path.join(stage, 'bin', 'fanloop'), path.join(stage, 'bin', 'fanloop-link'));
  });
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-hardlink-'));
  await assert.rejects(
    () => installRelease({
      version,
      dataHome: root,
      platform: {os: 'linux', arch: 'amd64'},
      releaseDir: linked.release,
      skillRoots: skillRoots(root),
    }),
    /not a regular file/,
  );
  assert.equal(fs.existsSync(path.join(root, 'current')), false);

  const header = Buffer.alloc(512, 0);
  header.write('bin/fanloop', 0);
  header.write('0000644\0', 100);
  header.write('0000000\0', 108);
  header.write('0000000\0', 116);
  header.write('00000000000\0', 124);
  header.write('00000000000\0', 136);
  header.fill(0x20, 148, 156);
  header[156] = '4'.charCodeAt(0);
  header.write('ustar\0', 257);
  header.write('00', 263);
  let sum = 0;
  for (const byte of header) {
    sum += byte;
  }
  header.write(`${sum.toString(8).padStart(6, '0')}\0 `, 148);
  const device = zlib.gzipSync(Buffer.concat([header, Buffer.alloc(1024)]));
  assert.throws(() => layout.extractReleaseArchive(device, fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-device-'))), /not a regular file/);
});

test('build-release accepts an existing empty directory', () => {
  const repo = path.resolve(__dirname, '..', '..');
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-build-release-'));
  const fakeBin = path.join(scratch, 'bin');
  fs.mkdirSync(fakeBin);
  fs.writeFileSync(path.join(fakeBin, 'go'), '#!/bin/sh\nif [ "$1" = "build" ]; then echo reached-build >&2; exit 1; fi\necho "unexpected go $*" >&2\nexit 1\n', {mode: 0o755});
  const run = (output) => spawnSync(path.join(repo, 'scripts', 'build-release.sh'), [output], {
    cwd: repo,
    env: {...process.env, PATH: `${fakeBin}${path.delimiter}${process.env.PATH}`},
    encoding: 'utf8',
  });
  const empty = fs.mkdtempSync(path.join(scratch, 'empty-'));
  const created = path.join(scratch, 'created');
  const accepted = run(empty);
  const fresh = run(created);
  for (const result of [accepted, fresh]) {
    assert.match(result.stderr, /reached-build/);
    assert.doesNotMatch(result.stderr, /File exists/);
  }
  assert.deepEqual(fs.readdirSync(empty), []);
  assert.deepEqual(fs.readdirSync(created), []);
  fs.writeFileSync(path.join(empty, 'leftover'), 'x');
  const rejected = run(empty);
  assert.notEqual(rejected.status, 0);
  assert.match(rejected.stderr, /not empty/);
  assert.equal(fs.readFileSync(path.join(empty, 'leftover'), 'utf8'), 'x');
});

test('rejects symlink archive entries before switching current', async () => {
  const version = '1.2.3';
  const linked = releaseAssets(version, (stage) => {
    fs.symlinkSync('fanloop', path.join(stage, 'bin', 'linked'));
  });
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-symlink-'));
  await assert.rejects(
    () => installRelease({
      version,
      dataHome: root,
      platform: {os: 'linux', arch: 'amd64'},
      releaseDir: linked.release,
      skillRoots: skillRoots(root),
    }),
    /not a regular file/,
  );
  assert.equal(fs.existsSync(path.join(root, 'current')), false);
});

test('caps downloads and explains a private repository 404', async () => {
  const server = http.createServer((request, response) => {
    if (request.url === '/missing') {
      response.writeHead(404);
      response.end('missing');
      return;
    }
    response.writeHead(200, {'Content-Type': 'application/octet-stream'});
    response.end(Buffer.alloc(32, 1));
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    const base = `http://127.0.0.1:${server.address().port}`;
    await assert.rejects(download(`${base}/missing`, 'fanloop.tar.gz'), /HTTP 404/);
    await assert.rejects(download(`${base}/missing`, 'fanloop.tar.gz'), /public repository or authentication/);
    await assert.rejects(download(`${base}/big`, 'fanloop.tar.gz', 8), /exceeds 8 bytes/);
  } finally {
    await new Promise((resolve, reject) => server.close((err) => err ? reject(err) : resolve()));
  }
});

test('prints an exact PATH instruction only when the binary is not already reachable', () => {
  const root = '/tmp/fanloop-home';
  const bin = path.join(root, 'current', 'bin');
  assert.deepEqual(layout.pathInstruction(root, {PATH: `/usr/bin${path.delimiter}${bin}`, npm_config_prefix: ''}), {
    onPath: true,
    instruction: '',
  });
  assert.equal(
    layout.pathInstruction(root, {PATH: '/usr/bin', npm_config_prefix: '/opt/fanloop'}).instruction,
    `export PATH="${bin}:$PATH"`,
  );
  assert.equal(layout.pathInstruction(root, {PATH: `/usr/bin${path.delimiter}/opt/fanloop/bin`, npm_config_prefix: '/opt/fanloop'}).onPath, true);
});
