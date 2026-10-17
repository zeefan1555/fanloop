# Workflow Configuration Snapshots

## Sub-features

- second-brain-skills main is the single source, with complete fanloop-workflow packages.
- New flow init fetches, validates and activates an immutable commit snapshot.
- Existing status/report/skills are offline; explicit repository source bypasses refresh.
- Complete path/content/executable-bit identity, shared locking, no partial activation.
- Default `fanloop config drift` compares the active Git snapshot with second-brain-skills main.
- Optional Fornax sync and drift only when `FANLOOP_USE_FORNAX=1` and `fornax.json` is a regular file.

## How to get to it (user POV)

Install into isolated data and Skill roots without configuration. Run flow init from an unrelated directory. Read Doctor, config/current/configuration.json and Status Skill paths; they identify the same commit snapshot. Explicit FANLOOP_CONFIG_SOURCE selects an author repository root.

## Driving it with fanloop

Use a local Git fixture through Git's url.insteadOf for repeatable remote-main tests, plus one authorized real GitHub main smoke. Initialize two independent Roots at the same SHA; confirm snapshot reuse. Advance main and delete a resource/package; the new snapshot must exactly match the new collection. Existing paths remain readable. Block Git and prove existing status/report/skills succeed while a fresh init returns CONFIGURATION_FETCH_FAILED, external, exit 1, retryable, with no State or current change. Invalid and symlink packages fail before activation. Dry-run validates without State or current change. Exercise concurrent inits and failed activation. Hold an existing Root State lock while a valid report waits; a refresh may prepare its candidate but must not switch the binding until that report completes. The report must succeed against one snapshot.

Use an explicit author source to prove immediate edit visibility and invalid-source rejection. scripts/validate-configuration.py validates the fixed full collection, records its complete manifest, supports affected --workflow selection and checks unchanged source. Unknown IDs fail before matrices.

For the Git default, point `url.insteadOf` at a local main, run `fanloop update config`, then `fanloop config drift`. Equal commit and manifest sha256 return `in_sync` and exit 0. A newer remote commit returns `local_behind`. A byte difference returns `local_modified` with detail `package content differs`. `ls-remote` failure returns `unavailable`. A Fornax-origin snapshot returns `local_behind` with detail `snapshot came from Fornax; run fanloop update config`.

For the Fornax opt-in, install the candidate into an isolated data home with a Fornax snapshot and a credential-free `fornax.json`, and set `FANLOOP_USE_FORNAX=1`. Put a fake `fornax-cli` on isolated PATH that exposes prefix Skill list, Online/prod releases and downloaded package bytes. Run `fanloop config drift`: equal file bytes return `in_sync` and exit 0. A mode-only difference also returns `in_sync`. An older recorded version returns `local_behind`. Other byte differences return `local_modified`. Missing keys, unreleased keys, CLI failure, or an unsafe download return `unavailable` without copying credentials or diagnostics. Confirm Runtime/installer never creates `fornax.json` and ordinary Doctor/Flow calls never invoke the fake CLI. Without the opt-in, `fornax.json` alone still uses Git.

## Gotchas

Snapshots are immutable; edit and commit the source repository, then run `fanloop update config`. Default Dashboard is read-only. Author roots use the same category layout. No runtime provider selector exists. Fornax sync and drift run only behind `FANLOOP_USE_FORNAX=1`. Capture configuration_manifest_sha256 with the frozen code candidate.
