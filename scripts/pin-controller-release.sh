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
controller_env=(
  "FANLOOP_DATA_HOME=$controller_home"
  "FANLOOP_CODEX_SKILLS_ROOT=$skill_roots/codex"
  "FANLOOP_AGENT_SKILLS_ROOT=$skill_roots/agent"
  "FANLOOP_TRAE_SKILLS_ROOT=$skill_roots/trae"
  "FANLOOP_CLAUDE_SKILLS_ROOT=$skill_roots/claude"
)

validate_controller() {
  env -u FANLOOP_CONFIG_SOURCE "${controller_env[@]}" "$controller_binary" flow status --root "$requirement_root" >/dev/null || return
  local doctor
  if ! doctor="$(env -u FANLOOP_CONFIG_SOURCE "${controller_env[@]}" "$controller_binary" doctor 2>&1)"; then
    echo "$doctor" >&2
    return 1
  fi
  if [[ "$doctor" != *'"status": "healthy"'* ]]; then
    echo "$doctor" >&2
    return 1
  fi
}

if [[ -e "$controller_home" || -L "$controller_home" ]]; then
  if [[ ! -x "$controller_binary" ]]; then
    echo "existing pinned controller is invalid: $controller_home" >&2
    exit 1
  fi
  validate_controller
  printf '%s\n' "$controller_binary"
  exit 0
fi

source_home="${FANLOOP_DATA_HOME:-$HOME/.fanloop}"
source_config="${FANLOOP_CONFIG_SOURCE-$(readlink "$source_home/config/current")}"
source_release="$(cd "$source_home/current/release" && pwd -P)"
source_binary="$source_release/bin/fanloop"

env FANLOOP_DATA_HOME="$source_home" FANLOOP_CONFIG_SOURCE="$source_config" \
  "$source_binary" flow status --root "$requirement_root" >/dev/null
mkdir "$controller_home"
trap 'rm -rf -- "$controller_home"' EXIT

# Pinning requires Python 3; the ordinary CLI does not. Use the CLI's source
# directory flock so concurrent Dashboard saves cannot produce a mixed snapshot.
python3 - "$source_config" "$controller_home/source" <<'PY'
import fcntl
import os
import shutil
import stat
import sys

source, target = sys.argv[1:]
root = os.path.join(source, "skills")
if not os.path.isabs(source) or source == "/":
    raise ValueError("configuration source must be an absolute repository directory")
for path in (source, root):
    if not stat.S_ISDIR(os.lstat(path).st_mode):
        raise ValueError(f"configuration source must use real directories: {path}")
fd = os.open(root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
try:
    fcntl.flock(fd, fcntl.LOCK_SH)
    os.mkdir(target)
    shutil.copytree(root, os.path.join(target, "skills"), symlinks=True)
finally:
    os.close(fd)
PY

"$source_binary" __install \
  --source "$source_release" \
  --config-source "$controller_home/source" \
  --data-root "$controller_home" \
  --codex-skills-root "$skill_roots/codex" \
  --agent-skills-root "$skill_roots/agent" \
  --trae-skills-root "$skill_roots/trae" \
  --claude-skills-root "$skill_roots/claude" \
  --replace-invalid >/dev/null

validate_controller

trap - EXIT
printf '%s\n' "$controller_binary"
