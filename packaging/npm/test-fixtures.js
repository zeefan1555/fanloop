'use strict';

const {execFileSync} = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const os = require('os');
const path = require('path');
const layout = require('./scripts/layout');

function writeValidTree(stage, version, bodies = {}) {
  fs.mkdirSync(path.join(stage, 'bin'), {recursive: true});
  fs.mkdirSync(path.join(stage, 'entrypoints', 'fanloop-workflow'), {recursive: true});
  fs.mkdirSync(path.join(stage, 'entrypoints', 'create-fanloop-workflow'), {recursive: true});
  fs.writeFileSync(path.join(stage, 'bin', 'fanloop'), bodies.binary || `#!/bin/sh\necho ${version}\n`, {mode: 0o755});
  fs.writeFileSync(path.join(stage, 'entrypoints', 'fanloop-workflow', 'SKILL.md'), bodies.workflow || '# workflow\n');
  fs.writeFileSync(path.join(stage, 'entrypoints', 'create-fanloop-workflow', 'SKILL.md'), bodies.author || '# author\n');
  const manifest = {
    schema_version: 4,
    release_version: version,
    cli: {version, binary_sha256: layout.fileDigest(path.join(stage, 'bin', 'fanloop'))},
    state_schema: {read_versions: [14], write_version: 14},
    skills: [
      {name: 'fanloop-workflow', version, path: 'entrypoints/fanloop-workflow', sha256: layout.directoryDigest(path.join(stage, 'entrypoints', 'fanloop-workflow'))},
      {name: 'create-fanloop-workflow', version, path: 'entrypoints/create-fanloop-workflow', sha256: layout.directoryDigest(path.join(stage, 'entrypoints', 'create-fanloop-workflow'))},
    ],
  };
  fs.writeFileSync(path.join(stage, 'release.json'), `${JSON.stringify(manifest)}\n`);
}

function skillRoots(parent) {
  return ['codex', 'agents', 'trae', 'claude'].map((name) => {
    const dir = path.join(parent, 'skills', name);
    fs.mkdirSync(dir, {recursive: true});
    return dir;
  });
}

function packStage(stage, archivePath, format) {
  const members = ['-C', stage, 'bin', 'entrypoints', 'release.json'];
  if (!format) {
    execFileSync('tar', ['-czf', archivePath, ...members]);
    return;
  }
  try {
    execFileSync('tar', [`--format=${format}`, '-czf', archivePath, ...members], {stdio: 'pipe'});
  } catch {
    execFileSync('tar', ['--format', format, '-czf', archivePath, ...members]);
  }
}

function releaseAssets(version, prepare, options = {}) {
  const release = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-assets-'));
  const stage = fs.mkdtempSync(path.join(os.tmpdir(), 'fanloop-stage-'));
  writeValidTree(stage, version);
  if (prepare) prepare(stage);
  const platform = options.platform || {os: 'linux', arch: 'amd64'};
  const archiveName = `fanloop-${version}-${platform.os}-${platform.arch}.tar.gz`;
  packStage(stage, path.join(release, archiveName), options.format);
  const archiveHash = crypto.createHash('sha256').update(fs.readFileSync(path.join(release, archiveName))).digest('hex');
  const manifest = {schema_version: 1, version, tag: `v${version}`, repository: 'zeefan1555/fanloop', assets: [{name: archiveName, os: platform.os, arch: platform.arch, sha256: archiveHash}]};
  fs.writeFileSync(path.join(release, 'release.json'), `${JSON.stringify(manifest)}\n`);
  const manifestHash = crypto.createHash('sha256').update(fs.readFileSync(path.join(release, 'release.json'))).digest('hex');
  fs.writeFileSync(path.join(release, 'checksums.txt'), `${archiveHash}  ${archiveName}\n${manifestHash}  release.json\n`);
  return {release, archiveName};
}

module.exports = {writeValidTree, skillRoots, packStage, releaseAssets};
