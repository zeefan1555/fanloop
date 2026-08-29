#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:?usage: assemble-release.sh <version> [dist]}"
dist="${2:-$repo_root/dist}"

cd "$repo_root"
mkdir -p "$dist"
go run "$repo_root/tools/release-manifest" \
  --version "$version" \
  --source "$repo_root" \
  --dist "$dist" \
  --output "$dist/release.json"
cp "$repo_root/scripts/install.js" "$dist/fanloop-install.js"
cp "$repo_root/scripts/install-github.sh" "$dist/fanloop-install.sh"
cp "$repo_root/scripts/fanloop-launcher.sh" "$dist/fanloop-launcher.sh"
chmod 0755 "$dist/fanloop-install.sh" "$dist/fanloop-launcher.sh"
