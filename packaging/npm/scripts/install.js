#!/usr/bin/env node
'use strict';

const {spawnSync} = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const layout = require('./layout');

const DOWNLOAD_LIMIT = 256 * 1024 * 1024;
const DOWNLOAD_TIMEOUT = 30000;
const RETRY_DELAYS = [100, 200, 400];

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

function downloadFailure(message) {
  const err = new Error(message);
  err.downloadFailure = true;
  return err;
}

function checkedDownloadURL(value, name, base) {
  let url;
  try {
    url = new URL(value, base);
  } catch {
    throw downloadFailure(`download ${name} failed: invalid URL`);
  }
  const loopback = ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname);
  if (url.username || url.password || (url.protocol !== 'https:' && !(url.protocol === 'http:' && loopback))) {
    throw downloadFailure(`download ${name} failed: unsafe URL`);
  }
  return url;
}

function transientDownloadError(err, bodyStage, signal) {
  // Older fetch implementations surface deadline cancellation as AbortError.
  if (signal.aborted && signal.reason?.name === 'TimeoutError') return true;
  const seen = new Set();
  for (let cause = err; cause && !seen.has(cause); cause = cause.cause) {
    seen.add(cause);
    if (cause.name === 'TimeoutError' || ['UND_ERR_CONNECT_TIMEOUT', 'UND_ERR_HEADERS_TIMEOUT', 'UND_ERR_BODY_TIMEOUT'].includes(cause.code)) {
      return true;
    }
    if (bodyStage && ['UND_ERR_SOCKET', 'UND_ERR_RES_CONTENT_LENGTH_MISMATCH'].includes(cause.code)) {
      return true;
    }
  }
  return false;
}

async function download(url, name, limit = DOWNLOAD_LIMIT) {
  const initial = checkedDownloadURL(url, name);
  for (let attempt = 0; attempt <= RETRY_DELAYS.length; attempt += 1) {
    const signal = AbortSignal.timeout(DOWNLOAD_TIMEOUT);
    let bodyStage = false;
    try {
      let current = initial;
      let response;
      for (let redirects = 0; ; redirects += 1) {
        response = await fetch(current, {headers: {'User-Agent': 'fanloop-install'}, redirect: 'manual', signal});
        if (![301, 302, 303, 307, 308].includes(response.status)) break;
        if (response.body) await response.body.cancel().catch(() => {});
        const location = response.headers.get('location');
        if (!location || redirects >= 5) {
          throw downloadFailure(`download ${name} failed: invalid or excessive redirects`);
        }
        current = checkedDownloadURL(location, name, current);
      }
      if (!response.ok) {
        if (response.body) await response.body.cancel().catch(() => {});
        if (response.status === 404) {
          throw downloadFailure(`download ${name} failed: HTTP 404. Fanloop releases require a public repository or authentication; this installer does not send a token.`);
        }
        throw downloadFailure(`download ${name} failed: HTTP ${response.status}`);
      }
      if (!response.body) throw downloadFailure(`download ${name} failed: empty response`);
      const reader = response.body.getReader();
      const chunks = [];
      let total = 0;
      bodyStage = true;
      try {
        while (true) {
          const {done, value} = await reader.read();
          if (done) break;
          total += value.byteLength;
          if (total > limit) throw downloadFailure(`download ${name} exceeds ${limit} bytes`);
          chunks.push(Buffer.from(value));
        }
      } finally {
        await reader.cancel().catch(() => {});
        reader.releaseLock();
      }
      return Buffer.concat(chunks);
    } catch (err) {
      if (err.downloadFailure) throw err;
      if (!transientDownloadError(err, bodyStage, signal)) {
        throw downloadFailure(`download ${name} failed: non-retryable transport error`);
      }
      if (attempt === RETRY_DELAYS.length) {
        throw downloadFailure(`download ${name} failed: transient transport error after 4 attempts`);
      }
      await new Promise((resolve) => setTimeout(resolve, RETRY_DELAYS[attempt]));
    }
  }
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
  synchronizeInstalledConfiguration(dataHome, options.env || process.env);
  return {version, dataHome};
}

function configurationSyncSkipped(env) {
  const flag = String(env.FANLOOP_NO_AUTO_CONFIG ?? '').trim().toLowerCase();
  if (flag === '1' || flag === 'true' || flag === 'yes') {
    return true;
  }
  return Object.prototype.hasOwnProperty.call(env, 'FANLOOP_CONFIG_SOURCE');
}

function synchronizeInstalledConfiguration(dataHome, env) {
  if (configurationSyncSkipped(env)) {
    return;
  }
  const binary = path.join(dataHome, 'current', 'bin', 'fanloop');
  const result = spawnSync(binary, ['update', 'config'], {
    stdio: 'inherit',
    timeout: 120000,
    env,
  });
  if (result.error || result.status !== 0) {
    console.error('fanloop install: configuration sync failed; run "fanloop update config"');
  }
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
  layout.ensureLauncherPath(installed.dataHome);
  reportInstall(installed.dataHome, installed.version);
}

if (require.main === module) {
  main().catch((err) => {
    console.error(`fanloop install: ${err.message}`);
    process.exit(1);
  });
}

module.exports = {DOWNLOAD_LIMIT, download, extractArchive, installRelease, readReleaseFile, reportInstall};
