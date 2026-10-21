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
  const version = fs.readFileSync(path.join(repo, 'VERSION'), 'utf8').trim();
  const run = (output, ref = {}) => spawnSync(path.join(repo, 'scripts', 'build-release.sh'), [output], {
    cwd: repo,
    env: {
      ...process.env,
      PATH: `${fakeBin}${path.delimiter}${process.env.PATH}`,
      // Branch jobs set these; they must not be treated as a release tag.
      GITHUB_REF: 'refs/heads/main',
      GITHUB_REF_NAME: 'main',
      ...ref,
    },
    encoding: 'utf8',
  });
  const empty = fs.mkdtempSync(path.join(scratch, 'empty-'));
  const created = path.join(scratch, 'created');
  const accepted = run(empty);
  const fresh = run(created);
  for (const result of [accepted, fresh]) {
    assert.match(result.stderr, /reached-build/);
    assert.doesNotMatch(result.stderr, /File exists/);
    assert.doesNotMatch(result.stderr, /does not match VERSION/);
  }
  assert.deepEqual(fs.readdirSync(empty), []);
  assert.deepEqual(fs.readdirSync(created), []);
  fs.writeFileSync(path.join(empty, 'leftover'), 'x');
  const rejected = run(empty);
  assert.notEqual(rejected.status, 0);
  assert.match(rejected.stderr, /not empty/);
  assert.equal(fs.readFileSync(path.join(empty, 'leftover'), 'utf8'), 'x');

  const mismatched = run(fs.mkdtempSync(path.join(scratch, 'tag-')), {
    GITHUB_REF: 'refs/tags/v9.9.9',
    GITHUB_REF_NAME: 'v9.9.9',
  });
  assert.notEqual(mismatched.status, 0);
  assert.match(mismatched.stderr, new RegExp(`tag v9\\.9\\.9 does not match VERSION ${version.replaceAll('.', '\\.')}`));
  assert.doesNotMatch(mismatched.stderr, /reached-build/);

  const matched = run(fs.mkdtempSync(path.join(scratch, 'match-')), {
    GITHUB_REF: `refs/tags/v${version}`,
    GITHUB_REF_NAME: `v${version}`,
  });
  assert.match(matched.stderr, /reached-build/);
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

function replaceReleaseBinary(stage, version, script) {
  fs.writeFileSync(path.join(stage, 'bin', 'fanloop'), script, {mode: 0o755});
  const manifest = JSON.parse(fs.readFileSync(path.join(stage, 'release.json'), 'utf8'));
  manifest.cli.binary_sha256 = layout.fileDigest(path.join(stage, 'bin', 'fanloop'));
  fs.writeFileSync(path.join(stage, 'release.json'), `${JSON.stringify(manifest)}\n`);
}

function configSyncEnv(overrides = {}) {
  const env = {...process.env, ...overrides};
  if (!Object.prototype.hasOwnProperty.call(overrides, 'FANLOOP_CONFIG_SOURCE')) {
    delete env.FANLOOP_CONFIG_SOURCE;
  }
  if (!Object.prototype.hasOwnProperty.call(overrides, 'FANLOOP_NO_AUTO_CONFIG')) {
    delete env.FANLOOP_NO_AUTO_CONFIG;
  }
  return env;
}

function stubFanloop(record, status) {
  return `#!/bin/sh\nprintf '%s\\n' "$0" "$@" > ${JSON.stringify(record)}\nexit ${status}\n`;
}

async function installWithStub(script, env) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-config-sync-'));
  const version = '1.2.3';
  const {release} = releaseAssets(version, (stage) => {
    replaceReleaseBinary(stage, version, script);
  });
  await installRelease({
    version,
    dataHome: root,
    platform: {os: 'linux', arch: 'amd64'},
    releaseDir: release,
    skillRoots: skillRoots(root),
    env,
  });
  return root;
}

test('synchronizes configuration with update config after switching current', async () => {
  const record = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-sync-record-')), 'argv');
  const root = await installWithStub(stubFanloop(record, 0), configSyncEnv());
  assert.equal(fs.readlinkSync(path.join(root, 'current')), path.join('releases', '1.2.3'));
  assert.equal(
    fs.readFileSync(record, 'utf8'),
    `${path.join(root, 'current', 'bin', 'fanloop')}\nupdate\nconfig\n`,
  );
});

test('skips configuration sync when FANLOOP_NO_AUTO_CONFIG=1', async () => {
  const record = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-sync-record-')), 'argv');
  await installWithStub(stubFanloop(record, 0), configSyncEnv({FANLOOP_NO_AUTO_CONFIG: '1'}));
  assert.equal(fs.existsSync(record), false);
});

