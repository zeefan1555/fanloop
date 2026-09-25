#!/usr/bin/env node
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { dataHome, install, ensureLauncherPath, isUnsafeSudo } = require('./install.js');
const registry = 'https://registry.npmjs.org/';
const packageName = '@zeefan1555/fanloop-cli';

function exitWithChild(result) {
  if (result.error) { console.error(result.error.message); process.exit(1); }
  if (result.signal) process.kill(process.pid, result.signal);
  process.exit(result.status ?? 1);
}
function installLauncher(env, packageRoot = path.resolve(__dirname, '..'), run = spawnSync) {
  const command = env.npm_execpath ? process.execPath : 'npm';
  const prefix = env.npm_execpath ? [env.npm_execpath] : [];
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-launcher-'));
  try {
    const staged = path.join(temporary, 'package');
    fs.cpSync(packageRoot, staged, { recursive: true });
    const args = prefix.concat(['install', '--global', '--install-links', '--ignore-scripts', '--no-audit', '--no-fund', staged]);
    const installed = run(command, args, { env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
    if (installed.error || installed.status !== 0) throw new Error(installed.error?.message || installed.stderr.trim() || 'Persistent launcher installation failed');
    const result = run(command, prefix.concat(['prefix', '--global']), { env, encoding: 'utf8' });
    if (result.error || result.status !== 0 || !result.stdout.trim()) throw new Error('Could not determine npm global prefix');
    ensureLauncherPath(path.join(result.stdout.trim(), 'bin'), env);
  } finally { fs.rmSync(temporary, { recursive: true, force: true }); }
}
function main(args = process.argv.slice(2), env = process.env) {
  if (isUnsafeSudo(env)) throw new Error('Do not run Fanloop with sudo; use your ordinary account.');
  if (args[0] === 'install' || args[0] === 'update') {
    if (args.length === 2 && args[1] === '--help') { console.log('Usage: loop ' + args[0] + '\nThe npm launcher manages the matched local CLI.'); return; }
    if (args.length !== 1) throw new Error('Usage: loop ' + args[0]);
    if (args[0] === 'install') {
      const result = install(env, path.resolve(__dirname, '..'), () => installLauncher(env));
      console.log('Fanloop ' + result.version + (result.kept ? ' is already up to date' : ' installed successfully'));
      return;
    }
    const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-update-'));
    let result;
    try {
      fs.writeSync(1, 'Updating Fanloop from ' + packageName + '@latest...\n');
      result = spawnSync('npx', ['--yes', '--prefer-online', '--package=' + packageName + '@latest', '--', 'fanloop', 'install'], {
        cwd: temporary, env: { ...env, NPM_CONFIG_REGISTRY: registry, FANLOOP_UPDATE_FORWARD_ONLY: '1' }, stdio: 'inherit',
      });
    } finally { fs.rmSync(temporary, { recursive: true, force: true }); }
    exitWithChild(result);
  }
  const binary = path.join(dataHome(env), 'current', 'bin', 'fanloop');
  if (!fs.existsSync(binary)) {
    if (args.length === 0 || args[0] === '--help' || args[0] === '-h') {
      console.log('Fanloop npm bootstrap\n\n  fanloop install  Install this package\n  fanloop update   Install the official latest package\n\nAfter installation, use loop --help for workflow commands.');
      return;
    }
    throw new Error('Fanloop is not installed. Run: NPM_CONFIG_REGISTRY=' + registry + ' npx --yes --package=' + packageName + '@latest -- fanloop install');
  }
  if (args.length === 0 || args[0] === '--help' || args[0] === '-h') console.log('Npm bootstrap commands: fanloop install | fanloop update\n');
  exitWithChild(spawnSync(binary, args, { env, stdio: 'inherit' }));
}
if (require.main === module) {
  try { main(); } catch (error) { console.error('Fanloop: ' + error.message); process.exit(1); }
}
module.exports = { main, installLauncher };
