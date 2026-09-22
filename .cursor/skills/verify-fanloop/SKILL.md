---
name: verify-fanloop
description: Drive Fanloop with the public CLI and the Feature Map, then leave a verify manifest that survives cleanup. Use before self-accepting a Fanloop change, or before a result whose output source is verify.evidence.outcome.
---

# Verify Fanloop

The drive procedure and Feature Map live in `workflows/fanloop-maintainer/skills/fanloop-dev-verify/`. This file is the Cursor entry. `features/` is that map. Do not copy it.

## Launch

From the repository root, install the candidate into an isolated data home. Do not point `FANLOOP_DATA_HOME` at `~/.fanloop`.

```bash
export FANLOOP_DATA_HOME="$(mktemp -d)"
export FANLOOP_CODEX_SKILLS_ROOT="$FANLOOP_DATA_HOME/skill-roots/codex"
export FANLOOP_AGENT_SKILLS_ROOT="$FANLOOP_DATA_HOME/skill-roots/agent"
export FANLOOP_TRAE_SKILLS_ROOT="$FANLOOP_DATA_HOME/skill-roots/trae"
export FANLOOP_CLAUDE_SKILLS_ROOT="$FANLOOP_DATA_HOME/skill-roots/claude"
mkdir -p "$FANLOOP_CODEX_SKILLS_ROOT" "$FANLOOP_AGENT_SKILLS_ROOT" "$FANLOOP_TRAE_SKILLS_ROOT" "$FANLOOP_CLAUDE_SKILLS_ROOT"
unset BOTMUX_CHAT_ID BOTMUX_SESSION_ID
./scripts/install-local.sh
export PATH="$FANLOOP_DATA_HOME/current/bin:$PATH"
```

Ready when `fanloop version` reports `commit_sha` equal to `git rev-parse HEAD` and the previous global `~/.fanloop/current` target is unchanged.

Teardown is Cleanup. Kill only processes you started. This CLI is short-lived. There is no server to keep up.

## Doctor

Run this before the first drive, and again after any surprise. Stop if any command is unhealthy.

```bash
fanloop version
fanloop doctor
fanloop verify doctor
```

`fanloop doctor` and `fanloop verify doctor` are different commands. `verify doctor` checks the isolated executable, data root, cleared Botmux binding, `version`, and `doctor`. It writes no manifest. A blocked doctor is not a bundle.

## Drive

Read `features/README.md`. Drive the feature you changed. For a cold proof, drive `features/installation-health.md` with the public binary only.

```bash
fanloop version
fanloop doctor
fanloop verify doctor
fanloop verify smoke
```

`verify smoke` is the fixed isolated journey. It is not an Acceptance Set and not the route matrix. Leaf `--help` is the input contract. Do not call internal Go.

A condition whose output `source` is `verify.evidence.outcome` accepts a result only when file evidence points at a schema-1 `manifest.json` and `outcome` equals the submitted enum (`passed`, `failed`, or `blocked`). `blocked` does not match a pass or fail condition. Record it with `flow report progress --status blocked` and the same file evidence. The manifest must exist before `--dry-run` and before the real report.

```bash
fanloop flow report result --root "$ROOT" --step-id verify_report \
  --condition-result '{"condition_id":"delivery_verified","output":{"type":"enum_value","value":"passed"}}' \
  --evidence '{"source":"file","content":"fanloop verify manifest","ref":"'"$EVIDENCE_DIR"'/manifest.json"}' \
  --terminal --summary "verification manifest passed"
```

## Evidence

`fanloop verify smoke` prints `run_id`, `evidence_dir`, and `outcome`. The proof file is `$EVIDENCE_DIR/manifest.json`. It contains `schema_version` 1 and `outcome` of `passed`, `failed`, or `blocked`, plus the commands that produced it. Exit code 1 means failed or blocked. Read `outcome` from the manifest, not from the exit code.

Keep this directory outside the requirement root and outside `verification/work`. Cleanup must not delete it.

## Cleanup

Smoke already removes its work root and keeps the evidence directory. A later cleanup is idempotent.

```bash
fanloop verify cleanup --run-id "$RUN_ID" --dry-run
fanloop verify cleanup --run-id "$RUN_ID"
test -f "$EVIDENCE_DIR/manifest.json"
```

Cleanup deletes only `verification/work/$RUN_ID`. It does not delete `verification/runs/` or a custom evidence directory. After cleanup, `manifest.json` must still be readable and the global `~/.fanloop/current` target must still be unchanged. Do not kill processes by name.

## Helpers

There is no helper script. The harness is the isolated `fanloop` binary.
