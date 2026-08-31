#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || "$1" != /* || ! -d "$1" || ! -f "$1/.fanloop/flow/state.json" ]]; then
  echo "usage: pin-controller-release.sh <ABSOLUTE_INITIALIZED_REQUIREMENT_ROOT>" >&2
  exit 2
fi

requirement_root="$(cd "$1" && pwd -P)"
source_release="$(cd "$HOME/.fanloop/current" && pwd -P)"
source_binary="$source_release/bin/fanloop"
controller_home="$requirement_root/bound-release-home"
skill_roots="$controller_home/skill-roots"

env \
  -u FANLOOP_DATA_HOME \
  -u FANLOOP_CODEX_SKILLS_ROOT \
  -u FANLOOP_AGENT_SKILLS_ROOT \
  -u FANLOOP_TRAE_SKILLS_ROOT \
  -u FANLOOP_CLAUDE_SKILLS_ROOT \
  "$source_binary" flow status --root "$requirement_root" >/dev/null

"$source_binary" __install \
  --source "$source_release" \
  --data-root "$controller_home" \
  --codex-skills-root "$skill_roots/codex" \
  --agent-skills-root "$skill_roots/agent" \
  --trae-skills-root "$skill_roots/trae" \
  --claude-skills-root "$skill_roots/claude" \
  --replace-invalid >/dev/null

controller_binary="$controller_home/current/bin/fanloop"
controller_env=(
  "FANLOOP_DATA_HOME=$controller_home"
  "FANLOOP_CODEX_SKILLS_ROOT=$skill_roots/codex"
  "FANLOOP_AGENT_SKILLS_ROOT=$skill_roots/agent"
  "FANLOOP_TRAE_SKILLS_ROOT=$skill_roots/trae"
  "FANLOOP_CLAUDE_SKILLS_ROOT=$skill_roots/claude"
)

env "${controller_env[@]}" "$controller_binary" flow status --root "$requirement_root" >/dev/null
doctor="$(env "${controller_env[@]}" "$controller_binary" doctor)"
if [[ "$doctor" != *'"status": "healthy"'* ]]; then
  echo "$doctor" >&2
  exit 1
fi

printf '%s\n' "$controller_binary"
