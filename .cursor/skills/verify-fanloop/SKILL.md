---
name: verify-fanloop
description: Drive Fanloop with the public CLI and the Feature Map, then leave a verify manifest that survives cleanup. Use before self-accepting a Fanloop change, before a result whose output source is verify.evidence.outcome, and when that result returns OUTPUT_INVALID.
---

# Verify Fanloop

This is the single code-owned verification procedure. `features/README.md` indexes its Feature Map. Workflow package Skills point here; they do not carry another copy.

## Launch

From a clean candidate, use isolated data and four Skill roots. The production source is second-brain-agent main. Generic runtime verification uses a fresh Git repository containing only tracked synthetic fixtures in skills/fanloop-workflow. Commit the fixture before offline installation.

```bash
export FANLOOP_DATA_HOME="$(mktemp -d)"
export FANLOOP_CODEX_SKILLS_ROOT="$FANLOOP_DATA_HOME/skill-roots/codex"
export FANLOOP_AGENT_SKILLS_ROOT="$FANLOOP_DATA_HOME/skill-roots/agent"
export FANLOOP_TRAE_SKILLS_ROOT="$FANLOOP_DATA_HOME/skill-roots/trae"
export FANLOOP_CLAUDE_SKILLS_ROOT="$FANLOOP_DATA_HOME/skill-roots/claude"
export FANLOOP_CONFIG_SOURCE="$(mktemp -d)"
mkdir -p "$FANLOOP_CONFIG_SOURCE/skills/fanloop-workflow"
cp -R internal/workflow/testdata/linear-runtime internal/workflow/testdata/runtime-controls "$FANLOOP_CONFIG_SOURCE/skills/fanloop-workflow/"
git -C "$FANLOOP_CONFIG_SOURCE" init -b main
git -C "$FANLOOP_CONFIG_SOURCE" add .
git -C "$FANLOOP_CONFIG_SOURCE" -c user.name=Verification -c user.email=verification@example.invalid commit -m 'Synthetic configuration'
unset BOTMUX_CHAT_ID BOTMUX_SESSION_ID
./scripts/install-local.sh --config-source "$FANLOOP_CONFIG_SOURCE"
export PATH="$FANLOOP_DATA_HOME/current/bin:$PATH"
```

Keep the explicit fixture source for credential-free smoke. To prove remote production initialization, use a separate data home, unset FANLOOP_CONFIG_SOURCE, install without a source, then run flow init and record the exact remote main SHA, configuration manifest and snapshot Skill paths. First-install Doctor warns until configuration exists; malformed explicit sources and broken bindings fail. Existing status/report/skills stay offline. Record authorized remote Git reads; do not execute business Skills.

Ready when `fanloop version` reports `commit_sha` equal to `git rev-parse HEAD` and the previous global `~/.fanloop/current` target is unchanged.

Before Launch, use the approved risk level to choose the smallest Feature Impact Set that still covers every changed user
surface. Build in small batches with focused checks; reserve this full isolated run for the stable candidate.

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
fanloop flow report result --root "$ROOT" --step-id verify_behavior \
  --condition-result '{"condition_id":"delivery_verified","output":{"type":"enum_value","value":"passed"}}' \
  --evidence '{"source":"file","content":"fanloop verify manifest","ref":"'"$EVIDENCE_DIR"'/manifest.json"}' \
  --next-step-id merge_and_release --summary "verification manifest passed"
```

Read each leaf `--help`, create an empty absolute Requirement directory (`INVALID_ARGUMENT` if it does not exist), and read fresh `flow status` before Workflow actions. Run a write's `--dry-run`, verify that State/Output/Event/Trace/Card business files did not change, then invoke and read back the real action. Dry-run may append `.fanloop/log/cli.jsonl`; flow init dry-run fetches and validates a temporary configuration without activation. Never equate dry-run with zero filesystem activity. Use synthetic data. External Trace/Lark writes require explicit authorization of identity, target and timing.

## Evidence

`fanloop verify smoke` prints `run_id`, `evidence_dir`, and `outcome`. The proof file is `$EVIDENCE_DIR/manifest.json`. It contains `schema_version` 1 and `outcome` of `passed`, `failed`, or `blocked`, plus the commands that produced it. Exit code 1 means failed or blocked. Read `outcome` from the manifest, not from the exit code.

Record exact candidate HEAD and binary digest, Feature/variant, argv/stdin/stdout/stderr/exit code, before/after Status and stable final state, relevant State/Output/Event/Trace/Card, `.fanloop/log/cli.jsonl` and observable side effects. Read events through public `trace status`/`trace render` and reconcile the `.fanloop/trace/events.jsonl` tail. Internal Go calls, a final output alone, or an old candidate cannot prove the public journey. Evidence permissions must limit access to the current user; do not copy historical Requirements or user credentials into evidence.

After the manifest passes, perform Pre-Certify once: require a clean worktree and record one candidate identity containing
HEAD, Git tree, patch digest, binary digest, verification contract digest, Feature Impact Set digest and, when production
configuration is in scope, `configuration_manifest_sha256`. Independent Code
Review and black-box verification may then run in parallel only with that same identity and isolated writable roots. Wait
for both receipts. Stage this Skill and `features/` from the frozen commit into a read-only verification directory for a
no-source-context verifier, and verify its digest before dispatch. Any source, test, verification-asset or configuration
candidate change invalidates the manifest and both certification results.

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

## Failure modes

`delivery_verified`, `blackbox_verification_passed`, `blackbox_verification_failed`, `post_merge_smoke_passed`, and `post_merge_smoke_failed` accept a result only when file evidence points at one absolute regular `manifest.json` written by `fanloop verify smoke`. The file is schema 1. Its `outcome` equals the submitted enum. `fanloop verify snapshot` captures public projections and durable files but does not write an outcome manifest. If a broader Acceptance Set fails while smoke passes, preserve the snapshot and original receipt, then use the Workflow's explicit changes-requested route instead of claiming a failed manifest.

`OUTPUT_INVALID` with source `verify.evidence.outcome` means that match failed. The path was missing or unreadable, was not an absolute regular `manifest.json`, was not schema 1, or its `outcome` differed. Two `manifest.json` evidence items fail the same way.

Re-run `fanloop verify smoke` and read `outcome` from its new manifest. Submit that enum with that absolute path. Use `fanloop verify snapshot` only for supplementary inventory and public projections. `blocked` matches neither a passed nor a failed condition. Report progress blocked and attach the smoke manifest. Do not edit the JSON.

Do not submit `passed` without that verify run. Do not hand-write `manifest.json`.
