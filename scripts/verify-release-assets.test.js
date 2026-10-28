'use strict';

const assert = require('node:assert/strict');
const {execFileSync} = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const os = require('os');
const path = require('path');
const test = require('node:test');
const layout = require('../packaging/npm/scripts/layout');
const {verifyDirectory} = require('./verify-release-assets');

function sha(bytes) {
  return crypto.createHash('sha256').update(bytes).digest('hex');
}

function writeArchive(file, entries) {
  const stage = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-verify-stage-'));
  for (const [name, body] of Object.entries(entries)) {
    const target = path.join(stage, name);
    fs.mkdirSync(path.dirname(target), {recursive: true});
    fs.writeFileSync(target, body);
  }
  execFileSync('tar', ['-czf', file, '-C', stage, ...Object.keys(entries).map((name) => name.split('/')[0]).filter((value, index, all) => all.indexOf(value) === index)]);
  fs.rmSync(stage, {recursive: true, force: true});
}

function platformArchive(file, version, osName, arch, incomplete) {
  const stage = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-verify-platform-'));
  fs.mkdirSync(path.join(stage, 'bin'), {recursive: true});
  fs.mkdirSync(path.join(stage, 'entrypoints', 'fanloop-workflow'), {recursive: true});
  fs.mkdirSync(path.join(stage, 'entrypoints', 'create-fanloop-workflow'), {recursive: true});
  fs.writeFileSync(path.join(stage, 'bin', 'fanloop'), `binary-${osName}-${arch}`, {mode: 0o755});
  fs.writeFileSync(path.join(stage, 'entrypoints', 'fanloop-workflow', 'SKILL.md'), '# workflow\n');
  fs.writeFileSync(path.join(stage, 'entrypoints', 'create-fanloop-workflow', 'SKILL.md'), '# author\n');
  const manifest = incomplete
    ? {schema_version: 4, release_version: version}
    : {
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
  execFileSync('tar', ['-czf', file, '-C', stage, 'bin', 'entrypoints', 'release.json']);
  fs.rmSync(stage, {recursive: true, force: true});
}

function fixture(mutate, incomplete = false, missingNpmEntry) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-verify-'));
  const version = '1.2.3';
  const files = {};
  for (const [osName, arch] of [['darwin', 'amd64'], ['darwin', 'arm64'], ['linux', 'amd64'], ['linux', 'arm64']]) {
    const name = `fanloop-${version}-${osName}-${arch}.tar.gz`;
    platformArchive(path.join(root, name), version, osName, arch, incomplete);
    files[name] = fs.readFileSync(path.join(root, name));
  }
  const npmEntries = {
    'package/package.json': `${JSON.stringify({name: 'fanloop', version})}\n`,
    'package/scripts/run.js': '#!/usr/bin/env node\n',
    'package/scripts/install.js': '#!/usr/bin/env node\n',
    'package/scripts/layout.js': '\'use strict\';\n',
    'package/scripts/controller.js': '\'use strict\';\n',
    'package/scripts/commands.json': '{}\n',
  };
  delete npmEntries[missingNpmEntry];
  writeArchive(path.join(root, `fanloop-${version}.tgz`), npmEntries);
  files[`fanloop-${version}.tgz`] = fs.readFileSync(path.join(root, `fanloop-${version}.tgz`));
  const assets = Object.keys(files).sort().map((name) => {
    const asset = {name, sha256: sha(files[name])};
    const archive = name.match(/^fanloop-\d+\.\d+\.\d+-([a-z]+)-([a-z0-9]+)\.tar\.gz$/);
    if (archive) {
      asset.os = archive[1];
      asset.arch = archive[2];
    } else {
      asset.kind = 'npm';
    }
    return asset;
  });
  const manifest = `${JSON.stringify({schema_version: 1, version, tag: `v${version}`, repository: 'zeefan1555/fanloop', assets}, null, 2)}\n`;
  fs.writeFileSync(path.join(root, 'release.json'), manifest);
  const lines = [`${sha(Buffer.from(manifest))}  release.json`];
  for (const asset of assets) {
    lines.push(`${asset.sha256}  ${asset.name}`);
  }
  fs.writeFileSync(path.join(root, 'checksums.txt'), `${lines.join('\n')}\n`);
  if (mutate) {
    mutate(root);
  }
  return root;
}

test('accepts a complete consistent release directory', () => {
  const manifest = verifyDirectory(fixture());
  assert.equal(manifest.tag, 'v1.2.3');
  assert.equal(manifest.assets.length, 5);
});

for (const entry of ['package/scripts/controller.js', 'package/scripts/commands.json']) {
  test(`rejects an npm wrapper missing ${entry}`, () => {
    assert.throws(() => verifyDirectory(fixture(undefined, false, entry)), (err) => err.message.includes(`is missing ${entry}`));
  });
}

test('rejects a checksum that does not match the archive', () => {
  const root = fixture((directory) => {
    const checksums = path.join(directory, 'checksums.txt');
    fs.writeFileSync(checksums, fs.readFileSync(checksums, 'utf8').replace(/^[0-9a-f]{64}  fanloop-1\.2\.3-linux-amd64\.tar\.gz/m, `${'0'.repeat(64)}  fanloop-1.2.3-linux-amd64.tar.gz`));
  });
  assert.throws(() => verifyDirectory(root), /sha256 mismatch/);
});

test('rejects an incomplete internal release manifest', () => {
  assert.throws(() => verifyDirectory(fixture(undefined, true)), /schema_version must be 4|release\.json|binary checksum|skills/);
});

test('rejects a release that is missing a platform archive', () => {
  const root = fixture((directory) => {
    fs.rmSync(path.join(directory, 'fanloop-1.2.3-darwin-arm64.tar.gz'));
  });
  assert.throws(() => verifyDirectory(root), /checksums.txt does not cover|release directory contains|sha256 mismatch/);
});
