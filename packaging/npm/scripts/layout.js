'use strict';

const crypto = require('crypto');
const fs = require('fs');
const {spawnSync} = require('child_process');
const os = require('os');
const path = require('path');
const zlib = require('zlib');

const PLATFORMS = [
  {os: 'darwin', arch: 'amd64'},
  {os: 'darwin', arch: 'arm64'},
  {os: 'linux', arch: 'amd64'},
  {os: 'linux', arch: 'arm64'},
];

function selectPlatform(platform, arch) {
  const osName = platform === 'darwin' || platform === 'linux' ? platform : '';
  const archName = arch === 'x64' || arch === 'amd64' ? 'amd64' : arch === 'arm64' ? 'arm64' : '';
  if (!osName || !archName) {
    throw new Error(`unsupported platform ${platform}/${arch}; fanloop publishes darwin and linux for amd64 and arm64`);
  }
  return {os: osName, arch: archName};
}

function validVersion(version) {
  return /^[0-9]+\.[0-9]+\.[0-9]+$/.test(version);
}

function assetName(version, platform) {
  if (!validVersion(version)) {
    throw new Error(`invalid release version: ${version}`);
  }
  selectPlatform(platform.os, platform.arch);
  return `fanloop-${version}-${platform.os}-${platform.arch}.tar.gz`;
}

function npmPackageName(version) {
  if (!validVersion(version)) {
    throw new Error(`invalid release version: ${version}`);
  }
  return `fanloop-${version}.tgz`;
}

function parseChecksums(text) {
  const entries = new Map();
  for (const raw of String(text).split(/\r?\n/)) {
    if (!raw.trim()) {
      continue;
    }
    const match = /^([a-f0-9]{64})\s+\*?(\S+)$/i.exec(raw.trim());
    if (!match) {
      throw new Error(`invalid checksums.txt line: ${raw}`);
    }
    const name = match[2];
    if (name.includes('/') || name.includes('\\') || name === '.' || name === '..') {
      throw new Error(`invalid checksum filename: ${name}`);
    }
    if (entries.has(name)) {
      throw new Error(`duplicate checksum entry: ${name}`);
    }
    entries.set(name, match[1].toLowerCase());
  }
  return entries;
}

function verifyChecksum(text, filename, bytes) {
  const expected = parseChecksums(text).get(filename);
  if (!expected) {
    throw new Error(`checksums.txt has no entry for ${filename}`);
  }
  const actual = crypto.createHash('sha256').update(bytes).digest('hex');
  if (!crypto.timingSafeEqual(Buffer.from(actual, 'hex'), Buffer.from(expected, 'hex'))) {
    throw new Error(`sha256 mismatch for ${filename}`);
  }
  return expected;
}

const EXPOSED_SKILLS = [
  {name: 'fanloop-workflow', path: 'entrypoints/fanloop-workflow'},
  {name: 'create-fanloop-workflow', path: 'entrypoints/create-fanloop-workflow'},
];

const ARCHIVE_REQUIRED = [
  'bin/fanloop',
  'release.json',
  'entrypoints/fanloop-workflow/SKILL.md',
  'entrypoints/create-fanloop-workflow/SKILL.md',
];

const RELEASE_VERSION_PATTERN = /^[0-9A-Za-z][0-9A-Za-z._+-]*$/;
const DIGEST_PATTERN = /^sha256:[0-9a-f]{64}$/;
const SKILL_NAME_PATTERN = /^[a-z][a-z0-9-]*$/;

function dataHome(env = process.env) {
  if (env.FANLOOP_DATA_HOME) {
    if (!path.isAbsolute(env.FANLOOP_DATA_HOME)) {
      throw new Error('FANLOOP_DATA_HOME must be absolute');
    }
    return path.resolve(env.FANLOOP_DATA_HOME);
  }
  return path.join(os.homedir(), '.fanloop');
}

function skillRootsFrom(env = process.env, home = os.homedir()) {
  return [
    env.FANLOOP_CODEX_SKILLS_ROOT || path.join(home, '.codex', 'skills'),
    env.FANLOOP_AGENT_SKILLS_ROOT || path.join(home, '.agents', 'skills'),
    env.FANLOOP_TRAE_SKILLS_ROOT || path.join(home, '.trae', 'skills'),
    env.FANLOOP_CLAUDE_SKILLS_ROOT || path.join(home, '.claude', 'skills'),
  ];
}

function fileDigest(file) {
  const info = fs.lstatSync(file);
  if (!info.isFile()) {
    throw new Error(`${file} is not a regular file`);
  }
  return `sha256:${crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex')}`;
}

