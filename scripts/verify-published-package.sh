#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repository_root"

version="${1:-$(node -p 'require("./package.json").version')}"
case "$version" in
  *[!0-9A-Za-z._+-]*|'')
    echo "invalid release version: $version" >&2
    exit 1
    ;;
esac
selector="${2:-$version}"
if [[ "$selector" != "$version" && "$selector" != "latest" ]]; then
  echo "invalid release selector: $selector" >&2
  exit 1
fi

smoke_root="$(mktemp -d)"
test -n "$smoke_root" && test -d "$smoke_root"

unset NODE_AUTH_TOKEN NPM_TOKEN
export NPM_CONFIG_REGISTRY=https://registry.npmjs.org
export NPM_CONFIG_USERCONFIG=/dev/null
export NPM_CONFIG_PREFIX="$smoke_root/npm"
export NPM_CONFIG_CACHE="$smoke_root/cache"
export FANLOOP_DATA_HOME="$smoke_root/data"
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

if [[ "$selector" == "latest" ]]; then
  selector_ready=false
  attempt=0
  while [ "$attempt" -lt 12 ]; do
    resolved="$(npm view "fanloop-cli@latest" version --prefer-online 2>/dev/null || true)"
    if [[ "$resolved" == "$version" ]]; then
      selector_ready=true
      break
    fi
    attempt=$((attempt + 1))
    sleep 5
  done
  test "$selector_ready" = true
fi

mkdir -p "$smoke_root/registry-check"
package_ready=false
attempt=0
while [ "$attempt" -lt 12 ]; do
  if npm pack "fanloop-cli@$selector" --pack-destination "$smoke_root/registry-check" --json >/dev/null 2>&1; then
    package_ready=true
    break
  fi
  attempt=$((attempt + 1))
  sleep 5
done
test "$package_ready" = true

node --version
npm --version
npx --version
install_output="$(
  cd "$smoke_root"
  npx --yes --package="fanloop-cli@$selector" -- fanloop install
)"
printf '%s\n' "$install_output"
test "$install_output" = "Fanloop $version installed successfully"
fanloop_bin="$NPM_CONFIG_PREFIX/bin/fanloop"
test -x "$fanloop_bin"

version_output="$("$fanloop_bin" version)"
printf '%s\n' "$version_output"
printf '%s\n' "$version_output" | grep "\"release_version\": \"$version\""
if [ -n "${FANLOOP_EXPECTED_COMMIT:-}" ]; then
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

"$fanloop_bin" version >/dev/null
printf 'Fanloop %s anonymous release smoke passed with %s packaged Skills\n' "$version" "$packaged_skill_count"
