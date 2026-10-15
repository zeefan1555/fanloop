#!/usr/bin/env node
'use strict';

const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const layout = require('./layout');

const DOWNLOAD_LIMIT = 256 * 1024 * 1024;

function treeSnapshot(root) {
  const lines = [];
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, {withFileTypes: true})) {
      const full = path.join(dir, entry.name);
      const relative = path.relative(root, full).split(path.sep).join('/');
      if (entry.isSymbolicLink() || (!entry.isFile() && !entry.isDirectory())) {
        lines.push(`${relative}\0other`);
        continue;
      }
      if (entry.isDirectory()) {
        walk(full);
        continue;
      }
      const mode = (fs.statSync(full).mode & 0o111) !== 0 ? 'x' : '-';
      const hash = crypto.createHash('sha256').update(fs.readFileSync(full)).digest('hex');
      lines.push(`${relative}\0${hash}\0${mode}`);
    }
  };
  walk(root);
  lines.sort();
  return lines.join('\n');
}

async function extractArchive(bytes, dataHome, version) {
  layout.assertReleaseArchive(bytes);
  const releases = path.join(dataHome, 'releases');
  await fs.promises.mkdir(releases, {recursive: true, mode: 0o755});
  const finalDir = path.join(releases, version);
  const staging = await fs.promises.mkdtemp(path.join(releases, '.extract-'));
  let moved = false;
  try {
    layout.extractReleaseArchive(bytes, staging);
    layout.validateReleaseTree(staging, version);
    let existing;
    try {
      existing = fs.lstatSync(finalDir);
    } catch (err) {
      if (err.code !== 'ENOENT') {
        throw err;
      }
    }
    if (existing) {
      if (!existing.isDirectory() || existing.isSymbolicLink()) {
        throw new Error(`refusing to replace immutable release ${version}`);
      }
      if (treeSnapshot(finalDir) === treeSnapshot(staging)) {
        return;
      }
      const backup = path.join(releases, `.backup-${version}-${crypto.randomBytes(4).toString('hex')}`);
      await fs.promises.rename(finalDir, backup);
      try {
        await fs.promises.rename(staging, finalDir);
        moved = true;
      } catch (err) {
        await fs.promises.rename(backup, finalDir);
        throw err;
      }
      await fs.promises.rm(backup, {recursive: true, force: true});
      return;
    }
    await fs.promises.rename(staging, finalDir);
    moved = true;
  } finally {
    if (!moved) {
      await fs.promises.rm(staging, {recursive: true, force: true});
    }
  }
}

function releaseBase(version) {
  return `https://github.com/zeefan1555/fanloop/releases/download/v${version}`;
}

async function download(url, name, limit = DOWNLOAD_LIMIT) {
  const response = await fetch(url, {headers: {'User-Agent': 'fanloop-install'}});
  if (response.status === 404) {
    throw new Error(`download ${name} failed: HTTP 404. Fanloop releases require a public repository or authentication; this installer does not send a token.`);
  }
  if (!response.ok) {
    throw new Error(`download ${name} failed: ${response.status}`);
  }
  if (!response.body) {
    throw new Error(`download ${name} failed: empty response`);
  }
  const reader = response.body.getReader();
  const chunks = [];
  let total = 0;
  while (true) {
    const {done, value} = await reader.read();
    if (done) {
      break;
    }
    total += value.byteLength;
    if (total > limit) {
      await reader.cancel();
      throw new Error(`download ${name} exceeds ${limit} bytes`);
    }
    chunks.push(Buffer.from(value));
  }
  return Buffer.concat(chunks);
}

async function readReleaseFile(name, options) {
  if (options.releaseDir) {
    const bytes = await fs.promises.readFile(path.join(options.releaseDir, name));
    if (bytes.length > DOWNLOAD_LIMIT) {
      throw new Error(`download ${name} exceeds ${DOWNLOAD_LIMIT} bytes`);
    }
    return bytes;
  }
  const base = options.releaseBase || releaseBase(options.version);
  return download(`${base}/${name}`, name, options.downloadLimit);
}

async function installRelease(options) {
  const version = options.version;
  const platform = options.platform;
  const dataHome = options.dataHome;
  if (!layout.validVersion(version) || version === '0.0.0') {
    throw new Error(`package version is not a release version: ${version}`);
  }
  layout.assertCurrentReplaceable(dataHome);
  const name = layout.assetName(version, platform);
  const readFile = options.readFile || ((filename) => readReleaseFile(filename, options));
  const checksumBytes = await readFile('checksums.txt');
  const manifestBytes = await readFile('release.json');
  const archive = await readFile(name);
  const checksums = checksumBytes.toString('utf8');
  const archiveHash = layout.verifyChecksum(checksums, name, archive);
  layout.verifyChecksum(checksums, 'release.json', manifestBytes);
  const manifest = JSON.parse(manifestBytes.toString('utf8'));
  if (manifest.schema_version !== 1 || manifest.version !== version || manifest.tag !== `v${version}`) {
    throw new Error('release.json does not match this package version');
  }
  const listed = (manifest.assets || []).find((asset) => asset.name === name);
  if (!listed || listed.sha256 !== archiveHash || listed.os !== platform.os || listed.arch !== platform.arch) {
    throw new Error(`release.json does not match checksums.txt for ${name}`);
  }
  await extractArchive(archive, dataHome, version);
  const roots = options.skillRoots || layout.skillRootsFrom(options.env || process.env);
  const created = layout.ensureSkillLinks(dataHome, roots);
  try {
    layout.switchCurrent(dataHome, version);
  } catch (err) {
    for (const link of created) {
      fs.rmSync(link, {force: true});
    }
    throw err;
  }
  return {version, dataHome};
}

function reportInstall(dataHome, version, env = process.env) {
  const binary = path.join(dataHome, 'current', 'bin', 'fanloop');
  console.log(`installed fanloop ${version}`);
  console.log(binary);
  const pathStatus = layout.pathInstruction(dataHome, env);
  if (!pathStatus.onPath) {
    console.error('fanloop is not on PATH. Add the binary with:');
    console.error(pathStatus.instruction);
  }
}

async function main() {
  const pkg = require('../package.json');
  const version = pkg.version;
  const installed = await installRelease({
    version,
    dataHome: layout.dataHome(),
    platform: layout.selectPlatform(process.platform, process.arch),
    releaseDir: process.env.FANLOOP_RELEASE_DIR || '',
  });
  reportInstall(installed.dataHome, installed.version);
}

if (require.main === module) {
  main().catch((err) => {
    console.error(`fanloop install: ${err.message}`);
    process.exit(1);
  });
}

module.exports = {DOWNLOAD_LIMIT, download, extractArchive, installRelease, readReleaseFile, reportInstall};
