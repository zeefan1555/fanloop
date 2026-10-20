---
status: accepted
date: 2026-10-21
amends: ADR-0132-git-workflow-configuration-center, ADR-0138-refresh-cli-and-git-config-on-fresh-init
---

# Rebind an existing Requirement to an installed configuration snapshot

Existing Requirements remain pinned when `fanloop update config` advances
`config/current`. An operator may now explicitly run `fanloop config rebind` for
one initialized Requirement. The target is either `config/current` or one named,
already installed directory under `config/snapshots`; arbitrary paths and remote
fetches are rejected.

Before changing the pin, Fanloop verifies the immutable snapshot manifest, loads
the same bound Workflow, and validates the current State and complete Event
history against it. The Requirement lock serializes the operation. A transaction
audit first records a prepared entry with the old/new snapshot identities, UTC
time and required reason, then the pin changes, then a committed entry with the
same transaction id is appended. Ordinary failures restore both files; an
interrupted process always leaves a prepared record that identifies the exact
recovery intent rather than an unaudited pin or a false committed result.

This is an explicit local migration, not automatic compatibility. There is no
fallback loader, State rewrite, Workflow alias, implicit rebind after update, or
new remote snapshot acquisition. Status and all later operations continue to use
the single absolute path in `.fanloop/configuration-snapshot`.
