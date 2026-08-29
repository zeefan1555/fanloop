#!/usr/bin/env bash
set -euo pipefail

repository="zeefan1555/fanloop"
download_root=""
launcher_staging=""

cleanup() {
  if [[ -n "$launcher_staging" ]]; then
    rm -f -- "$launcher_staging"
  fi
  if [[ -n "$download_root" ]]; then
    rm -rf -- "$download_root"
  fi
}
trap cleanup EXIT

for command in node tar; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "$command is required to install Fanloop" >&2
    exit 1
  fi
done

case "$(uname -s)" in
  Darwin) platform="darwin" ;;
  Linux) platform="linux" ;;
  *) echo "unsupported platform: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) architecture="amd64" ;;
  arm64|aarch64) architecture="arm64" ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [[ -n "${FANLOOP_RELEASE_DIR:-}" ]]; then
  source_root="$(cd "$FANLOOP_RELEASE_DIR" && pwd)"
else
  if ! command -v gh >/dev/null 2>&1; then
    echo "GitHub CLI is required: https://cli.github.com" >&2
    exit 1
  fi
  if ! gh auth status --hostname github.com >/dev/null 2>&1; then
    echo "Authenticate once with: gh auth login" >&2
    exit 1
  fi
  if [[ -n "${FANLOOP_RELEASE_TAG:-}" ]]; then
    if [[ ! "$FANLOOP_RELEASE_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "invalid Fanloop release tag: $FANLOOP_RELEASE_TAG" >&2
      exit 1
    fi
  fi
  download_root="$(mktemp -d)"
  download_release() {
    gh release download "$@" -R "$repository" \
      -p release.json \
      -p fanloop-install.js \
      -p fanloop-launcher.sh \
      -p "fanloop-*-$platform-$architecture.tar.xz" \
      -D "$download_root"
  }
  if [[ -n "${FANLOOP_RELEASE_TAG:-}" ]]; then
    download_release "$FANLOOP_RELEASE_TAG"
  else
    download_release
  fi
  source_root="$download_root"
fi

manifest="$source_root/release.json"
installer="$source_root/fanloop-install.js"
launcher="$source_root/fanloop-launcher.sh"
for file in "$manifest" "$installer" "$launcher"; do
  if [[ ! -f "$file" ]]; then
    echo "missing GitHub Release asset: $(basename "$file")" >&2
    exit 1
  fi
done

shopt -s nullglob
archives=("$source_root"/fanloop-*-$platform-$architecture.tar.xz)
shopt -u nullglob
if [[ "${#archives[@]}" -ne 1 ]]; then
  echo "expected one Fanloop archive for $platform-$architecture, found ${#archives[@]}" >&2
  exit 1
fi

bin_dir="${FANLOOP_BIN_DIR:-${HOME:?HOME is required}/.local/bin}"
mkdir -p "$bin_dir"
launcher_target="$bin_dir/fanloop"
if [[ -e "$launcher_target" || -L "$launcher_target" ]]; then
  if [[ ! -f "$launcher_target" ]] || ! grep -q '^# fanloop-github-release-launcher$' "$launcher_target"; then
    echo "refusing to replace unmanaged launcher: $launcher_target" >&2
    exit 1
  fi
fi
launcher_staging="$(mktemp "$bin_dir/.fanloop-launcher.XXXXXX")"
cp "$launcher" "$launcher_staging"
chmod 0755 "$launcher_staging"

FANLOOP_RELEASE_MANIFEST="$manifest" FANLOOP_RELEASE_ARCHIVE="${archives[0]}" node "$installer"
mv -f "$launcher_staging" "$launcher_target"
launcher_staging=""