function directoryDigest(root) {
  const paths = [];
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, {withFileTypes: true})) {
      const full = path.join(dir, entry.name);
      if (entry.isSymbolicLink()) {
        continue;
      }
      if (entry.isDirectory()) {
        walk(full);
        continue;
      }
      if (entry.isFile()) {
        paths.push(path.relative(root, full).split(path.sep).join('/'));
      }
    }
  };
  walk(root);
  paths.sort();
  const hash = crypto.createHash('sha256');
  for (const relative of paths) {
    const digest = fileDigest(path.join(root, ...relative.split('/'))).slice('sha256:'.length);
    hash.update(relative);
    hash.update(Buffer.from([0]));
    hash.update(digest);
    hash.update('\n');
  }
  return `sha256:${hash.digest('hex')}`;
}

function rejectUnknown(value, allowed, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label} is missing`);
  }
  for (const key of Object.keys(value)) {
    if (!allowed.includes(key)) {
      throw new Error(`${label} has unknown field ${key}`);
    }
  }
}

function validateReleaseTree(root, version) {
  const manifest = JSON.parse(fs.readFileSync(path.join(root, 'release.json'), 'utf8'));
  rejectUnknown(manifest, ['schema_version', 'release_version', 'cli', 'state_schema', 'skills'], 'release.json');
  if (manifest.schema_version !== 4) {
    throw new Error('release.json schema_version must be 4');
  }
  if (!RELEASE_VERSION_PATTERN.test(manifest.release_version || '') || manifest.release_version !== version) {
    throw new Error(`release.json version does not match ${version}`);
  }
  rejectUnknown(manifest.cli, ['version', 'binary_sha256'], 'release.json cli');
  if (manifest.cli.version !== version || !RELEASE_VERSION_PATTERN.test(manifest.cli.version)) {
    throw new Error('CLI version must match release version');
  }
  if (!DIGEST_PATTERN.test(manifest.cli.binary_sha256 || '')) {
    throw new Error('release.json binary_sha256 is invalid');
  }
  const binary = fileDigest(path.join(root, 'bin', 'fanloop'));
  if (binary !== manifest.cli.binary_sha256) {
    throw new Error('binary checksum does not match release.json');
  }
  rejectUnknown(manifest.state_schema, ['read_versions', 'write_version'], 'release.json state_schema');
  const reads = manifest.state_schema.read_versions;
  const write = manifest.state_schema.write_version;
  if (!Array.isArray(reads) || reads.length === 0 || !reads.every((item) => Number.isInteger(item))) {
    throw new Error('release.json state_schema read_versions is invalid');
  }
  if (!Number.isInteger(write) || write <= 0 || !reads.includes(write)) {
    throw new Error('writable state schema must also be readable');
  }
  if (!Array.isArray(manifest.skills) || manifest.skills.length < 1) {
    throw new Error('release.json skills are incomplete');
  }
  const seen = new Set();
  for (const skill of manifest.skills) {
    rejectUnknown(skill, ['name', 'version', 'path', 'sha256'], 'release.json skill');
    const exposed = EXPOSED_SKILLS.find((item) => item.name === skill.name && item.path === skill.path);
    if (!exposed || seen.has(skill.name) || !SKILL_NAME_PATTERN.test(skill.name) || !RELEASE_VERSION_PATTERN.test(skill.version || '')) {
      throw new Error(`invalid or duplicate skill ${skill.name}`);
    }
    if (!DIGEST_PATTERN.test(skill.sha256 || '')) {
      throw new Error(`skill ${skill.name} sha256 is invalid`);
    }
    const digest = directoryDigest(path.join(root, ...skill.path.split('/')));
    if (digest !== skill.sha256) {
      throw new Error(`skill ${skill.name} checksum does not match release.json`);
    }
    seen.add(skill.name);
  }
  for (const exposed of EXPOSED_SKILLS) {
    if (!seen.has(exposed.name)) {
      throw new Error(`release is missing exposed Skill ${exposed.name}`);
    }
  }
  return manifest;
}

function fieldText(header, start, end) {
  return header.subarray(start, end).toString('utf8').replace(/\0.*$/, '').trim();
}

function checksumMatches(header) {
  const stored = parseInt(fieldText(header, 148, 156), 8);
  if (!Number.isFinite(stored)) {
    return false;
  }
  let unsigned = 0;
  let signed = 0;
  for (let index = 0; index < header.length; index += 1) {
    const value = index >= 148 && index < 156 ? 0x20 : header[index];
    unsigned += value;
    signed += value < 128 ? value : value - 256;
  }
  return stored === unsigned || stored === signed;
}

function isArchiveMetadata(name) {
  return name.split('/').some((part) => part === 'PaxHeader' || part === 'PaxHeaders' || part.startsWith('PaxHeaders.') || part.startsWith('._'));
}

function archiveParts(name) {
  const trimmed = name.replace(/^\.\/+/, '').replace(/\/+$/, '');
  const parts = trimmed.split('/');
  if (!trimmed || name.startsWith('/') || parts.some((part) => part === '' || part === '.' || part === '..')) {
    throw new Error(`archive path escapes the release: ${name}`);
  }
  return parts;
}

function safeArchivePath(root, parts) {
  const target = path.resolve(root, ...parts);
  const relative = path.relative(path.resolve(root), target);
  if (relative === '' || relative.startsWith('..') || path.isAbsolute(relative)) {
    throw new Error(`archive path escapes the release: ${parts.join('/')}`);
  }
  return target;
}

function readReleaseArchive(bytes, destination) {
  let raw;
  try {
    raw = zlib.gunzipSync(bytes);
  } catch (err) {
    throw new Error(`read release archive: ${err.message}`);
  }
  const seen = new Set();
  let offset = 0;
  while (offset + 512 <= raw.length) {
    const header = raw.subarray(offset, offset + 512);
    if (header.every((byte) => byte === 0)) {
      break;
    }
    if (!checksumMatches(header)) {
      throw new Error('archive header checksum is invalid');
    }
    const nameField = header.subarray(0, 100).toString('utf8').replace(/\0.*$/, '');
    const magic = header.subarray(257, 262).toString('utf8');
    const prefix = magic.startsWith('ustar') ? header.subarray(345, 500).toString('utf8').replace(/\0.*$/, '') : '';
    const name = prefix ? `${prefix}/${nameField}` : nameField;
    const type = String.fromCharCode(header[156]);
    const size = parseInt(fieldText(header, 124, 136), 8);
    if (!Number.isFinite(size) || size < 0) {
      throw new Error(`archive entry ${name} has an invalid size`);
    }
    const dataStart = offset + 512;
    const dataEnd = dataStart + size;
    if (dataEnd > raw.length) {
      throw new Error(`archive entry ${name} is truncated`);
    }
    const data = raw.subarray(dataStart, dataEnd);
    offset = dataStart + (Math.ceil(size / 512) * 512);
    if (!name || name === '.' || name === './') {
      continue;
    }
    if (type === 'x' || type === 'g') {
      continue;
    }
    const parts = archiveParts(name);
    const cleanName = parts.join('/');
    if (isArchiveMetadata(cleanName)) {
      if (type !== '0' && type !== '\0' && type !== '5') {
        throw new Error(`archive entry ${name} is not a regular file`);
      }
      continue;
    }
    if (type === '5') {
      if (destination) {
        fs.mkdirSync(safeArchivePath(destination, parts), {recursive: true});
      }
      continue;
    }
    if (type !== '0' && type !== '\0') {
      throw new Error(`archive entry ${name} is not a regular file`);
    }
    if (seen.has(cleanName)) {
      throw new Error(`duplicate archive entry ${name}`);
    }
    seen.add(cleanName);
    if (destination) {
      const target = safeArchivePath(destination, parts);
      fs.mkdirSync(path.dirname(target), {recursive: true});
      const mode = parseInt(fieldText(header, 100, 108), 8);
      fs.writeFileSync(target, data, {flag: 'wx', mode: Number.isFinite(mode) && (mode & 0o111) !== 0 ? 0o755 : 0o644});
    }
  }
  for (const required of ARCHIVE_REQUIRED) {
    if (!seen.has(required)) {
      throw new Error(`archive is missing ${required}`);
    }
  }
}

function assertReleaseArchive(bytes) {
  readReleaseArchive(bytes, null);
}

function extractReleaseArchive(bytes, destination) {
  readReleaseArchive(bytes, destination);
}

function sameFilePath(left, right) {
  return path.resolve(left) === path.resolve(right);
}

function ensureSkillLinks(dataHome, roots) {
  const created = [];
  const rollback = () => {
    for (const link of created) {
      fs.rmSync(link, {force: true});
    }
  };
  try {
    for (const skill of EXPOSED_SKILLS) {
      const target = path.join(dataHome, 'current', skill.path);
      for (const root of roots) {
        const linkPath = path.join(root, skill.name);
        let info;
        try {
          info = fs.lstatSync(linkPath);
        } catch (err) {
          if (err.code !== 'ENOENT') {
            throw err;
          }
          fs.mkdirSync(path.dirname(linkPath), {recursive: true, mode: 0o755});
          fs.symlinkSync(target, linkPath);
          created.push(linkPath);
          continue;
        }
        if (!info.isSymbolicLink()) {
          throw new Error(`refusing to replace non-Fanloop path ${linkPath}`);
        }
        const actual = fs.readlinkSync(linkPath);
        const resolved = path.isAbsolute(actual) ? actual : path.resolve(path.dirname(linkPath), actual);
        if (!sameFilePath(resolved, target)) {
          throw new Error(`refusing to replace non-Fanloop Skill link ${linkPath}`);
        }
      }
    }
  } catch (err) {
    rollback();
    throw err;
  }
  return created;
}

function assertCurrentReplaceable(dataHome) {
  const current = path.join(dataHome, 'current');
  let info;
  try {
    info = fs.lstatSync(current);
  } catch (err) {
    if (err.code === 'ENOENT') {
      return;
    }
    throw err;
  }
  if (info.isSymbolicLink()) return false;
  if (!info.isDirectory()) throw new Error(`${current} is not a symlink or managed directory`);
  const links = {bin: 'release/bin', entrypoints: 'release/entrypoints', 'release.json': 'release/release.json'};
  const entries = fs.readdirSync(current);
  if (entries.length !== 4 || entries.some((name) => name !== 'release' && !Object.hasOwn(links, name))) throw new Error('current is not a symlink or valid managed directory; refusing unmanaged entries');
  for (const [name, target] of Object.entries(links)) {
    if (path.normalize(fs.readlinkSync(path.join(current, name))) !== path.normalize(target)) throw new Error(`refusing invalid managed current link ${name}`);
  }
  const releaseTarget = path.resolve(current, fs.readlinkSync(path.join(current, 'release')));
  const releases = path.join(dataHome, 'releases');
  const relative = path.relative(releases, releaseTarget);
  if (relative === '.' || relative === '..' || path.dirname(relative) !== '.') throw new Error('refusing invalid managed current release');
  for (const dir of [releases, releaseTarget]) {
    const entry = fs.lstatSync(dir);
    if (!entry.isDirectory() || entry.isSymbolicLink()) throw new Error('refusing invalid managed current release directory');
  }
  return true;
}

function switchCurrent(dataHome, version) {
  if (!validVersion(version)) {
    throw new Error(`invalid release version: ${version}`);
  }
  const releaseName = path.join('releases', version);
  const releaseDir = path.join(dataHome, releaseName);
  const info = fs.lstatSync(releaseDir);
  if (!info.isDirectory() || info.isSymbolicLink()) {
    throw new Error(`release directory is not a real directory: ${releaseDir}`);
  }
  const managed = assertCurrentReplaceable(dataHome);
  const current = path.join(dataHome, 'current');
  if (managed) {
    const result = spawnSync(path.join(releaseDir, 'bin', 'fanloop'), ['__activate-current', '--data-root', dataHome, '--version', version], {stdio: 'pipe'});
    if (result.error || result.status !== 0 || !fs.lstatSync(current).isSymbolicLink() || fs.readlinkSync(current) !== releaseName) {
      throw new Error('managed current activation failed');
    }
    return true;
  }
  try {
    if (fs.readlinkSync(current) === releaseName) {
      return false;
    }
  } catch (err) {
    if (err.code !== 'ENOENT') {
      throw err;
    }
  }
  const tmp = path.join(dataHome, `.current-${crypto.randomBytes(6).toString('hex')}`);
  fs.symlinkSync(releaseName, tmp);
  try {
    fs.renameSync(tmp, current);
  } catch (err) {
    fs.rmSync(tmp, {force: true});
    throw err;
  }
  return true;
}

function absoluteShellPath(value) {
  if (typeof value !== 'string' || !path.isAbsolute(value)) {
    throw new Error('shell PATH configuration paths must be absolute');
  }
  if (/[\x00-\x1f\x7f:]/.test(value)) {
    throw new Error('invalid shell PATH configuration path');
  }
  return path.resolve(value);
}

function launcherBin(dataHome, env) {
  return Object.prototype.hasOwnProperty.call(env, 'npm_config_prefix')
    ? path.join(absoluteShellPath(env.npm_config_prefix), 'bin')
    : path.join(absoluteShellPath(dataHome), 'current', 'bin');
}

function shellExport(bin) {
  return `export PATH='${bin.replace(/'/g, "'\\''")}':"$PATH"`;
}

