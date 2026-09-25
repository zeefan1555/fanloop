#!/usr/bin/env node
// Version-line allocation follows the verified npm release process.
const fs = require('node:fs');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { stableVersion } = require('./install.js');
function selectVersionFrom(metadata, baseline, run = spawnSync) {
  const base = stableVersion(baseline);
  if (!base || metadata.name !== '@zeefan1555/fanloop-cli' || metadata.publishConfig?.registry !== 'https://registry.npmjs.org/') throw new Error('Invalid release identity or VERSION');
  const result = run('npm', ['view', metadata.name, 'versions', '--json', '--prefer-online', '--registry=https://registry.npmjs.org/'], { encoding: 'utf8' });
  let versions = [];
  if (result.status !== 0) {
    if (!/\bE404\b/.test((result.stderr || '') + (result.stdout || ''))) throw new Error('Cannot read published versions: ' + (result.error?.message || result.stderr || result.status));
  } else {
    const value = JSON.parse(result.stdout);
    versions = Array.isArray(value) ? value : [value];
  }
  let patch = base[2];
  for (const value of versions) {
    const candidate = stableVersion(String(value));
    if (candidate && candidate[0] === base[0] && candidate[1] === base[1]) patch = Math.max(patch, candidate[2] + 1);
  }
  return base[0] + '.' + base[1] + '.' + patch;
}
function selectVersion(root = path.resolve(__dirname, '..'), run = spawnSync) {
  const metadata = JSON.parse(fs.readFileSync(path.join(root, 'package.json'), 'utf8'));
  const baseline = fs.readFileSync(path.join(root, 'VERSION'), 'utf8').trim();
  return selectVersionFrom(metadata, baseline, run);
}
if (require.main === module) {
  try { console.log(selectVersion()); } catch (error) { console.error(error.message); process.exit(1); }
}
module.exports = { selectVersion, selectVersionFrom };