test('skips configuration sync when FANLOOP_CONFIG_SOURCE is set', async () => {
  const record = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-sync-record-')), 'argv');
  await installWithStub(stubFanloop(record, 0), configSyncEnv({FANLOOP_CONFIG_SOURCE: '/tmp/fanloop-author-source'}));
  assert.equal(fs.existsSync(record), false);
});

test('configuration sync failure does not fail install', async () => {
  const record = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-sync-record-')), 'argv');
  const errors = [];
  const original = console.error;
  console.error = (...args) => {
    errors.push(args.map(String).join(' '));
  };
  try {
    const root = await installWithStub(stubFanloop(record, 1), configSyncEnv());
    assert.equal(fs.readlinkSync(path.join(root, 'current')), path.join('releases', '1.2.3'));
  } finally {
    console.error = original;
  }
  assert.equal(errors.includes('fanloop install: configuration sync failed; run "fanloop update config"'), true);
  assert.equal(fs.readFileSync(record, 'utf8').endsWith('update\nconfig\n'), true);
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
    `export PATH='/opt/fanloop/bin':"$PATH"`,
  );
  assert.equal(layout.pathInstruction(root, {PATH: `/usr/bin${path.delimiter}/opt/fanloop/bin`, npm_config_prefix: '/opt/fanloop'}).onPath, true);
});

test('persists a single safely quoted PATH block for zsh and preserves user bytes', () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-profile-'));
  const zdot = path.join(home, 'zsh config');
  const prefix = path.join(home, "npm 'prefix");
  const bin = path.join(prefix, 'bin');
  fs.mkdirSync(zdot);
  fs.mkdirSync(bin, {recursive: true});
  fs.writeFileSync(path.join(bin, 'fanloop'), '#!/bin/sh\necho candidate\n', {mode: 0o755});
  const original = Buffer.from('# user\r\nexport USER_VALUE=kept\n# no final newline');
  fs.writeFileSync(path.join(zdot, '.zshrc'), original);
  const env = {HOME: home, ZDOTDIR: zdot, SHELL: '/bin/zsh', npm_config_prefix: prefix, PATH: '/usr/bin:/bin'};
  layout.ensureLauncherPath(path.join(home, 'data'), env);
  const first = fs.readFileSync(path.join(zdot, '.zshrc'));
  assert.deepEqual(first.subarray(0, original.length), original);
  layout.ensureLauncherPath(path.join(home, 'data'), env);
  assert.deepEqual(fs.readFileSync(path.join(zdot, '.zshrc')), first);
  for (const name of ['.zshrc', '.zprofile']) {
    assert.equal(fs.readFileSync(path.join(zdot, name), 'utf8').match(/# >>> fanloop PATH >>>/g).length, 1);
  }
  if (fs.existsSync('/bin/zsh')) {
    for (const mode of ['-ic', '-lc']) {
      const result = spawnSync('/bin/zsh', [mode, 'fanloop'], {env, encoding: 'utf8'});
      assert.equal(result.status, 0, result.stderr);
      assert.equal(result.stdout.trim(), 'candidate');
    }
  }
});

test('selects bash login precedence and sh profile without touching other files', () => {
  for (const login of ['.bash_profile', '.bash_login', '.profile', null]) {
    const home = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-bash-'));
    if (login) fs.writeFileSync(path.join(home, login), '# user\n');
    const nativeBin = path.join(home, 'data', 'current', 'bin');
    fs.mkdirSync(nativeBin, {recursive: true});
    fs.writeFileSync(path.join(nativeBin, 'fanloop'), '#!/bin/sh\necho candidate\n', {mode: 0o755});
    const env = {HOME: home, SHELL: '/bin/bash', PATH: '/usr/bin:/bin'};
    layout.ensureLauncherPath(path.join(home, 'data'), env);
    assert.equal(fs.existsSync(path.join(home, '.bashrc')), true);
    assert.equal(fs.existsSync(path.join(home, login || '.bash_profile')), true);
    assert.equal(fs.readdirSync(home).filter((name) => ['.bash_profile', '.bash_login', '.profile'].includes(name)).length, 1);
    // Explicitly source the login file: some system /etc/profile files reset HOME.
    for (const profile of ['.bashrc', login || '.bash_profile']) {
      const result = spawnSync('/bin/bash', ['--noprofile', '--norc', '-c', '. "$HOME/$1"; fanloop', 'check', profile], {env, encoding: 'utf8'});
      assert.equal(result.status, 0, result.stderr);
      assert.equal(result.stdout.trim(), 'candidate');
    }
    layout.ensureLauncherPath(path.join(home, 'data'), {...env, SHELL: '/bin/sh'});
    assert.equal(fs.existsSync(path.join(home, '.profile')), true);
  }
});

test('PATH configuration rejects unsupported shells, invalid paths and unwriteable profiles', (t) => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-profile-error-'));
  const data = path.join(home, 'data');
  const env = {HOME: home, SHELL: '/bin/zsh'};
  // Even a broken empty-HOME implementation must stay inside this test's isolated home.
  t.mock.method(os, 'homedir', () => home);
  assert.throws(() => layout.ensureLauncherPath(data, {...env, SHELL: '/bin/fish'}), /unsupported shell/);
  for (const overrides of [{HOME: ''}, {ZDOTDIR: ''}, {npm_config_prefix: ''}, {HOME: 'relative'}, {ZDOTDIR: 'relative'}, {npm_config_prefix: 'relative'}, {npm_config_prefix: `${home}\ncommand`}, {npm_config_prefix: `${home}:other`}]) {
    assert.throws(() => layout.ensureLauncherPath(data, {...env, ...overrides}), /absolute|invalid/);
  }
  fs.mkdirSync(path.join(home, '.zshrc'));
  assert.throws(() => layout.ensureLauncherPath(data, env), /PATH configuration/);
  assert.equal(fs.existsSync(path.join(home, '.zprofile')), false);
});

