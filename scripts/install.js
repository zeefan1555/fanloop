#!/usr/bin/env node
// Bootstrap/archive safeguards adapted from webcast/treeloop_cli d02050d.
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { execFileSync, spawnSync } = require('node:child_process');

function dataHome(env = process.env) {
  return path.resolve(env.FANLOOP_DATA_HOME || path.join(os.homedir(), '.fanloop'));
}
function checksum(file) {
  return crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
}
function stableVersion(value) {
  return /^\d+\.\d+\.\d+$/.test(value) ? value.split('.').map(Number) : null;
}
function installedVersion(env) {
  try {
    fs.accessSync(path.join(dataHome(env), 'current/bin/fanloop'), fs.constants.X_OK);
    return JSON.parse(fs.readFileSync(path.join(dataHome(env), 'current/release.json'), 'utf8')).release_version;
  } catch { return ''; }
}
function newerInstalled(env, candidate) {
  if (!env.FANLOOP_UPDATE_FORWARD_ONLY) return '';
  const installed = installedVersion(env);
  const a = stableVersion(installed), b = stableVersion(candidate);
  if (!a || !b) return '';
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return a[i] > b[i] ? installed : '';
  }
  return '';
}
function runPayload(binary, source, expectedVersion, env) {
  const args = ['__install', '--source', source, '--data-root', dataHome(env), '--replace-invalid'];
  if (env.FANLOOP_CONFIG_SOURCE) args.push('--config-source', path.resolve(env.FANLOOP_CONFIG_SOURCE));
  for (const [flag, variable, directory] of [
    ['codex', 'FANLOOP_CODEX_SKILLS_ROOT', '.codex'],
    ['agent', 'FANLOOP_AGENT_SKILLS_ROOT', '.agents'],
    ['trae', 'FANLOOP_TRAE_SKILLS_ROOT', '.trae'],
    ['claude', 'FANLOOP_CLAUDE_SKILLS_ROOT', '.claude'],
  ]) args.push('--' + flag + '-skills-root', path.resolve(env[variable] || path.join(os.homedir(), directory, 'skills')));
  const result = spawnSync(binary, args, { env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
  if (result.error || result.status !== 0) throw new Error(result.error?.message || result.stderr.trim() || 'Payload installation failed');
  if (installedVersion(env) !== expectedVersion) throw new Error('Installer did not activate the expected release');
}
function validateArchive(archive) {
  const options = { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] };
  const names = execFileSync('tar', ['-tf', archive], options).trim().split(/\r?\n/);
  const details = execFileSync('tar', ['-tvf', archive], options).trim().split(/\r?\n/);
  if (!names[0] || names.length !== details.length) throw new Error('Empty or ambiguous archive');
  const seen = new Set();
  names.forEach((listed, index) => {
    const name = listed.replace(/^\.\//, '').replace(/\/$/, '');
    if (!name || name.includes('\\') || path.posix.isAbsolute(name) || path.posix.normalize(name) !== name || name === '..' || name.startsWith('../')) {
      throw new Error('Unsafe archive path: ' + listed);
    }
    if (!'-d'.includes(details[index][0]) || seen.has(name)) throw new Error('Unsupported or duplicate archive entry: ' + listed);
    seen.add(name);
  });
  if (!seen.has('bin/fanloop') || !seen.has('release.json')) throw new Error('Archive is missing its binary or manifest');
}
function install(env = process.env, packageRoot = path.resolve(__dirname, '..'), prepareLauncher = () => {}) {
  const metadata = JSON.parse(fs.readFileSync(path.join(packageRoot, 'package.json'), 'utf8'));
  const version = metadata.version;
  if (metadata.name !== '@zeefan1555/fanloop-cli' || !stableVersion(version)) throw new Error('Invalid npm package identity/version');
  const kept = newerInstalled(env, version);
  if (kept) {
    const current = path.join(dataHome(env), 'current');
    runPayload(path.join(current, 'bin/fanloop'), current, kept, env);
    return { version: kept, kept: true };
  }
  const platform = { darwin: 'darwin', linux: 'linux' }[process.platform];
  const arch = { x64: 'amd64', arm64: 'arm64' }[process.arch];
  if (!platform || !arch) throw new Error('Unsupported platform: ' + process.platform + '/' + process.arch);
  const name = 'fanloop-' + version + '-' + platform + '-' + arch + '.tar.gz';
  const archive = path.join(packageRoot, 'releases', name);
  if (!fs.lstatSync(archive).isFile()) throw new Error('Archive must be a regular file');
  const records = new Map();
  for (const line of fs.readFileSync(path.join(packageRoot, 'checksums.txt'), 'utf8').trim().split('\n')) {
    const match = /^([a-f0-9]{64})  ([^/\\\s]+)$/.exec(line);
    if (!match || records.has(match[2])) throw new Error('Invalid checksum list');
    records.set(match[2], match[1]);
  }
  if (records.get(name) !== checksum(archive)) throw new Error('Archive checksum mismatch');
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-install-'));
  try {
    validateArchive(archive);
    execFileSync('tar', ['-xf', archive, '-C', temporary], { stdio: ['ignore', 'ignore', 'pipe'] });
    const manifest = JSON.parse(fs.readFileSync(path.join(temporary, 'release.json'), 'utf8'));
    const binary = path.join(temporary, 'bin', 'fanloop');
    if (manifest.schema_version !== 4 || manifest.release_version !== version || manifest.cli?.version !== version) throw new Error('Package and manifest versions do not match');
    if (manifest.cli.binary_sha256 !== 'sha256:' + checksum(binary)) throw new Error('Binary checksum mismatch');
    prepareLauncher();
    runPayload(binary, temporary, version, env);
    return { version, kept: false };
  } finally { fs.rmSync(temporary, { recursive: true, force: true }); }
}
function shellQuote(value) { return "'" + value.replace(/'/g, "'\\''") + "'"; }
function ensureLauncherPath(bin, env = process.env) {
  if (/[\r\n]/.test(bin)) throw new Error('Invalid npm bin path');
  if ((env.PATH || '').split(path.delimiter)[0] === bin) return;
  const line = 'export PATH=' + shellQuote(bin) + ':"$PATH"';
  const home = env.HOME || os.homedir();
  const shell = path.basename(env.SHELL || '');
  const files = shell === 'zsh' ? [path.join(env.ZDOTDIR || home, '.zshrc'), path.join(env.ZDOTDIR || home, '.zprofile')]
    : shell === 'bash' ? [path.join(home, '.bashrc'), path.join(home, '.bash_profile')] : [];
  for (const file of files) {
    try {
      const content = fs.existsSync(file) ? fs.readFileSync(file, 'utf8') : '';
      const retained = content.split('\n').filter(item => item !== line && item !== '# Fanloop npm launcher').join('\n').trimEnd();
      const updated = retained + '\n# Fanloop npm launcher\n' + line + '\n';
      if (updated !== content) {
        fs.mkdirSync(path.dirname(file), { recursive: true });
        fs.writeFileSync(file, updated);
      }
    } catch { console.error('Could not update ' + file + '; configure PATH manually.'); }
  }
  console.log('For this terminal, run: ' + line);
}
function isUnsafeSudo(env = process.env) {
  return process.geteuid?.() === 0 && env.SUDO_USER && env.SUDO_USER !== 'root' && /^[1-9]\d*$/.test(env.SUDO_UID || '');
}
module.exports = { dataHome, checksum, stableVersion, installedVersion, newerInstalled, validateArchive, install, ensureLauncherPath, shellQuote, isUnsafeSudo };
