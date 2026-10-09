---
name: remnote-direct-build
description: "Implement a bounded RemNote Direct change, run the applicable checks, and freeze one candidate. Use when remnote-direct-define has passed and the change still needs a frozen candidate. Do not push, and do not use this skill for flashcard authoring."
---

# Build

Pass `candidate_built` only after the applicable local checks pass and the candidate is frozen. Record `candidate_head`, `source_digest`, `artifact_digests`, `checks`, `record_path`, and the actual impact. Scope changes return to Define. A failed check stays here. Do not call `fanloop flow`. The route table is in [remnote-direct-maintainer](../remnote-direct-maintainer/SKILL.md).

Use the recorded scope, actual diff and affected ADRs. Reuse the existing command
framework, task plans, schemas, SDK catalog and repository scripts. Business
composition belongs in the CLI; the plugin owns supported SDK transport, identity,
authorization and serialization. No MCP server, compatibility path or arbitrary
JavaScript executor.

For a behavior change, first run the smallest regression that demonstrates the
missing behavior; fix the shared owner and cover its callers. Use the repository's
existing test runner. Run its applicable focused and full quality checks, generated
SDK freshness check, and production build. Documentation-only work uses relevant
content/link/command checks, not the code or native-write suite. Record commands,
exit codes and full candidate identity; failures are product, test, environment
or contract drift, not reasons to drop checks.

Build in isolation if the daily command links into repository dist. Do not
silently update the user's installed executable by running a build. CLI-only work
keeps plugin source and installed asset digests unchanged; do not rebuild/deploy
the plugin unless the actual impact requires it. Follow the current supported
platform and existing local installation scripts, not a new platform framework.

Recheck the whole diff against scope, ADRs, generated files and private-data
exclusions. Stage owned paths explicitly; do not sweep unrelated dirty files into
the candidate. Commit using repository message/trailer rules, but do not push
before acceptance. Freeze that commit, the complete source identity and applicable
artifact digests; knowledge content and private runtime receipts stay outside Git.
Subsequent candidate edits require a new identity and renewed applicable checks.

Return checks and artifact paths with the candidate. Source/behavior declarations
alone do not prove native persistence; that belongs to Verify.
