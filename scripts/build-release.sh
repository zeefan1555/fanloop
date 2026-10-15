#!/usr/bin/env bash
# Build the GitHub Release assets for the VERSION in this repository.
# Cross-compiles darwin/linux amd64/arm64 on the current machine.
set -euo pipefail
export COPYFILE_DISABLE=1

if [[ $# -gt 1 ]]; then
  echo "usage: $0 [OUTPUT_DIR]" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$repo_root"
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
command -v go >/dev/null || { echo "Go 1.23+ is required" >&2; exit 1; }
command -v node >/dev/null || { echo "Node.js 18+ is required" >&2; exit 1; }
command -v npm >/dev/null || { echo "npm is required" >&2; exit 1; }

version="$(<"$repo_root/VERSION")"
version="${version//$'\n'/}"
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "VERSION must contain a semantic version such as 1.2.3" >&2
  exit 1
fi
# GITHUB_REF_NAME is the branch name on push and pull_request jobs. Only a tag
# workflow publishes a release, so compare VERSION only for refs/tags/*.
if [[ "${GITHUB_REF:-}" == refs/tags/* && "${GITHUB_REF_NAME:-}" != "v$version" ]]; then
  echo "tag ${GITHUB_REF_NAME:-} does not match VERSION $version" >&2
  exit 1
fi

if [[ $# -eq 1 ]]; then
  build_parent="$(cd "$(dirname -- "$1")" && pwd -P)"
  output_root="$build_parent/$(basename -- "$1")"
  if [[ -L "$output_root" || ( -e "$output_root" && ! -d "$output_root" ) ]]; then
    echo "output path is not a directory: $output_root" >&2
    exit 1
  fi
  if [[ -d "$output_root" ]]; then
    if [[ -n "$(find "$output_root" -mindepth 1 -maxdepth 1 -print)" ]]; then
      echo "output directory is not empty: $output_root" >&2
      exit 1
    fi
  else
    mkdir -- "$output_root"
  fi
else
  mkdir -p "$repo_root/dist"
  output_root="$(mktemp -d "$repo_root/dist/release-$version-XXXXXX")"
fi

commit="$(git rev-parse HEAD)"
printf 'Building GitHub release %s from %s in %s\n' "$version" "$commit" "$output_root" >&2

platforms=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
)
probe="$(mktemp -d)"
mkdir "$probe/d"
: >"$probe/d/f"
tar_flags=()
if tar --format=ustar -cf /dev/null -C "$probe" d >/dev/null 2>&1; then
  tar_flags+=(--format=ustar)
elif tar --format ustar -cf /dev/null -C "$probe" d >/dev/null 2>&1; then
  tar_flags+=(--format ustar)
else
  echo "tar cannot write a ustar archive" >&2
  exit 1
fi
tar_try() {
  tar "${tar_flags[@]}" "$@" -cf /dev/null -C "$probe" d >/dev/null 2>&1
}
if tar_try --no-xattrs; then
  tar_flags+=(--no-xattrs)
fi
if tar_try --no-mac-metadata; then
  tar_flags+=(--no-mac-metadata)
fi
if tar_try --disable-copyfile; then
  tar_flags+=(--disable-copyfile)
fi
if tar_try --mtime=@0; then
  tar_flags+=(--mtime=@0)
fi
if tar_try --owner=0 --group=0 --numeric-owner; then
  tar_flags+=(--owner=0 --group=0 --numeric-owner)
fi
rm -rf "$probe"

write_platform_archive() {
  local stage="$1"
  local archive="$2"
  local list
  list="$(mktemp)"
  (
    cd "$stage"
    find bin entrypoints release.json ! -name '._*' ! -name '.DS_Store' -print | LC_ALL=C sort
  ) >"$list"
  if tar "${tar_flags[@]}" --no-recursion --exclude='._*' --exclude='.DS_Store' -C "$stage" -cf /dev/null -T "$list" >/dev/null 2>&1; then
    tar "${tar_flags[@]}" --no-recursion --exclude='._*' --exclude='.DS_Store' -C "$stage" -czf "$archive" -T "$list"
  else
    tar "${tar_flags[@]}" --exclude='._*' --exclude='.DS_Store' -C "$stage" -czf "$archive" bin entrypoints release.json
  fi
  rm -f "$list"
}

archives=()
for pair in "${platforms[@]}"; do
  read -r goos goarch <<<"$pair"
  stage="$(mktemp -d)"
  mkdir "$stage/bin"
  printf 'compiling %s/%s\n' "$goos" "$goarch" >&2
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags "-s -w -X github.com/zeefan1555/fanloop/internal/buildinfo.ReleaseVersion=$version -X github.com/zeefan1555/fanloop/internal/buildinfo.CLIVersion=$version -X github.com/zeefan1555/fanloop/internal/buildinfo.Commit=$commit" \
    -o "$stage/bin/fanloop" . >&2
  cp -R "$repo_root/entrypoints" "$stage/"
  cp -R "$repo_root/.agents/skills/create-fanloop-workflow" "$stage/entrypoints/"
  go run ./tools/release-manifest --version "$version" --source "$repo_root" \
    --dist "$stage" --output "$stage/release.json" >&2
  archive="fanloop-$version-$goos-$goarch.tar.gz"
  write_platform_archive "$stage" "$output_root/$archive"
  rm -rf "$stage"
  archives+=("$archive")
done

package_stage="$(mktemp -d)"
cp -R "$repo_root/packaging/npm/." "$package_stage/"
rm -f "$package_stage/install.test.js"
node -e 'const fs=require("fs"); const pkg=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); pkg.version=process.argv[2]; fs.writeFileSync(process.argv[1], JSON.stringify(pkg, null, 2)+"\n");' \
  "$package_stage/package.json" "$version"
npm_json="$(cd "$package_stage" && npm pack --pack-destination "$output_root" --json)"
npm_name="$(node -e 'const fs=require("fs"); let raw=fs.readFileSync(0,"utf8"); const start=raw.indexOf("["); const parsed=JSON.parse(raw.slice(start)); if(!parsed[0]||!parsed[0].filename) throw new Error("npm pack did not report a filename"); process.stdout.write(parsed[0].filename);' <<<"$npm_json")"
if [[ "$npm_name" != "fanloop-$version.tgz" ]]; then
  echo "npm pack produced $npm_name, want fanloop-$version.tgz" >&2
  exit 1
fi
rm -rf "$package_stage"

node - "$output_root" "$version" "${archives[@]}" "$npm_name" <<'NODE'
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const [out, version, ...names] = process.argv.slice(2);
const sha = (name) => crypto.createHash('sha256').update(fs.readFileSync(path.join(out, name))).digest('hex');
const assets = names.map((name) => {
  const asset = {name, sha256: sha(name)};
  const archive = name.match(/^fanloop-\d+\.\d+\.\d+-([a-z]+)-([a-z0-9]+)\.tar\.gz$/);
  if (archive) {
    asset.os = archive[1];
    asset.arch = archive[2];
  } else if (name.endsWith('.tgz')) {
    asset.kind = 'npm';
  }
  return asset;
});
const manifest = {
  schema_version: 1,
  version,
  tag: `v${version}`,
  repository: 'zeefan1555/fanloop',
  assets,
};
fs.writeFileSync(path.join(out, 'release.json'), `${JSON.stringify(manifest, null, 2)}\n`);
NODE

files=("${archives[@]}" "$npm_name" release.json)
checksums="$output_root/checksums.txt"
: >"$checksums"
for name in "${files[@]}"; do
  if command -v sha256sum >/dev/null; then
    hash="$(sha256sum "$output_root/$name" | awk '{print $1}')"
  else
    hash="$(shasum -a 256 "$output_root/$name" | awk '{print $1}')"
  fi
  printf '%s  %s\n' "$hash" "$name" >>"$checksums"
done

printf '%s\n' "$output_root"
