#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { checksum, stableVersion } = require('./install.js');
const platforms = [['darwin', 'amd64'], ['darwin', 'arm64'], ['linux', 'amd64'], ['linux', 'arm64']];
function packageRelease(version) {
  if (!stableVersion(version)) throw new Error('Release version must be stable major.minor.patch');
  const root = path.resolve(__dirname, '..');
  const git = (...args) => execFileSync('git', args, { cwd: root, encoding: 'utf8' }).trim();
  if (git('status', '--porcelain')) throw new Error('Commit the source before building a release artifact');
  const commit = git('rev-parse', 'HEAD');
  const env = { ...process.env, GOTOOLCHAIN: 'local', CGO_ENABLED: '0' };
  const [hostOS, hostArch] = execFileSync('go', ['env', 'GOHOSTOS', 'GOHOSTARCH'], { env, encoding: 'utf8' }).trim().split(/\s+/);
  fs.mkdirSync(path.join(root, 'dist'), { recursive: true });
  const output = fs.mkdtempSync(path.join(root, 'dist', 'npm-release-'));
  const content = path.join(output, 'content'), staging = path.join(output, 'npm');
  fs.mkdirSync(content); fs.mkdirSync(staging);
  const sourceArchive = path.join(output, 'source.tar');
  execFileSync('git', ['archive', '--format=tar', '--output=' + sourceArchive, commit, '--', 'entrypoints'], { cwd: root });
  execFileSync('tar', ['-xf', sourceArchive, '-C', content]);
  fs.mkdirSync(path.join(staging, 'scripts'));
  fs.mkdirSync(path.join(staging, 'releases'));
  const checksums = [];
  for (const [os, arch] of platforms) {
    const release = path.join(output, os + '-' + arch);
    fs.mkdirSync(path.join(release, 'bin'), { recursive: true });
    fs.cpSync(path.join(content, 'entrypoints'), path.join(release, 'entrypoints'), { recursive: true });
    console.error('Building Fanloop ' + version + ' for ' + os + '/' + arch + ' at ' + commit);
    const fields = 'github.com/zeefan1555/fanloop/internal/buildinfo.';
    execFileSync('go', ['build', '-trimpath', '-buildvcs=false', '-ldflags', '-s -w -X ' + fields + 'ReleaseVersion=' + version + ' -X ' + fields + 'CLIVersion=' + version + ' -X ' + fields + 'Commit=' + commit, '-o', path.join(release, 'bin/fanloop'), '.'], { cwd: root, env: { ...env, GOOS: os, GOARCH: arch }, stdio: ['ignore', 'inherit', 'inherit'] });
    execFileSync('go', ['run', './tools/release-manifest', '--version', version, '--source', content, '--dist', release, '--output', path.join(release, 'release.json')], { cwd: root, env: { ...env, GOOS: hostOS, GOARCH: hostArch }, stdio: ['ignore', 'inherit', 'inherit'] });
    const name = 'fanloop-' + version + '-' + os + '-' + arch + '.tar.gz';
    const archive = path.join(staging, 'releases', name);
    execFileSync('tar', ['-czf', archive, '-C', release, 'bin', 'entrypoints', 'release.json']);
    checksums.push(checksum(archive) + '  ' + name);
  }
  fs.writeFileSync(path.join(staging, 'checksums.txt'), checksums.join('\n') + '\n');
  for (const name of ['install.js', 'run.js']) {
    fs.copyFileSync(path.join(root, 'scripts', name), path.join(staging, 'scripts', name));
    fs.chmodSync(path.join(staging, 'scripts', name), 0o755);
  }
  fs.copyFileSync(path.join(root, 'README.md'), path.join(staging, 'README.md'));
  const metadata = JSON.parse(fs.readFileSync(path.join(root, 'package.json'), 'utf8'));
  metadata.version = version;
  delete metadata.scripts;
  fs.writeFileSync(path.join(staging, 'package.json'), JSON.stringify(metadata, null, 2) + '\n');
  const packed = JSON.parse(execFileSync('npm', ['pack', '--json', '--pack-destination', output], { cwd: staging, encoding: 'utf8' }));
  const tarball = path.join(output, packed[0].filename);
  if (git('rev-parse', 'HEAD') !== commit || git('status', '--porcelain')) throw new Error('Source changed during release build');
  fs.writeFileSync(path.join(root, 'dist/npm-release.json'), JSON.stringify({ version, commit, tarball }, null, 2) + '\n');
  return tarball;
}
if (require.main === module) {
  try { console.log(packageRelease(process.argv[2])); } catch (error) { console.error('Fanloop package failed: ' + error.message); process.exit(1); }
}
module.exports = { packageRelease, platforms };
