#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
export GOTOOLCHAIN=local
command -v go >/dev/null || { echo 'Go 1.23+ is required on the release builder' >&2; exit 1; }
command -v node >/dev/null || { echo 'Node.js 18+ is required' >&2; exit 1; }
[[ $# -le 1 ]] || { echo 'usage: build-npm-release.sh [VERSION]' >&2; exit 1; }
version="${1:-$(node scripts/resolve-release-version.js)}"
node scripts/package-release.js "$version"
