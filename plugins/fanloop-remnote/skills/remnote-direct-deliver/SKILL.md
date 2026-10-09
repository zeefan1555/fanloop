---
name: remnote-direct-deliver
description: "Push an accepted RemNote Direct candidate and adopt it locally when that impact requires it. Use when remnote-direct-verify has passed for the same candidate. Do not publish a marketplace release."
---

# Deliver

Pass `change_delivered` only after the remote SHA, applicable CI, local adoption, and smoke are read back. Record `candidate_head`, `remote_head`, `ci`, `installed_artifacts`, `smoke`, and `record_path`. A pushed commit with failed CI or smoke is not delivered. Install or smoke problems on the same candidate stay here. Do not call `fanloop flow`. The route table is in [remnote-direct-maintainer](../remnote-direct-maintainer/SKILL.md).

Recheck the frozen source/artifacts and Verify evidence. Push the accepted commit
to the recorded Git destination using repository policy. If protection requires a
PR, follow that route; never force-push or bypass protection. Read back the actual
remote SHA and required CI results for that SHA. Missing, failed or cancelled
required checks block completion. No-CI repositories need an explicit, verified
not_applicable reason; absent evidence is not success.

If integration changes source content, return to Build and verify the new
candidate. If a merge only changes commit identity, prove the tree/artifact
equivalence and recheck the exact integrated commit's applicable CI before adoption.
Retain the source and integrated identities instead of calling them the same SHA.

After applicable remote checks pass, adopt only affected local components using
existing repository helpers. CLI-only changes update the CLI entry and preserve
plugin assets. Plugin changes replace the verified static files and reload the
existing plugin. Source push is not plugin deployment; neither implies marketplace
publication. Record the real installed path, version and artifact digests.

Use the installed `remnote-cli` directly for final smoke. Installation or lifecycle
changes require normal restart/cold-start checks and readback of the already known
test IDs; do not create replacement test objects to conceal persistence failure.
Documentation-only delivery has no installation or KB-write side effects.

Any failed required acceptance prevents adoption. Restore only the recorded prior
local installation when necessary, retain the failed candidate and explicit
outcome, and reconcile any partial KB writes without claiming atomic rollback.
A pushed commit with failed CI/smoke is not a completed change.

Finish the single local record with remote/CI/install identities, applicable
checks, test IDs, timings, unsupported/untested capabilities and restoration state.
Never include pairing keys, private journals, knowledge bodies or screenshots in Git.
