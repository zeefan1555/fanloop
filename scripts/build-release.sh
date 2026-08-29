#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
go_bundle=""

cleanup() {
  if [[ -n "$go_bundle" ]]; then
    rm -rf -- "$go_bundle"
  fi
}
trap cleanup EXIT

if ! command -v go >/dev/null || ! go version | grep -Eq 'go1\.(2[3-9]|[3-9][0-9])\.'; then
  if [[ "$(uname -s)" != "Linux" || "$(uname -m)" != "x86_64" ]]; then
    echo "Go 1.23+ is required to build Commonloop releases" >&2
    exit 1
  fi
  go_bundle="$(mktemp -d)"
  curl -fsSL https://dl.google.com/go/go1.23.12.linux-amd64.tar.gz -o "$go_bundle/go.tar.gz"
  echo "d3847fef834e9db11bf64e3fb34db9c04db14e068eeb064f49af747010454f90  $go_bundle/go.tar.gz" | sha256sum -c -
  tar -xzf "$go_bundle/go.tar.gz" -C "$go_bundle"
  export PATH="$go_bundle/go/bin:$PATH"
fi

export GOFLAGS="${GOFLAGS:+$GOFLAGS }-buildvcs=false"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"

"$repo_root/scripts/prepare-release.sh"
