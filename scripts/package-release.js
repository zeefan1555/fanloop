#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

function packageRelease(version, distArg = "dist") {
  if (!/^[0-9A-Za-z][0-9A-Za-z._+-]*$/.test(version)) {
    throw new Error("release version is required");
  }
  const repository = path.resolve(__dirname, "..");
  const dist = path.resolve(distArg);
  const manifestPath = path.join(dist, "release.json");
  const manifest = JSON.parse(fs.readFileSync(manifestPath));
  if (manifest.release_version !== version || manifest.cli?.version !== version) {
    throw new Error("release.json version does not match the npm package version");
  }

  const staging = path.join(dist, "npm");
  fs.rmSync(staging, { recursive: true, force: true });
  fs.mkdirSync(path.join(staging, "scripts"), { recursive: true });
  fs.mkdirSync(path.join(staging, "releases"));
  const packageJSON = JSON.parse(fs.readFileSync(path.join(repository, "package.json")));
  packageJSON.version = version;
  delete packageJSON.scripts["package:release"];
  packageJSON.files = [...new Set([...packageJSON.files, "releases"])];
  fs.writeFileSync(path.join(staging, "package.json"), `${JSON.stringify(packageJSON, null, 2)}\n`);
  for (const file of ["install.js", "run.js"]) {
    fs.copyFileSync(path.join(repository, "scripts", file), path.join(staging, "scripts", file));
  }
  fs.copyFileSync(path.join(repository, "README.md"), path.join(staging, "README.md"));
  fs.copyFileSync(manifestPath, path.join(staging, "release.json"));
  for (const asset of manifest.assets || []) {
    if (!asset.file || path.basename(asset.file) !== asset.file) throw new Error("release asset must be a file name");
    fs.copyFileSync(path.join(dist, asset.file), path.join(staging, "releases", asset.file));
  }

  const output = execFileSync("npm", ["pack", "--json", "--pack-destination", dist], {
    cwd: staging,
    encoding: "utf8",
  });
  const [{ filename }] = JSON.parse(output);
  return path.join(dist, filename);
}

if (require.main === module) {
  try {
    const artifact = packageRelease(process.argv[2], process.argv[3]);
    console.log(artifact);
  } catch (error) {
    console.error(`Fanloop package failed: ${error.message}`);
    process.exit(1);
  }
}

module.exports = { packageRelease };
