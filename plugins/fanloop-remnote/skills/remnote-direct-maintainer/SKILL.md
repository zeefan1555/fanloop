---
name: remnote-direct-maintainer
description: "Change the separate RemNote Direct CLI or plugin repository. Use when the user explicitly authorizes work on remnote-cli, the Direct plugin, its protocol, install path, or task-plan code. Do not use this skill to create ordinary flashcards. Follow define, build, verify, and deliver below; do not call fanloop flow."
---

# RemNote Direct maintainer

This skill is the Cursor entry for maintaining RemNote Direct. Ordinary card creation uses [remnote-flashcards](../remnote-flashcards/SKILL.md). There is no Fanloop engine here: do not run `fanloop flow`. The four steps below are the workflow, and the pass/fail lines are the old route gates.

Repository instructions in the Direct repo supply the source path, commands, Git destination, supported install method, and the authorized test environment. Do not infer a knowledge base, day, HOME tag, or title from an example.

At the start of each step, send one short user-visible status: the step id, what you are about to do, and which gate is still open. That status is not a completion receipt. If you cannot message the user, stop. Do not write `artifacts/panorama-*` or treat a saved file as delivery.

Read the matching procedure skill before acting:

| Step | Procedure |
| --- | --- |
| `define_change` | [remnote-direct-define](../remnote-direct-define/SKILL.md) |
| `build_candidate` | [remnote-direct-build](../remnote-direct-build/SKILL.md) |
| `verify_candidate` | [remnote-direct-verify](../remnote-direct-verify/SKILL.md) |
| `deliver_change` | [remnote-direct-deliver](../remnote-direct-deliver/SKILL.md) |

One local `delivery-record.md` stays outside tracked source. Flow notes keep IDs, hashes, counts, and conclusions. They do not keep credentials or knowledge-base text.

## Invariants

- Select every affected impact row (documentation, cli_read, cli_write, plugin_protocol, installation). Mixed work takes the union. A two-card smoke is not universal acceptance.
- Freeze one candidate. A later edit to a candidate file invalidates verify and deliver for that identity.
- `partial` or `unknown` knowledge-base writes stop further writes. Keep the plan, key, and known IDs. Read them back. Do not auto-replay, change the key, or delete and recreate.
- A local install restore is not a knowledge-base rollback.
- Do not add an MCP server, a compatibility path, or an arbitrary JavaScript executor.
- Do not publish a marketplace release. Do not force-push or bypass branch protection.
- No pairing secrets, private journals, knowledge bodies, or screenshots in Git.

## 1. `define_change`

Pass `change_defined` only when the recorded object has all of: `repository_root`, `record_path`, `baseline`, `impact`, `acceptance_cases`, `delivery_policy`, and the allowed side effects. Paths, the Git target, and the knowledge base come from this repository and this environment.

Fail closed:

- Missing real dependency, path, or authorization: blocked. Do not record `change_defined`.
- `scope_needs_fix`: return to this step.

## 2. `build_candidate`

Pass `candidate_built` only when the recorded object has all of: `candidate_head`, `source_digest`, `artifact_digests`, `checks`, `record_path`, and the actual impact. Every applicable local check has passed, and the candidate commit is frozen. Do not push in this step.

Fail closed:

- Scope, authorization, impact, or acceptance changed: `scope_needs_fix`, back to `define_change`. The previous candidate is not accepted.
- Build or local check failed, or the diff exceeds the recorded scope: `build_needs_fix`, stay on this step. Classify the failure as product, test, environment, or contract drift. Do not drop a check to get a pass.
- Do not widen permissions or change the impact class to make the candidate fit.

## 3. `verify_candidate`

Pass `candidate_verified` only when the recorded object has all of: `candidate_head`, `artifact_digests`, `case_results`, `not_applicable`, and `record_path`. The commit and digests still match Build. Each applicable case has raw command output and native readback. `not_applicable` needs an impact-based reason. Unavailable required behavior is failed or blocked, not `not_applicable`. A fixture simulation proves only the scope it states.

Fail closed:

- `scope_needs_fix`: back to `define_change`.
- Candidate files must change: `build_needs_fix`, back to `build_candidate`. Old verify evidence does not carry over.
- The acceptance method itself is wrong, on the same candidate: `acceptance_needs_fix`, stay on this step.
- Environment unreachable, or a write result is `partial` or `unknown`: blocked. Do not loop and do not replay.

## 4. `deliver_change`

Pass `change_delivered` only when the recorded object has all of: `candidate_head`, `remote_head`, `ci`, `installed_artifacts`, `smoke`, and `record_path`. The pushed SHA is the accepted candidate, or a merge whose tree and artifacts were proved equivalent and whose own CI was rechecked. Applicable remote checks and local adoption readback are done. A documentation-only change with no runtime effect needs an explicit `not_applicable` reason. Missing evidence is not success.

Fail closed:

- `scope_needs_fix`: back to `define_change`.
- `build_needs_fix`: back to `build_candidate`.
- `acceptance_needs_fix`: back to `verify_candidate`.
- Push, CI, install, or smoke failed for this candidate: `delivery_needs_fix`, stay on this step. A pushed commit with failed CI or smoke is not delivered.
- Required check missing, failed, or cancelled: blocked. Do not adopt.

After `change_delivered`, send a separate user-visible close: remote SHA, CI, installed path and digests, smoke, untested capabilities, and any restore. The status line from the start of this step is not that close.

## Repair

Record the failure class, the raw evidence, and side effects that already happened. Return to the earliest responsible step in the table above. A return trip invalidates later step outputs. Recover old IDs and the install baseline from the local record and read them again. Do not auto-retry or clean up an unknown knowledge-base write. If the environment is unreachable, stop.
