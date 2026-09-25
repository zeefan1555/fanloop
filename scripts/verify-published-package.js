#!/usr/bin/env node
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync, execFileSync } = require('node:child_process');
const assert = require('node:assert/strict');
const { stableVersion } = require('./install.js');
const packageName = '@zeefan1555/fanloop-cli';
const registry = 'https://registry.npmjs.org/';
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
async function verifyPackage(version, selector = version) {
  if (!stableVersion(version)) throw new Error('Expected a stable release version');
  const local = selector.endsWith('.tgz');
  if (!local && selector !== version && selector !== 'latest') throw new Error('Invalid package selector');
  const spec = local ? fs.realpathSync(selector) : packageName + '@' + selector;
  const repo = path.resolve(__dirname, '..');
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-npm-smoke-'));
  const evidenceRoot = path.join(repo, 'dist', 'npm-smoke');
  fs.mkdirSync(evidenceRoot, { recursive: true });
  const evidence = fs.mkdtempSync(path.join(evidenceRoot, version + '-'));
  const env = { ...process.env };
  for (const name of Object.keys(env)) {
    if (/^(NODE_AUTH_TOKEN|NPM_TOKEN|BOTMUX_CHAT_ID|BOTMUX_SESSION_ID|FANLOOP_.*|npm_config_.*)$/i.test(name)) delete env[name];
  }
  Object.assign(env, {
    HOME: path.join(temporary, 'home'), SHELL: '/bin/sh',
    NPM_CONFIG_USERCONFIG: path.join(temporary, 'user.npmrc'), NPM_CONFIG_GLOBALCONFIG: path.join(temporary, 'global.npmrc'),
    NPM_CONFIG_PREFIX: path.join(temporary, 'npm'), NPM_CONFIG_CACHE: path.join(temporary, 'cache'),
    NPM_CONFIG_REGISTRY: registry, FANLOOP_DATA_HOME: path.join(temporary, 'data'),
    FANLOOP_CONFIG_SOURCE: path.join(repo, 'tests', 'testdata'),
  });
  for (const [name, directory] of [['CODEX', 'codex'], ['AGENT', 'agent'], ['TRAE', 'trae'], ['CLAUDE', 'claude']]) {
    env['FANLOOP_' + name + '_SKILLS_ROOT'] = path.join(temporary, 'skills', directory);
  }
  const commands = path.join(temporary, 'bin');
  fs.mkdirSync(commands); fs.mkdirSync(env.HOME);
  fs.writeFileSync(env.NPM_CONFIG_USERCONFIG, ''); fs.writeFileSync(env.NPM_CONFIG_GLOBALCONFIG, '');
  for (const name of ['node', 'npm', 'npx', 'tar', 'gzip', 'sh']) {
    const executable = name === 'node' ? process.execPath : execFileSync('/bin/sh', ['-c', 'command -v ' + name], { encoding: 'utf8' }).trim();
    if (!executable) throw new Error('Smoke dependency missing: ' + name);
    fs.symlinkSync(fs.realpathSync(executable), path.join(commands, name));
  }
  env.PATH = commands;
  const run = (command, args, expected = 0) => {
    const result = spawnSync(command, args, { cwd: temporary, env, encoding: 'utf8', timeout: 180000, maxBuffer: 16 * 1024 * 1024 });
    fs.appendFileSync(path.join(evidence, 'commands.jsonl'), JSON.stringify({ command, args, status: result.status, stdout: result.stdout, stderr: result.stderr, error: result.error?.message }) + '\n');
    if (result.error || result.status !== expected) throw new Error(command + ' ' + args.join(' ') + ': ' + (result.error?.message || result.stderr || result.stdout));
    return result.stdout;
  };
  try {
    assert.equal(spawnSync('go', ['version'], { env }).error?.code, 'ENOENT', 'Client smoke must not have Go');
    if (!local) {
      let visible = false;
      for (let attempt = 0; attempt < 18; attempt++) {
        const result = spawnSync('npm', ['view', spec, 'version', '--json', '--prefer-online'], { cwd: temporary, env, encoding: 'utf8', timeout: 30000 });
        if (result.status === 0 && JSON.parse(result.stdout) === version) { visible = true; break; }
        console.error('Waiting for registry visibility: ' + spec + ' (' + (attempt + 1) + '/18)');
        await pause(5000);
      }
      assert.ok(visible, 'Published selector does not resolve to ' + version);
    }
    const installArgs = ['--yes', '--prefer-online', '--package=' + spec, '--', 'fanloop', 'install'];
    run('npx', installArgs);
    const binary = path.join(env.NPM_CONFIG_PREFIX, 'bin', 'fanloop');
    const identity = JSON.parse(run(binary, ['version'])).data;
    assert.equal(identity.release_version, version);
    if (process.env.FANLOOP_EXPECTED_COMMIT) assert.equal(identity.commit_sha, process.env.FANLOOP_EXPECTED_COMMIT);
    assert.equal(JSON.parse(run(binary, ['doctor'])).data.status, 'healthy');
    assert.equal(JSON.parse(run(binary, ['verify', 'doctor'])).data.outcome, 'passed');
    const smoke = JSON.parse(run(binary, ['verify', 'smoke', '--evidence-dir', path.join(evidence, 'verify')])).data;
    assert.equal(smoke.outcome, 'passed');
    const manifestPath = path.join(smoke.evidence_dir, 'manifest.json');
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    assert.equal(manifest.schema_version, 1); assert.equal(manifest.outcome, 'passed');
    assert.equal(manifest.candidate.commit_sha, identity.commit_sha);
    run(binary, ['install']);
    const live = path.join(env.FANLOOP_DATA_HOME, 'current', 'workflows');
    const wantedRoot = path.join(repo, 'tests', 'testdata', 'workflows');
    const wanted = fs.readdirSync(wantedRoot).filter(name => fs.statSync(path.join(wantedRoot, name)).isDirectory()).sort();
    assert.deepEqual(fs.readdirSync(live).sort(), wanted);
    for (const id of wanted) {
      const root = path.join(temporary, 'requirements', id);
      fs.mkdirSync(root, { recursive: true });
      run(binary, ['flow', 'status', '--root', root], 1);
      run(binary, ['flow', 'init', '--root', root, '--workflow', id, '--title', 'Npm artifact smoke', '--dry-run']);
      assert.ok(!fs.existsSync(path.join(root, '.fanloop/flow/state.json')));
      run(binary, ['flow', 'init', '--root', root, '--workflow', id, '--title', 'Npm artifact smoke']);
      assert.ok(JSON.parse(run(binary, ['flow', 'status', '--root', root])).ok);
      assert.equal(JSON.parse(fs.readFileSync(path.join(root, '.fanloop/flow/state.json'))).release.workflow.id, id);
    }
    const skillRoot = path.join(live, wanted[0], 'skills');
    const skill = path.join(skillRoot, fs.readdirSync(skillRoot)[0], 'SKILL.md');
    fs.appendFileSync(skill, '\nLocal live edit retained by npm updates.\n');
    const edited = fs.readFileSync(skill);
    if (selector === 'latest') run(binary, ['update']);
    else run(binary, ['install']);
    assert.deepEqual(fs.readFileSync(skill), edited, 'Isolated verification overwrote live configuration');
    assert.equal(JSON.parse(run(binary, ['doctor'])).data.status, 'healthy');
    const finalVersion = JSON.parse(run(binary, ['version'])).data;
    assert.equal(finalVersion.release_version, version);
    assert.equal(finalVersion.commit_sha, identity.commit_sha);
    const result = { outcome: 'passed', version, commit: identity.commit_sha, selector: local ? 'local-tarball' : selector, evidence_dir: evidence, client_has_go: false, workflows: wanted, verify_manifest: manifestPath };
    fs.writeFileSync(path.join(evidence, 'result.json'), JSON.stringify(result, null, 2) + '\n');
    console.log(JSON.stringify(result));
    return result;
  } finally { fs.rmSync(temporary, { recursive: true, force: true }); }
}
if (require.main === module) verifyPackage(process.argv[2], process.argv[3]).catch(error => { console.error('Npm smoke failed: ' + error.message); process.exit(1); });
module.exports = { verifyPackage };
