#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { stableVersion } = require('./install.js');
const packageName = '@zeefan1555/fanloop-cli';
const registry = 'https://registry.npmjs.org/';
function publish(args, run = spawnSync, root = path.resolve(__dirname, '..')) {
  if (args.length !== 2 || !args.includes('--tag=candidate') || !args.includes('--access=public')) throw new Error('Publish requires --tag=candidate --access=public');
  const artifact = JSON.parse(fs.readFileSync(path.join(root, 'dist/npm-release.json'), 'utf8'));
  if (!stableVersion(artifact.version) || !fs.statSync(artifact.tarball).isFile()) throw new Error('Build a release artifact first');
  const execute = (command, argv, options = {}) => {
    const result = run(command, argv, { cwd: root, encoding: 'utf8', ...options });
    if (result.error || result.status !== 0) throw new Error(command + ' failed: ' + (result.error?.message || result.stderr || result.status));
    return (result.stdout || '').trim();
  };
  const head = execute('git', ['rev-parse', 'HEAD']);
  if (head !== artifact.commit || execute('git', ['status', '--porcelain'])) throw new Error('Release source is not the frozen clean commit');
  const assertCurrentMaster = () => {
    const current = execute('git', ['ls-remote', 'origin', 'refs/heads/master']).split(/\s+/)[0];
    if (current !== head) throw new Error('master advanced; do not publish an obsolete candidate');
  };
  const anonymous = { ...process.env };
  for (const key of Object.keys(anonymous)) if (/^(NODE_AUTH_TOKEN|NPM_TOKEN|npm_config_.*)$/i.test(key)) delete anonymous[key];
  const anonymousRoot = fs.mkdtempSync(path.join(root, 'dist', 'npm-anonymous-'));
  Object.assign(anonymous, { NPM_CONFIG_USERCONFIG: path.join(anonymousRoot, 'user.npmrc'), NPM_CONFIG_GLOBALCONFIG: path.join(anonymousRoot, 'global.npmrc'), NPM_CONFIG_REGISTRY: registry });
  fs.writeFileSync(anonymous.NPM_CONFIG_USERCONFIG, ''); fs.writeFileSync(anonymous.NPM_CONFIG_GLOBALCONFIG, '');
  const latest = () => {
    const result = run('npm', ['view', packageName, 'dist-tags', '--json', '--prefer-online', '--registry=' + registry], { cwd: root, env: anonymous, encoding: 'utf8' });
    if (result.status !== 0) {
      if (/\b(E404|ETARGET)\b/.test((result.stderr || '') + (result.stdout || ''))) return null;
      throw new Error('Cannot read current latest: ' + (result.error?.message || result.stderr));
    }
    const version = JSON.parse(result.stdout).latest;
    if (version === undefined) return null;
    if (!stableVersion(version)) throw new Error('Current latest is not a stable version');
    return version;
  };
  const verify = selector => execute(process.execPath, [path.join(root, 'scripts/verify-published-package.js'), artifact.version, selector], { env: { ...process.env, FANLOOP_EXPECTED_COMMIT: head }, stdio: 'inherit' });
  const tag = (...argv) => execute('npm', ['dist-tag', ...argv, '--registry=' + registry], { stdio: 'inherit' });
  assertCurrentMaster();
  const previous = latest();
  verify(artifact.tarball);
  execute('npm', ['publish', artifact.tarball, '--registry=' + registry, '--access=public', '--tag=candidate'], { stdio: 'inherit' });
  verify(artifact.version);
  assertCurrentMaster();
  if (latest() !== previous) throw new Error('latest changed during candidate verification; leave it untouched');
  try {
    tag('add', packageName + '@' + artifact.version, 'latest');
    verify('latest');
  } catch (error) {
    if (latest() !== artifact.version) throw new Error(error.message + '; latest changed concurrently and was not overwritten');
    if (previous) tag('add', packageName + '@' + previous, 'latest');
    else tag('rm', packageName, 'latest');
    let restored = false;
    for (let attempt = 0; attempt < 12; attempt++) {
      if (latest() === previous) { restored = true; break; }
      execute('sleep', ['5']);
    }
    if (!restored) throw new Error('Latest verification failed and rollback readback did not match');
    throw new Error('Latest promotion or verification failed; restored previous latest. ' + error.message);
  }
  console.log('Published ' + packageName + '@' + artifact.version + ' from ' + head);
}
if (require.main === module) {
  try { publish(process.argv.slice(2)); } catch (error) { console.error(error.message); process.exit(1); }
}
module.exports = { publish };
