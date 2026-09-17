#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 1 ]]; then
  echo "usage: $0 [OUTPUT_DIR]" >&2
  exit 1
fi
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$repo_root"
export GOTOOLCHAIN=local
command -v go >/dev/null || { echo "Go 1.23+ is required" >&2; exit 1; }
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
if [[ $# -eq 1 ]]; then
  build_parent="$(cd "$(dirname -- "$1")" && pwd -P)"
  build_root="$build_parent/$(basename -- "$1")"
else
  mkdir -p "$repo_root/dist"
  build_parent="$(cd "$repo_root/dist" && pwd -P)"
  build_root="$build_parent/local-$stamp-XXXXXX"
fi
git_directory="$(git rev-parse --absolute-git-dir)"
git_common_directory="$(cd "$(git rev-parse --git-common-dir)" && pwd -P)"
case "$build_root/" in
  "$repo_root/skills/"*|"$repo_root/entrypoints/"*|"$repo_root/workflows/"*|"$repo_root/.git/"*|"$git_directory/"*|"$git_common_directory/"*)
    echo "Build output must be outside copied component trees and Git metadata" >&2
    exit 1
    ;;
esac
if [[ $# -eq 1 ]]; then
  # mkdir refuses existing paths; never remove an arbitrary user directory.
  mkdir -- "$build_root"
else
  build_root="$(mktemp -d "$build_root")"
fi
source_paths=(. ':(exclude)skills' ':(exclude)skills/**' ':(exclude)exemplars' ':(exclude)exemplars/**')
case "$build_root/" in
  "$repo_root/"*) source_paths+=(":(exclude,literal)${build_root#"$repo_root/"}") ;;
esac
source_fingerprint() {
  {
    git diff --binary --no-ext-diff --no-textconv HEAD -- "${source_paths[@]}" || exit
    git ls-files --others --exclude-standard -z -- "${source_paths[@]}" |
      while IFS= read -r -d '' file; do
        if [[ -f "$file" && ! -L "$file" ]]; then
          printf '\0%s\0' "$file"
          git hash-object --no-filters -- "$file" || exit
        fi
      done
  } | git hash-object --stdin
}
commit="$(git rev-parse HEAD)"
source_status="$(git status --porcelain -- "${source_paths[@]}")"
source_digest="$(source_fingerprint)"
base_version="$(<"$repo_root/VERSION")"
if [[ ! "$base_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "VERSION must contain a semantic version such as 1.2.3" >&2
  exit 1
fi
version="$base_version-dev.${commit:0:12}"
if [[ -n "$source_status" ]]; then
  version="$base_version-dev.${source_digest:0:12}"
fi
printf 'Building %s from %s in %s\n' "$version" "$commit" "$build_root" >&2

mkdir "$build_root/bin" "$build_root/workflows"
export GOOS="$(go env GOHOSTOS)" GOARCH="$(go env GOHOSTARCH)" CGO_ENABLED=0
go build -trimpath -buildvcs=false \
  -ldflags "-s -w -X github.com/zeefan1555/fanloop/internal/buildinfo.ReleaseVersion=$version -X github.com/zeefan1555/fanloop/internal/buildinfo.CLIVersion=$version -X github.com/zeefan1555/fanloop/internal/buildinfo.Commit=$commit" \
  -o "$build_root/bin/fanloop" . >&2
cp -R "$repo_root/entrypoints" "$build_root/"
for workflow_source in "$repo_root"/workflows/*/; do
  workflow_name="$(basename "$workflow_source")"
  mkdir "$build_root/workflows/$workflow_name"
  cp "$workflow_source"*.yaml "$build_root/workflows/$workflow_name/"
done
go run ./tools/release-manifest --version "$version" --source "$repo_root" \
  --dist "$build_root" --output "$build_root/release.json" >&2
if [[ "$(git rev-parse HEAD)" != "$commit" || "$(git status --porcelain -- "${source_paths[@]}")" != "$source_status" || "$(source_fingerprint)" != "$source_digest" ]]; then
  echo "Source changed during build; rerun from a stable working tree" >&2
  exit 1
fi
printf '%s\n' "$build_root"
