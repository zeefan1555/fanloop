#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
build_root="$("$repo_root/scripts/build-local.sh" "$@")"

# The runtime owns validation, user-path protection and the atomic current switch.
"$build_root/bin/fanloop" __install --source "$build_root" \
  --data-root "${FANLOOP_DATA_HOME:-$HOME/.fanloop}" \
  --codex-skills-root "${FANLOOP_CODEX_SKILLS_ROOT:-$HOME/.codex/skills}" \
  --agent-skills-root "${FANLOOP_AGENT_SKILLS_ROOT:-$HOME/.agents/skills}" \
  --trae-skills-root "${FANLOOP_TRAE_SKILLS_ROOT:-$HOME/.trae/skills}" \
  --claude-skills-root "${FANLOOP_CLAUDE_SKILLS_ROOT:-$HOME/.claude/skills}"
