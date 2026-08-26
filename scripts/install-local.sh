#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

FANLOOP_LOCAL_BUILD=1 "$repo_root/scripts/build-release.sh"
node "$repo_root/scripts/install.js"
