#!/usr/bin/env bash
# fanloop-github-release-launcher
set -euo pipefail

repository="zeefan1555/fanloop"

if [[ "$#" -eq 1 && "$1" == "update" ]]; then
  if ! command -v gh >/dev/null 2>&1; then
    echo "fanloop update requires GitHub CLI: https://cli.github.com" >&2
    exit 1
  fi
  temporary="$(mktemp -d)"
  trap 'rm -rf -- "$temporary"' EXIT
  gh release download -R "$repository" -p fanloop-install.sh -O "$temporary/fanloop-install.sh"
  FANLOOP_UPDATE_FORWARD_ONLY=1 bash "$temporary/fanloop-install.sh"
  exit
fi

data_root="${FANLOOP_DATA_HOME:-${HOME:?HOME is required}/.fanloop}"
binary="$data_root/current/bin/fanloop"
if [[ ! -x "$binary" ]]; then
  echo "Fanloop is not installed. Run: gh release download -R $repository -p fanloop-install.sh -O - | bash" >&2
  exit 1
fi
exec "$binary" "$@"
