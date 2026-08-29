#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
package_json="$repo_root/package.json"
if [[ "${FANLOOP_LOCAL_BUILD:-}" == "1" ]]; then
  version="$(node -p 'require(process.argv[1]).version' "$package_json")"
else
  version="$(node "$repo_root/scripts/resolve-release-version.js" "$package_json")"
fi
echo "Resolved release version: $version"
release_tag="v$version"
head_sha="$(git -C "$repo_root" rev-parse HEAD)"
tag_sha="$(git -C "$repo_root" rev-parse -q --verify "refs/tags/$release_tag^{commit}" 2>/dev/null || true)"
temporary="$(mktemp -d)"
created_local_tag=false
cleanup() {
  if [[ "$created_local_tag" == true ]]; then
    git -C "$repo_root" tag -d "$release_tag" >/dev/null 2>&1 || true
  fi
  rmdir "$temporary"
}
trap cleanup EXIT
if [[ -n "$tag_sha" && "$tag_sha" != "$head_sha" ]]; then
  echo "release tag $release_tag points to $tag_sha, not current commit $head_sha" >&2
  exit 1
fi
if [[ -z "$tag_sha" ]]; then
  git -C "$repo_root" tag "$release_tag" "$head_sha"
  created_local_tag=true
fi

cd "$repo_root"
goreleaser_args=(release --clean --skip=publish)
if [[ "${FANLOOP_LOCAL_BUILD:-}" == "1" ]]; then
  goreleaser_args=(release --clean --snapshot)
fi
FANLOOP_BUILD_COMMIT="$head_sha" FANLOOP_RELEASE_VERSION="$version" GOFLAGS=-buildvcs=false \
  go run github.com/goreleaser/goreleaser/v2@v2.5.1 "${goreleaser_args[@]}"
./scripts/package-release.sh "$version" "$repo_root/dist"

release_dir="$repo_root/releases"
if [[ "$release_dir" != "$repo_root/releases" || "$repo_root" == "/" ]]; then
  echo "unsafe release directory" >&2
  exit 1
fi
rm -rf -- "$release_dir"
mkdir -p "$release_dir"
cp "$repo_root/dist/release.json" "$repo_root/release.json"
cp "$repo_root"/dist/fanloop-"$version"-*.tar.xz "$release_dir/"
if [[ "${FANLOOP_LOCAL_BUILD:-}" != "1" ]]; then
  node "$repo_root/scripts/resolve-release-version.js" --write "$package_json" "$version"
fi

node -e '
const fs = require("fs");
const path = require("path");
const root = process.argv[1];
const manifest = JSON.parse(fs.readFileSync(path.join(root, "release.json")));
if (manifest.assets.length !== 4) throw new Error("release must contain four platform assets");
for (const asset of manifest.assets) {
  if (!fs.existsSync(path.join(root, "releases", asset.file))) throw new Error(`missing ${asset.file}`);
}
' "$repo_root"
npm pack --dry-run --json >/dev/null
