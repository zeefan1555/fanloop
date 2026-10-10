#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_args=()
if [[ ${FANLOOP_CONFIG_SOURCE+x} ]]; then config_args=(--config-source "$FANLOOP_CONFIG_SOURCE"); fi
build_args=()
while [[ $# -gt 0 ]]; do
  if [[ "$1" == --config-source ]]; then
    [[ $# -ge 2 && -n "$2" ]] || { echo "--config-source requires a repository path" >&2; exit 2; }
    config_args=(--config-source "$2"); shift 2
  else build_args+=("$1"); shift; fi
done
build_root="$("$repo_root/scripts/build-local.sh" ${build_args[@]+"${build_args[@]}"})"

# The runtime owns validation, user-path protection and the atomic current switch.
"$build_root/bin/fanloop" __install --source "$build_root" \
  ${config_args[@]+"${config_args[@]}"} \
  --data-root "${FANLOOP_DATA_HOME:-$HOME/.fanloop}" \
  --codex-skills-root "${FANLOOP_CODEX_SKILLS_ROOT:-$HOME/.codex/skills}" \
  --agent-skills-root "${FANLOOP_AGENT_SKILLS_ROOT:-$HOME/.agents/skills}" \
  --trae-skills-root "${FANLOOP_TRAE_SKILLS_ROOT:-$HOME/.trae/skills}" \
  --claude-skills-root "${FANLOOP_CLAUDE_SKILLS_ROOT:-$HOME/.claude/skills}"