function ensureLauncherPath(dataHome, env = process.env) {
  const home = absoluteShellPath(Object.prototype.hasOwnProperty.call(env, 'HOME') ? env.HOME : os.homedir());
  const bin = launcherBin(dataHome, env);
  const shell = path.basename(env.SHELL || '');
  let profiles;
  if (shell === 'zsh') {
    const root = Object.prototype.hasOwnProperty.call(env, 'ZDOTDIR') ? absoluteShellPath(env.ZDOTDIR) : home;
    profiles = [path.join(root, '.zshrc'), path.join(root, '.zprofile')];
  } else if (shell === 'bash') {
    const login = ['.bash_profile', '.bash_login', '.profile'].find((name) => fs.existsSync(path.join(home, name))) || '.bash_profile';
    profiles = [path.join(home, '.bashrc'), path.join(home, login)];
  } else if (shell === 'sh') {
    profiles = [path.join(home, '.profile')];
  } else {
    throw new Error('unsupported shell for persistent PATH configuration; use zsh, bash or sh');
  }
  const block = Buffer.from(`# >>> fanloop PATH >>>\n${shellExport(bin)}\n# <<< fanloop PATH <<<\n`);
  // Prepare every edit before writing, so invalid profile types or markers fail first.
  const edits = profiles.map((file) => {
    let original = Buffer.alloc(0);
    let target = file;
    let mode = 0o600;
    let exists = false;
    try {
      const entry = fs.lstatSync(file);
      exists = true;
      if (entry.isSymbolicLink()) target = fs.realpathSync(file);
      const info = fs.statSync(target);
      if (!info.isFile()) throw new Error('profile is not a regular file');
      mode = info.mode & 0o777;
      original = fs.readFileSync(target);
    } catch (err) {
      // A dangling existing symlink is not an absent profile and must remain untouched.
      if (err.code !== 'ENOENT' || exists) throw new Error('shell PATH configuration could not read a profile');
    }
    // Latin-1 keeps regex offsets aligned with the original bytes, including non-UTF8 content.
    const text = original.toString('latin1');
    const starts = [...text.matchAll(/^# >>> fanloop PATH >>>\r?$/gm)];
    const ends = [...text.matchAll(/^# <<< fanloop PATH <<<\r?$/gm)];
    let content;
    if (starts.length || ends.length) {
      if (starts.length !== 1 || ends.length !== 1 || starts[0].index >= ends[0].index) {
        throw new Error('shell PATH configuration has malformed Fanloop markers');
      }
      let end = ends[0].index + ends[0][0].length;
      if (text[end] === '\n') end += 1;
      content = Buffer.concat([original.subarray(0, starts[0].index), block, original.subarray(end)]);
    } else {
      const separator = original.length && original[original.length - 1] !== 10 ? Buffer.from('\n') : Buffer.alloc(0);
      content = Buffer.concat([original, separator, block]);
    }
    return {file: target, original, content, mode};
  });
  for (const {file, original, content, mode} of edits) {
    if (original.equals(content)) continue;
    const temporary = path.join(path.dirname(file), `.fanloop-path-${crypto.randomBytes(6).toString('hex')}`);
    try {
      fs.mkdirSync(path.dirname(file), {recursive: true});
      fs.writeFileSync(temporary, content, {flag: 'wx', mode});
      fs.chmodSync(temporary, mode);
      fs.renameSync(temporary, file);
    } catch {
      throw new Error('shell PATH configuration could not write a profile; CLI installation may already be active');
    } finally {
      fs.rmSync(temporary, {force: true});
    }
  }
  return profiles;
}

function pathInstruction(dataHome, env = process.env) {
  const bin = path.join(dataHome, 'current', 'bin');
  const npmBin = env.npm_config_prefix ? launcherBin(dataHome, env) : '';
  const entries = String(env.PATH || '').split(path.delimiter).filter(Boolean);
  if ((npmBin && entries.includes(npmBin)) || entries.includes(bin)) {
    return {onPath: true, instruction: ''};
  }
  return {onPath: false, instruction: shellExport(launcherBin(dataHome, env))};
}

module.exports = {
  ARCHIVE_REQUIRED,
  EXPOSED_SKILLS,
  PLATFORMS,
  assetName,
  assertCurrentReplaceable,
  assertReleaseArchive,
  extractReleaseArchive,
  dataHome,
  directoryDigest,
  ensureSkillLinks,
  ensureLauncherPath,
  fileDigest,
  npmPackageName,
  parseChecksums,
  pathInstruction,
  selectPlatform,
  skillRootsFrom,
  switchCurrent,
  validVersion,
  validateReleaseTree,
  verifyChecksum,
};
