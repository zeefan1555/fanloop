#!/usr/bin/env node
'use strict';

const {execFileSync} = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const os = require('os');
const path = require('path');
const layout = require('../packaging/npm/scripts/layout');

const NPM_REQUIRED = [
  'package/package.json',
  'package/scripts/run.js',
  'package/scripts/install.js',
  'package/scripts/layout.js',
];

function fail(message) {
  throw new Error(message);
}

function sha256(file) {
  return crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
}

function listedEntries(archive) {
  return execFileSync('tar', ['-tzf', archive], {encoding: 'utf8'})
    .split('\n')
    .filter(Boolean)
    .map((entry) => entry.replace(/^\.\/+/, '').replace(/\/$/, ''));
}

function assertEntries(archive, required) {
  const entries = new Set(listedEntries(archive));
  for (const entry of entries) {
    if (entry.startsWith('/') || entry.split('/').includes('..')) {
      fail(`${path.basename(archive)} contains unsafe path ${entry}`);
    }
  }
  for (const requiredEntry of required) {
    if (!entries.has(requiredEntry)) {
      fail(`${path.basename(archive)} is missing ${requiredEntry}`);
    }
  }
}

function assertPlatformArchive(archive, version) {
  const stage = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-verify-inner-'));
  try {
    layout.extractReleaseArchive(fs.readFileSync(archive), stage);
    layout.validateReleaseTree(stage, version);
  } catch (err) {
    fail(`${path.basename(archive)}: ${err.message}`);
  } finally {
    fs.rmSync(stage, {recursive: true, force: true});
  }
}

function assertNonEmptyMember(archive, entry) {
  const listing = execFileSync('tar', ['-tvzf', archive, '--', entry], {encoding: 'utf8'});
  const line = listing.split('\n').find((item) => item.trim().endsWith(entry));
  const size = line && line.match(/(?:^|\s)(\d+)\s+(?:\d{4}-\d{2}-\d{2}|[A-Z][a-z]{2}\s+\d+)/);
  if (!size || Number(size[1]) <= 0) {
    fail(`${path.basename(archive)} contains an empty ${entry}`);
  }
}

function verifyDirectory(directory) {
  const manifestPath = path.join(directory, 'release.json');
  if (!fs.existsSync(manifestPath)) {
    fail(`missing ${manifestPath}`);
  }
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  if (manifest.schema_version !== 1 || !layout.validVersion(manifest.version)) {
    fail('release.json schema_version or version is invalid');
  }
  if (manifest.tag !== `v${manifest.version}` || manifest.repository !== 'zeefan1555/fanloop') {
    fail('release.json tag or repository does not match the version');
  }
  const expected = new Map();
  for (const platform of layout.PLATFORMS) {
    const name = layout.assetName(manifest.version, platform);
    expected.set(name, {os: platform.os, arch: platform.arch, kind: ''});
  }
  const npmName = layout.npmPackageName(manifest.version);
  expected.set(npmName, {os: '', arch: '', kind: 'npm'});
  if (!Array.isArray(manifest.assets) || manifest.assets.length !== expected.size) {
    fail(`release.json must list ${expected.size} assets`);
  }
  const seen = new Set();
  for (const asset of manifest.assets) {
    const want = expected.get(asset.name);
    if (!want || seen.has(asset.name)) {
      fail(`unexpected release.json asset ${asset.name}`);
    }
    if (!/^[0-9a-f]{64}$/.test(asset.sha256 || '')) {
      fail(`release.json sha256 for ${asset.name} is invalid`);
    }
    if ((asset.os || '') !== want.os || (asset.arch || '') !== want.arch || (asset.kind || '') !== want.kind) {
      fail(`release.json platform metadata for ${asset.name} is invalid`);
    }
    seen.add(asset.name);
  }
  const checksums = layout.parseChecksums(fs.readFileSync(path.join(directory, 'checksums.txt'), 'utf8'));
  const expectedFiles = new Set([...expected.keys(), 'release.json']);
  if (checksums.size !== expectedFiles.size) {
    fail('checksums.txt does not cover exactly the release assets and release.json');
  }
  for (const name of expectedFiles) {
    if (!checksums.has(name)) {
      fail(`checksums.txt is missing ${name}`);
    }
  }
  const present = fs.readdirSync(directory).filter((name) => name !== 'checksums.txt');
  if (present.length !== expectedFiles.size || present.some((name) => !expectedFiles.has(name))) {
    fail(`release directory contains ${present.sort().join(', ')}, want ${[...expectedFiles].sort().join(', ')}`);
  }
  for (const name of expectedFiles) {
    const file = path.join(directory, name);
    const actual = sha256(file);
    if (actual !== checksums.get(name)) {
      fail(`sha256 mismatch for ${name}`);
    }
    const listed = manifest.assets.find((asset) => asset.name === name);
    if (listed && listed.sha256 !== actual) {
      fail(`release.json sha256 does not match ${name}`);
    }
  }
  for (const [name, meta] of expected) {
    const archive = path.join(directory, name);
    if (meta.kind === 'npm') {
      assertEntries(archive, NPM_REQUIRED);
      const packaged = JSON.parse(execFileSync('tar', ['-xOzf', archive, 'package/package.json'], {encoding: 'utf8'}));
      if (packaged.name !== 'fanloop' || packaged.version !== manifest.version) {
        fail('npm package name or version does not match release.json');
      }
      continue;
    }
    assertPlatformArchive(archive, manifest.version);
    assertNonEmptyMember(archive, 'bin/fanloop');
  }
  return manifest;
}

async function verifyGitHubTag(tag) {
  if (!/^v[0-9]+\.[0-9]+\.[0-9]+$/.test(tag)) {
    fail(`invalid release tag ${tag}`);
  }
  const response = await fetch(`https://api.github.com/repos/zeefan1555/fanloop/releases/tags/${tag}`, {
    headers: {'User-Agent': 'fanloop-verify', Accept: 'application/vnd.github+json'},
  });
  if (response.status === 404) {
    fail(`GitHub release ${tag} lookup failed: HTTP 404. Fanloop releases require a public repository or authentication; this verifier does not send a token.`);
  }
  if (!response.ok) {
    fail(`GitHub release ${tag} lookup failed: ${response.status}`);
  }
  const payload = await response.json();
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-release-'));
  for (const asset of payload.assets || []) {
    if (!asset.name || asset.name.includes('/') || asset.name.includes('..') || !asset.browser_download_url) {
      fail(`GitHub release has an unsafe asset ${asset.name}`);
    }
    const download = await fetch(asset.browser_download_url, {headers: {'User-Agent': 'fanloop-verify'}});
    if (!download.ok) {
      fail(`download ${asset.name} failed: ${download.status}`);
    }
    fs.writeFileSync(path.join(directory, asset.name), Buffer.from(await download.arrayBuffer()));
  }
  return verifyDirectory(directory);
}

async function main(argv) {
  if (argv[0] === '--github' && argv[1] && !argv[2]) {
    const manifest = await verifyGitHubTag(argv[1]);
    console.log(`release ${manifest.tag} assets are complete`);
    return;
  }
  if (argv[0] && !argv[0].startsWith('-') && !argv[1]) {
    const manifest = verifyDirectory(path.resolve(argv[0]));
    console.log(`release ${manifest.tag} assets are complete`);
    return;
  }
  fail('usage: verify-release-assets.js <asset-directory> | --github vX.Y.Z');
}

if (require.main === module) {
  main(process.argv.slice(2)).catch((err) => {
    console.error(`fanloop release verify: ${err.message}`);
    process.exit(1);
  });
}

module.exports = {verifyDirectory, verifyGitHubTag};