test('updates only its own PATH block and refuses malformed markers', () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-marker-'));
  const env = {HOME: home, SHELL: '/bin/sh'};
  layout.ensureLauncherPath(path.join(home, 'old'), env);
  const profile = path.join(home, '.profile');
  fs.appendFileSync(profile, '# after\n');
  layout.ensureLauncherPath(path.join(home, 'new'), env);
  assert.match(fs.readFileSync(profile, 'utf8'), /new\/current\/bin/);
  assert.equal(fs.readFileSync(profile, 'utf8').endsWith('# after\n'), true);
  fs.writeFileSync(profile, '# >>> fanloop PATH >>>\nuser command\n');
  const before = fs.readFileSync(profile);
  assert.throws(() => layout.ensureLauncherPath(path.join(home, 'new'), env), /marker/);
  assert.deepEqual(fs.readFileSync(profile), before);
});

test('profile atomic writes preserve mode and symlinks, and partial write failure keeps user bytes', (t) => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-atomic-profile-'));
  const target = path.join(home, 'user-profile');
  const profile = path.join(home, '.profile');
  fs.writeFileSync(target, '# original user bytes\n', {mode: 0o640});
  fs.chmodSync(target, 0o640);
  fs.symlinkSync('user-profile', profile);
  const env = {HOME: home, SHELL: '/bin/sh'};
  layout.ensureLauncherPath(path.join(home, 'data'), env);
  assert.equal(fs.readlinkSync(profile), 'user-profile');
  assert.equal(fs.statSync(target).mode & 0o777, 0o640);
  const before = fs.readFileSync(target);
  const write = fs.writeFileSync;
  t.mock.method(fs, 'writeFileSync', (file, bytes, options) => {
    if (path.basename(file).startsWith('.fanloop-path-')) {
      write(file, bytes.subarray(0, 8), options);
      throw new Error('simulated partial write');
    }
    return write(file, bytes, options);
  });
  assert.throws(() => layout.ensureLauncherPath(path.join(home, 'other'), env), /could not write/);
  assert.deepEqual(fs.readFileSync(target), before);
  assert.equal(fs.readlinkSync(profile), 'user-profile');
  assert.equal(fs.readdirSync(home).some((name) => name.startsWith('.fanloop-path-')), false);
});

