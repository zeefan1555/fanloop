const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { checksum, newerInstalled, validateArchive, shellQuote, install } = require('./install.js');
const { installLauncher } = require('./run.js');
const { selectVersion } = require('./resolve-release-version.js');
const { publish } = require('./publish-npm-package.js');
function temporary(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-npm-test-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  return root;
}
test('version allocation handles first release and never hides a registry error', t => {
  const root = temporary(t);
  fs.writeFileSync(path.join(root, 'VERSION'), '0.6.2\n');
  fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify({ name: '@zeefan1555/fanloop-cli', publishConfig: { registry: 'https://registry.npmjs.org/' } }));
  assert.equal(selectVersion(root, () => ({ status: 1, stderr: 'npm error E404' })), '0.6.2');
  assert.equal(selectVersion(root, () => ({ status: 0, stdout: JSON.stringify(['0.6.1', '0.6.9', '0.7.2', '0.6.99-beta']) })), '0.6.10');
  assert.throws(() => selectVersion(root, () => ({ status: 1, stderr: 'npm error E401' })), /Cannot read/);
});
test('GitHub release publishes the verified public npm package', () => {
  const workflow = fs.readFileSync(path.join(__dirname, '..', '.github', 'workflows', 'release.yml'), 'utf8');
  assert.match(workflow, /registry-url: https:\/\/registry\.npmjs\.org\//);
  assert.match(workflow, /node scripts\/publish-npm-package\.js --access=public --tag=candidate/);
  assert.match(workflow, /NODE_AUTH_TOKEN: \$\{\{ secrets\.NPM_TOKEN \}\}/);
});
test('update never downgrades a healthy newer stable release; direct install can roll back', t => {
  const root = temporary(t);
  fs.mkdirSync(path.join(root, 'current/bin'), { recursive: true });
  fs.writeFileSync(path.join(root, 'current/bin/fanloop'), '#!/bin/sh\n', { mode: 0o755 });
  fs.writeFileSync(path.join(root, 'current/release.json'), JSON.stringify({ release_version: '0.6.10' }));
  assert.equal(newerInstalled({ FANLOOP_DATA_HOME: root, FANLOOP_UPDATE_FORWARD_ONLY: '1' }, '0.6.2'), '0.6.10');
  assert.equal(newerInstalled({ FANLOOP_DATA_HOME: root, FANLOOP_UPDATE_FORWARD_ONLY: '1' }, '0.6.10'), '');
  assert.equal(newerInstalled({ FANLOOP_DATA_HOME: root }, '0.6.2'), '');
});
test('update leaves Workflow sourcing to the installed Fanloop payload', t => {
  const root = temporary(t), pkg = path.join(root, 'package'), current = path.join(root, 'data/current');
  fs.mkdirSync(path.join(current, 'bin'), { recursive: true });
  fs.mkdirSync(pkg);
  const captured = path.join(root, 'argv.txt');
  fs.writeFileSync(path.join(current, 'bin/fanloop'), '#!/bin/sh\nprintf \'%s\\n\' "$@" > ' + shellQuote(captured) + '\n', { mode: 0o755 });
  fs.writeFileSync(path.join(current, 'release.json'), JSON.stringify({ release_version: '0.6.10' }));
  fs.writeFileSync(path.join(pkg, 'package.json'), JSON.stringify({ name: '@zeefan1555/fanloop-cli', version: '0.6.2' }));

  assert.deepEqual(install({ FANLOOP_DATA_HOME: path.join(root, 'data'), FANLOOP_UPDATE_FORWARD_ONLY: '1' }, pkg), { version: '0.6.10', kept: true });
  const argv = fs.readFileSync(captured, 'utf8');
  assert.doesNotMatch(argv, /^--config-source$/m);
  assert.doesNotMatch(argv, /^--sync-workflows$/m);
});
test('archive boundary accepts regular payloads and rejects links before extraction', t => {
  const root = temporary(t);
  fs.mkdirSync(path.join(root, 'bin'));
  fs.writeFileSync(path.join(root, 'bin/fanloop'), 'binary');
  fs.writeFileSync(path.join(root, 'release.json'), '{}');
  const safe = path.join(root, 'safe.tar.gz');
  execFileSync('tar', ['-czf', safe, '-C', root, 'bin', 'release.json']);
  validateArchive(safe);
  assert.match(checksum(safe), /^[0-9a-f]{64}$/);
  fs.symlinkSync('/tmp', path.join(root, 'bin/escape'));
  const unsafe = path.join(root, 'unsafe.tar.gz');
  execFileSync('tar', ['-czf', unsafe, '-C', root, 'bin', 'release.json']);
  assert.throws(() => validateArchive(unsafe), /Unsupported/);
});
test('PATH instructions quote command substitution and apostrophes literally', () => {
  const value = "/tmp/it's $(touch must-not-run) `echo unsafe`";
  const actual = execFileSync('/bin/sh', ['-c', 'printf %s ' + shellQuote(value)], { encoding: 'utf8' });
  assert.equal(actual, value);
});
for (const scenario of ['success', 'local-failure', 'candidate-failure', 'master-advanced', 'latest-failure', 'first-latest-failure', 'promotion-response-failure', 'latest-moved-before-rollback']) {
  test('publication transaction: ' + scenario, t => {
    const root = temporary(t), commit = 'a'.repeat(40);
    fs.mkdirSync(path.join(root, 'dist'));
    const tarball = path.join(root, 'dist/package.tgz');
    fs.writeFileSync(tarball, 'test fixture');
    fs.writeFileSync(path.join(root, 'dist/npm-release.json'), JSON.stringify({ version: '0.6.2', commit, tarball }));
    let latest = scenario === 'first-latest-failure' ? null : '0.6.1', masterReads = 0;
    const calls = [];
    const run = (command, args) => {
      calls.push([command, ...args]);
      const ok = stdout => ({ status: 0, stdout, stderr: '' });
      if (command === 'git') {
        if (args[0] === 'rev-parse') return ok(commit);
        if (args[0] === 'status') return ok('');
        masterReads++;
        return ok((scenario === 'master-advanced' && masterReads === 2 ? 'b'.repeat(40) : commit) + '\trefs/heads/master');
      }
      if (command === process.execPath) {
        const selector = args.at(-1);
        if ((scenario === 'local-failure' && selector === tarball) || (scenario === 'candidate-failure' && selector === '0.6.2') || (scenario.includes('latest-failure') && selector === 'latest') || (scenario === 'latest-moved-before-rollback' && selector === 'latest')) {
          if (scenario === 'latest-moved-before-rollback') latest = '0.6.3';
          return { status: 1, stderr: 'smoke failed' };
        }
        return ok('');
      }
      if (command === 'npm' && args[0] === 'view') return ok(JSON.stringify(latest ? { latest } : {}));
      if (command === 'npm' && args[0] === 'dist-tag') {
        latest = args[1] === 'rm' ? null : args[2].split('@').at(-1);
        if (scenario === 'promotion-response-failure' && latest === '0.6.2') return { status: 1, stderr: 'response lost after tag write' };
      }
      return ok('');
    };
    const action = () => publish(['--tag=candidate', '--access=public'], run, root);
    if (scenario === 'success') { action(); assert.equal(latest, '0.6.2'); }
    else {
      assert.throws(action);
      assert.equal(latest, scenario === 'first-latest-failure' ? null : scenario === 'latest-moved-before-rollback' ? '0.6.3' : '0.6.1');
    }
    const published = calls.some(call => call[0] === 'npm' && call[1] === 'publish');
    assert.equal(published, scenario !== 'local-failure');
    if (scenario === 'candidate-failure') assert.ok(!calls.some(call => call[1] === 'dist-tag'));
    if (scenario === 'latest-moved-before-rollback') assert.equal(calls.filter(call => call[1] === 'dist-tag').length, 1);
  });
}

test('launcher persistence failure happens before payload activation', t => {
  const root = temporary(t), pkg = path.join(root, 'package'), stage = path.join(root, 'payload');
  fs.mkdirSync(path.join(pkg, 'releases'), { recursive: true });
  fs.mkdirSync(path.join(stage, 'bin'), { recursive: true });
  const marker = path.join(root, 'payload-was-run');
  const binary = path.join(stage, 'bin/fanloop');
  fs.writeFileSync(binary, '#!/bin/sh\ntouch ' + shellQuote(marker) + '\n', { mode: 0o755 });
  fs.writeFileSync(path.join(stage, 'release.json'), JSON.stringify({ schema_version: 4, release_version: '0.6.2', cli: { version: '0.6.2', binary_sha256: 'sha256:' + checksum(binary) } }));
  fs.writeFileSync(path.join(pkg, 'package.json'), JSON.stringify({ name: '@zeefan1555/fanloop-cli', version: '0.6.2' }));
  const arch = { x64: 'amd64', arm64: 'arm64' }[process.arch];
  const name = 'fanloop-0.6.2-' + process.platform + '-' + arch + '.tar.gz';
  const archive = path.join(pkg, 'releases', name);
  execFileSync('tar', ['-czf', archive, '-C', stage, 'bin', 'release.json']);
  fs.writeFileSync(path.join(pkg, 'checksums.txt'), checksum(archive) + '  ' + name + '\n');
  assert.throws(() => install({ FANLOOP_DATA_HOME: path.join(root, 'data') }, pkg, () => { throw new Error('npm prefix not writable'); }), /npm prefix not writable/);
  assert.ok(!fs.existsSync(marker));
  assert.ok(!fs.existsSync(path.join(root, 'data/current')));
});

test('official update uses the production Fornax configuration path', t => {
  const root = temporary(t), pkg = path.join(root, 'package'), stage = path.join(root, 'payload');
  fs.mkdirSync(path.join(pkg, 'releases'), { recursive: true });
  fs.mkdirSync(path.join(stage, 'bin'), { recursive: true });
  const captured = path.join(root, 'argv.json');
  const binary = path.join(stage, 'bin/fanloop');
  fs.writeFileSync(binary, '#!/bin/sh\nprintf \'%s\\n\' "$@" > ' + shellQuote(captured) + '\ndata_root=\nwhile [ "$#" -gt 0 ]; do\n  if [ "$1" = "--data-root" ]; then data_root=$2; shift 2; else shift; fi\ndone\nmkdir -p "$data_root/current/bin"\ncp "$0" "$data_root/current/bin/fanloop"\nprintf \'{"release_version":"0.6.2"}\\n\' > "$data_root/current/release.json"\n', { mode: 0o755 });
  fs.writeFileSync(path.join(stage, 'release.json'), JSON.stringify({ schema_version: 4, release_version: '0.6.2', cli: { version: '0.6.2', binary_sha256: 'sha256:' + checksum(binary) } }));
  fs.writeFileSync(path.join(pkg, 'package.json'), JSON.stringify({ name: '@zeefan1555/fanloop-cli', version: '0.6.2' }));
  const arch = { x64: 'amd64', arm64: 'arm64' }[process.arch];
  const name = 'fanloop-0.6.2-' + process.platform + '-' + arch + '.tar.gz';
  const archive = path.join(pkg, 'releases', name);
  execFileSync('tar', ['-czf', archive, '-C', stage, 'bin', 'release.json']);
  fs.writeFileSync(path.join(pkg, 'checksums.txt'), checksum(archive) + '  ' + name + '\n');
  install({ FANLOOP_DATA_HOME: path.join(root, 'data'), FANLOOP_UPDATE_FORWARD_ONLY: '1' }, pkg);
  const argv = fs.readFileSync(captured, 'utf8');
  assert.doesNotMatch(argv, /^--config-source$/m);
  assert.doesNotMatch(argv, /^--sync-workflows$/m);
});

test('published package smoke uses isolated repository fixtures', () => {
  const source = fs.readFileSync(path.join(__dirname, 'verify-published-package.js'), 'utf8');
  assert.match(source, /FANLOOP_CONFIG_SOURCE: path\.join\(repo, 'tests', 'testdata'\)/);
  assert.match(source, /assert\.deepEqual\(fs\.readFileSync\(skill\), edited/);
});

test('persistent launcher installs from an external copy', t => {
  const root = temporary(t), pkg = path.join(root, 'package'), prefix = path.join(root, 'npm');
  fs.mkdirSync(pkg); fs.writeFileSync(path.join(pkg, 'package.json'), '{}');
  const env = { PATH: path.join(prefix, 'bin'), HOME: root, SHELL: '/bin/sh' };
  installLauncher(env, pkg, (_command, args) => {
    if (args.includes('install')) {
      const source = args.at(-1);
      assert.ok(!source.startsWith(pkg));
      assert.ok(fs.existsSync(path.join(source, 'package.json')));
      return { status: 0, stdout: '', stderr: '' };
    }
    return { status: 0, stdout: prefix + '\n', stderr: '' };
  });
});
