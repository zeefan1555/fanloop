---
status: accepted
date: 2026-10-17
amends: ADR-0137-take-workflow-configuration-from-fornax, ADR-0138-refresh-cli-and-git-config-on-fresh-init
amended_by: ADR-0140-update-reexecs-new-release-for-config-sync
---

# Sync Workflow configuration from GitHub by default

GitHub `zeefan1555/second-brain-skills` `main` is the only default Workflow configuration source for `flow init`, `fanloop update`, `fanloop update config`, and `fanloop config drift`. Fornax stays in the tree. It runs only when `FANLOOP_USE_FORNAX` is `1`, `true`, or `yes` and `fornax.json` is a regular file. This amends ADR-0137 where `fanloop update`, `fanloop update config`, and `fanloop config drift` always used Fornax, and ADR-0138 where those commands stayed on Fornax after init returned to Git.

`fanloop update config` calls the Git sync. It runs `ls-remote` on `refs/heads/main`, skips checkout when `config/current` already names that commit, and otherwise checks out, validates, publishes `config/snapshots/<sha>`, and activates it under the configuration lock. A missing `fornax.json` is not an error. Success prints `fanloop configuration updated: <sha> (<id>, ...)` or `fanloop configuration is already up to date: <sha> (<id>, ...)`, using the full 40-character commit. Git failure exits 1 with `fanloop update: configuration fetch failed: ...` and leaves `config/current` unchanged. `FANLOOP_CONFIG_SOURCE` is refused because an author source is not synced.

`fanloop update` installs the GitHub Release of the CLI, then uses that same Git sync.

`fanloop config drift` in Git mode reads `config/current` and refuses `FANLOOP_CONFIG_SOURCE`. `configuration_commit` is the snapshot `commit_sha`. `remote_commit` is the `ls-remote` main sha. Every snapshot file must match the manifest sha256; otherwise the status is `local_modified` and the detail is `package content differs`. Equal shas are `in_sync`. Different shas are `local_behind`. An `ls-remote` failure is `unavailable`. A Fornax-origin snapshot is `local_behind` with detail `snapshot came from Fornax; run fanloop update config`. Each workflow id in the snapshot is one `workflows` entry with `workflow_id` and `status` and no `skill_key`. Exit 0 only for `in_sync`.

`WorkflowMirrorCheck.skill_key` is optional. `ConfigDriftResponse.remote_commit` is an optional string. The command summary is `Compare the active Workflow snapshot with second-brain-skills main`. No field ids or enums change.

Doctor keeps printing `commit_sha` for Git snapshots. Fornax code, the Fornax snapshot format, and the opt-in path stay.