test('retries interrupted 2xx bodies and caps permanent interruption at four attempts', async () => {
  const counts = new Map();
  const server = http.createServer((request, response) => {
    const count = (counts.get(request.url) || 0) + 1;
    counts.set(request.url, count);
    response.writeHead(200, {'Content-Length': '8'});
    if (request.url === '/recover' && count === 2) {
      response.end('complete');
      return;
    }
    response.write('part');
    setTimeout(() => response.destroy(), 10);
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    const base = `http://127.0.0.1:${server.address().port}`;
    assert.equal((await download(`${base}/recover`, 'archive')).toString(), 'complete');
    assert.equal(counts.get('/recover'), 2);
    await assert.rejects(download(`${base}/forever`, 'archive'), /four attempts|4 attempts/);
    assert.equal(counts.get('/forever'), 4);
  } finally {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
  }
});

test('HTTP failures, oversize and unsafe redirects never retry or leak signed URLs', async () => {
  const counts = new Map();
  const secret = 'signed-test-query';
  const server = http.createServer((request, response) => {
    const pathname = new URL(request.url, 'http://localhost').pathname;
    counts.set(pathname, (counts.get(pathname) || 0) + 1);
    if (pathname === '/unsafe') {
      response.writeHead(302, {Location: `http://username:password@127.0.0.1:${server.address().port}/target?token=${secret}`});
    } else if (pathname === '/large') {
      response.writeHead(200);
    } else {
      response.writeHead(Number(pathname.slice(1)));
    }
    response.end('too big body');
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    const base = `http://127.0.0.1:${server.address().port}`;
    for (const pathname of ['/404', '/503', '/large', '/unsafe']) {
      await assert.rejects(download(`${base}${pathname}?token=${secret}`, 'archive', 4), (err) => {
        assert.doesNotMatch(err.message, /signed-test-query|username|password|http:\/\//);
        return true;
      });
      assert.equal(counts.get(pathname), 1);
    }
    assert.equal(counts.has('/target'), false);
  } finally {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
  }
});

test('validates the initial URL before fetch and sanitizes unknown or certificate errors', async (t) => {
  let calls = 0;
  t.mock.method(global, 'fetch', async () => {
    calls += 1;
    const err = new Error('https://user:password@example.invalid?token=private-query');
    err.cause = {code: 'CERT_HAS_EXPIRED'};
    throw err;
  });
  for (const url of ['http://example.invalid/file', 'https://user:password@example.invalid', 'not a URL']) {
    await assert.rejects(download(url, 'archive'), /unsafe|invalid/);
  }
  assert.equal(calls, 0);
  await assert.rejects(download('https://example.invalid/file?token=private-query', 'archive'), (err) => {
    assert.doesNotMatch(err.message, /private-query|password|https:\/\//);
    return true;
  });
  assert.equal(calls, 1);
});

test('known transport timeouts retry but unknown and pre-body socket errors do not', async (t) => {
  let calls = 0;
  let code = 'UND_ERR_CONNECT_TIMEOUT';
  t.mock.method(global, 'fetch', async () => {
    calls += 1;
    const err = new Error('private URL must be discarded');
    err.cause = {code};
    throw err;
  });
  await assert.rejects(download('https://example.invalid/file', 'archive'), /4 attempts|four attempts/);
  assert.equal(calls, 4);
  for (code of ['UND_ERR_SOCKET', 'ECONNRESET', 'UNKNOWN']) {
    calls = 0;
    await assert.rejects(download('https://example.invalid/file', 'archive'), /download archive failed/);
    assert.equal(calls, 1);
  }
});

test('one 30-second signal covers headers, redirects and body on each attempt', async (t) => {
  const deadlines = [];
  const signals = [];
  t.mock.method(AbortSignal, 'timeout', (ms) => {
    deadlines.push(ms);
    return new AbortController().signal;
  });
  let calls = 0;
  t.mock.method(global, 'fetch', async (url, options) => {
    calls += 1;
    assert.equal(options.redirect, 'manual');
    signals.push(options.signal);
    if (calls === 1) return new Response('', {status: 302, headers: {Location: 'https://example.invalid/body'}});
    return new Response('body');
  });
  assert.equal((await download('https://example.invalid/redirect', 'archive')).toString(), 'body');
  assert.deepEqual(deadlines, [30000]);
  assert.equal(signals[0], signals[1]);
});

test('deadline cancellation remains retryable when fetch surfaces only AbortError', async (t) => {
  let calls = 0;
  t.mock.method(AbortSignal, 'timeout', () => {
    const controller = new AbortController();
    controller.abort(new DOMException('deadline exceeded', 'TimeoutError'));
    return controller.signal;
  });
  t.mock.method(global, 'fetch', async () => {
    calls += 1;
    throw new DOMException('request aborted', 'AbortError');
  });
  await assert.rejects(download('https://example.invalid/body', 'archive'), /4 attempts/);
  assert.equal(calls, 4);
});

test('body timeout restarts with a fresh deadline and bounded redirect loops reject', async (t) => {
  const realTimeout = AbortSignal.timeout;
  const signals = [];
  let shortDeadline = true;
  t.mock.method(AbortSignal, 'timeout', (ms) => {
    assert.equal(ms, 30000);
    const signal = realTimeout.call(AbortSignal, shortDeadline ? 500 : ms);
    signals.push(signal);
    return signal;
  });
  let requests = 0;
  const server = http.createServer((request, response) => {
    requests += 1;
    if (request.url === '/loop') {
      response.writeHead(302, {Location: '/loop'});
      response.end();
    } else {
      response.writeHead(200);
      response.write('waiting');
      // Bound even the pre-implementation red run instead of leaving a test server hung.
      setTimeout(() => response.destroy(), 5000);
    }
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    const base = `http://127.0.0.1:${server.address().port}`;
    await assert.rejects(download(`${base}/body`, 'archive'), /4 attempts|four attempts/);
    assert.equal(requests, 4);
    assert.equal(new Set(signals).size, 4);
    requests = 0;
    shortDeadline = false;
    await assert.rejects(download(`${base}/loop`, 'archive'), /redirect/);
    assert.equal(requests, 6);
  } finally {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
  }
});
