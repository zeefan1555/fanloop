#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || "$1" != /* || ! -d "$1" || ! -f "$1/.fanloop/flow/state.json" ]]; then
  echo "usage: pin-controller-release.sh <ABSOLUTE_INITIALIZED_REQUIREMENT_ROOT>" >&2
  exit 2
fi
requirement_root="$(cd "$1" && pwd -P)"
controller_home="$requirement_root/bound-release-home"
skill_roots="$controller_home/skill-roots"
controller_binary="$controller_home/current/bin/fanloop"
controller_env=("FANLOOP_DATA_HOME=$controller_home"
  "FANLOOP_CODEX_SKILLS_ROOT=$skill_roots/codex" "FANLOOP_AGENT_SKILLS_ROOT=$skill_roots/agent"
  "FANLOOP_TRAE_SKILLS_ROOT=$skill_roots/trae" "FANLOOP_CLAUDE_SKILLS_ROOT=$skill_roots/claude")
validate_controller() {
  env -u FANLOOP_CONFIG_SOURCE "${controller_env[@]}" "$controller_binary" flow status --root "$requirement_root" >/dev/null
  env -u FANLOOP_CONFIG_SOURCE "${controller_env[@]}" "$controller_binary" doctor >/dev/null
}
if [[ -e "$controller_home" || -L "$controller_home" ]]; then
  [[ -x "$controller_binary" ]] || { echo "existing pinned controller is invalid" >&2; exit 1; }
  validate_controller
  printf '%s\n' "$controller_binary"
  exit 0
fi
source_home="${FANLOOP_DATA_HOME:-$HOME/.fanloop}"
source_release="$(cd "$source_home/current/release" && pwd -P)"
source_binary="$source_release/bin/fanloop"
"$source_binary" flow status --root "$requirement_root" >/dev/null
mkdir "$controller_home"
trap 'rm -rf -- "$controller_home"' EXIT
config_args=()
if [[ ${FANLOOP_CONFIG_SOURCE+x} ]]; then
  [[ -n "$FANLOOP_CONFIG_SOURCE" ]] || { echo "FANLOOP_CONFIG_SOURCE must not be empty" >&2; exit 1; }
  config_args=(--config-source "$FANLOOP_CONFIG_SOURCE")
else
  # Resolve once; the selected committed snapshot cannot change while it is copied.
  snapshot="$(cd "$source_home/config/current" && pwd -P)"
  mkdir -p "$controller_home/config/snapshots"
  cp -R "$snapshot" "$controller_home/config/snapshots/$(basename "$snapshot")"
  ln -s "$controller_home/config/snapshots/$(basename "$snapshot")" "$controller_home/config/current"
fi
"$source_binary" __install --source "$source_release" ${config_args[@]+"${config_args[@]}"} \
  --data-root "$controller_home" --codex-skills-root "$skill_roots/codex" \
  --agent-skills-root "$skill_roots/agent" --trae-skills-root "$skill_roots/trae" \
  --claude-skills-root "$skill_roots/claude" --replace-invalid >/dev/null
validate_controller
trap - EXIT
printf '%s\n' "$controller_binary"
