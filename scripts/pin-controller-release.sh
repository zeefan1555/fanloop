#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || "$1" != /* || ! -d "$1" || ! -f "$1/.fanloop/flow/state.json" ]]; then
  printf '%s\n' 'usage: pin-controller-release.sh <ABSOLUTE_INITIALIZED_REQUIREMENT_ROOT>' >&2
  exit 2
fi
requirement_root="$1"
source_home="${FANLOOP_DATA_HOME:-$HOME/.fanloop}"
manager="${FANLOOP_PROTECT_BINARY-$source_home/current/bin/fanloop}"
if [[ "$manager" != /* || ! -f "$manager" || ! -x "$manager" ]]; then
  printf '%s\n' 'protect manager must be an explicit executable file; set FANLOOP_PROTECT_BINARY to the candidate binary' >&2
  exit 1
fi
if [[ ${FANLOOP_PROTECT_BINARY+x} ]]; then
  manager_parent="$(cd -P -- "$(dirname -- "$manager")" && pwd -P)"
  if [[ -L "$manager" || "$manager_parent/$(basename -- "$manager")" != "$manager" ]]; then
    printf '%s\n' 'FANLOOP_PROTECT_BINARY must name a clean absolute physical binary without linked ancestors' >&2
    exit 1
  fi
fi
if ! manager_help="$("$manager" __protect --help 2>/dev/null)" || [[ "$manager_help" != *'fanloop __protect <absolute-root> [flags]'* ]]; then
  printf '%s\n' 'current binary does not support protect; explicitly select the candidate manager with FANLOOP_PROTECT_BINARY' >&2
  exit 1
fi
source_args=()
if [[ ${FANLOOP_CONTROLLER_RELEASE+x} ]]; then
  if [[ -z "$FANLOOP_CONTROLLER_RELEASE" || "$FANLOOP_CONTROLLER_RELEASE" != /* ]]; then
    printf '%s\n' 'FANLOOP_CONTROLLER_RELEASE must be an explicit absolute physical Release' >&2
    exit 1
  fi
  source_args=(--source "$FANLOOP_CONTROLLER_RELEASE")
fi
exec "$manager" __protect ${source_args[@]+"${source_args[@]}"} "$requirement_root"
