#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repository_root"

version="${1:-$(node -p 'require("./package.json").version')}"
selector="${2:-exact}"
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid release version: $version" >&2
  exit 1
fi
if [[ "$selector" != "exact" && "$selector" != "latest" && "$selector" != "local" ]]; then
  echo "invalid release selector: $selector" >&2
  exit 1
fi

smoke_root="$(mktemp -d)"
trap 'rm -rf -- "$smoke_root"' EXIT
export FANLOOP_DATA_HOME="$smoke_root/data"
export FANLOOP_BIN_DIR="$smoke_root/bin"
export FANLOOP_CODEX_SKILLS_ROOT="$smoke_root/codex-skills"
export FANLOOP_AGENT_SKILLS_ROOT="$smoke_root/agent-skills"
export FANLOOP_TRAE_SKILLS_ROOT="$smoke_root/trae-skills"
export FANLOOP_CLAUDE_SKILLS_ROOT="$smoke_root/claude-skills"

skill_roots=(
  "$FANLOOP_CODEX_SKILLS_ROOT"
  "$FANLOOP_AGENT_SKILLS_ROOT"
  "$FANLOOP_TRAE_SKILLS_ROOT"
  "$FANLOOP_CLAUDE_SKILLS_ROOT"
)
external_markers=()
for index in "${!skill_roots[@]}"; do
  skill_root="${skill_roots[$index]}"
  external_target="$smoke_root/external-skill-$index"
  marker="$external_target/preserved"
  mkdir -p "$skill_root" "$external_target"
  printf 'preserve external Skill target\n' >"$marker"
  ln -s "$external_target" "$skill_root/fanloop-workflow"
  external_markers+=("$marker")
done

case "$selector" in
  local)
    : "${FANLOOP_RELEASE_DIR:?FANLOOP_RELEASE_DIR is required for local verification}"
    install_output="$(bash "$repository_root/scripts/install-github.sh")"
    ;;
  exact)
    install_output="$(FANLOOP_RELEASE_TAG="v$version" bash "$repository_root/scripts/install-github.sh")"
    ;;
  latest)
    install_output="$(bash "$repository_root/scripts/install-github.sh")"
    ;;
esac
printf '%s\n' "$install_output"
test "$install_output" = "Fanloop $version installed successfully"
fanloop_bin="$FANLOOP_BIN_DIR/fanloop"
test -x "$fanloop_bin"

version_output="$("$fanloop_bin" version)"
printf '%s\n' "$version_output"
printf '%s\n' "$version_output" | grep "\"release_version\": \"$version\""
if [[ -n "${FANLOOP_EXPECTED_COMMIT:-}" ]]; then
  printf '%s\n' "$version_output" | grep "\"commit_sha\": \"$FANLOOP_EXPECTED_COMMIT\""
fi
"$fanloop_bin" doctor | grep '"status": "healthy"'

if [[ "$selector" == "latest" ]]; then
  update_output="$("$fanloop_bin" update)"
  printf '%s\n' "$update_output"
  test "$update_output" = "Fanloop $version installed successfully"
  "$fanloop_bin" doctor | grep '"status": "healthy"'
fi

manifest="$FANLOOP_DATA_HOME/current/release.json"
test -f "$manifest"
workflow_ids="$(node -e 'const manifest = require(process.argv[1]); for (const workflow of manifest.workflows) console.log(workflow.id)' "$manifest")"
workflow_count=0
while IFS= read -r workflow_id; do
  test -n "$workflow_id"
  workflow_count=$((workflow_count + 1))
  requirement="$smoke_root/requirements/$workflow_id"
  mkdir -p "$requirement"
  init_output="$("$fanloop_bin" flow init --root "$requirement" --workflow "$workflow_id" --title "Release smoke: $workflow_id")"
  printf '%s\n' "$init_output"
  printf '%s\n' "$init_output" | node -e '
let content = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", chunk => content += chunk);
process.stdin.on("end", () => {
  const response = JSON.parse(content);
  if (!response.ok || response.data?.workflow?.id !== process.argv[1] || !response.data?.state?.current?.context?.step_id) {
    throw new Error(`Workflow ${process.argv[1]} did not initialize to a current Step`);
  }
});
' "$workflow_id"
done <<< "$workflow_ids"
test "$workflow_count" -gt 0

packaged_skill_count="$(node -e 'const manifest = require(process.argv[1]); process.stdout.write(String(manifest.skills.length))' "$manifest")"
test "$packaged_skill_count" -gt 0
for skill_root in "${skill_roots[@]}"; do
  test "$(find "$skill_root" -maxdepth 1 -type l | wc -l | tr -d ' ')" = 1
  test -L "$skill_root/fanloop-workflow"
done
for marker in "${external_markers[@]}"; do
  test -f "$marker"
  grep 'preserve external Skill target' "$marker" >/dev/null
done

printf 'Fanloop %s GitHub Release smoke passed with %s packaged Skills\n' "$version" "$packaged_skill_count"
