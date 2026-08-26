#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:?usage: package-release.sh <version> [dist]}"
dist="${2:-$repo_root/dist}"

cd "$repo_root"
mkdir -p "$dist"
manifest_args=(
  --version "$version" \
  --source "$repo_root" \
  --dist "$dist" \
  --output "$dist/release.json"
)
go run "$repo_root/tools/release-manifest" "${manifest_args[@]}"
node "$repo_root/scripts/package-release.js" "$version" "$dist"
