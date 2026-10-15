# Workflow Configuration Snapshots

## Sub-features

- second-brain-skills main is the single source, with complete fanloop-workflow packages.
- New flow init fetches, validates and activates an immutable commit snapshot.
- Existing status/report/skills are offline; explicit repository source bypasses refresh.
- Complete path/content/executable-bit identity, shared locking, no partial activation.
- Optional personal Fornax mirror with Git-first publication and read-only drift; never a runtime provider.

## How to get to it (user POV)

Install into isolated data and Skill roots without configuration. Run flow init from an unrelated directory. Read Doctor, config/current/configuration.json and Status Skill paths; they identify the same commit snapshot. Explicit FANLOOP_CONFIG_SOURCE selects an author repository root.

## Driving it with fanloop

Use a local Git fixture through Git's url.insteadOf for repeatable remote-main tests, plus one authorized real GitHub main smoke. Initialize two independent Roots at the same SHA; confirm snapshot reuse. Advance main and delete a resource/package; the new snapshot must exactly match the new collection. Existing paths remain readable. Block Git and prove existing status/report/skills succeed while a fresh init returns CONFIGURATION_FETCH_FAILED, external, exit 1, retryable, with no State or current change. Invalid and symlink packages fail before activation. Dry-run validates without State or current change. Exercise concurrent inits and failed activation. Hold an existing Root State lock while a valid report waits; a refresh may prepare its candidate but must not switch the binding until that report completes. The report must succeed against one snapshot.

Use an explicit author source to prove immediate edit visibility and invalid-source rejection. scripts/validate-configuration.py validates the fixed full collection, records its complete manifest, supports affected --workflow selection and checks unchanged source. Unknown IDs fail before matrices.

For mirror coverage, install the candidate into an isolated data home with a committed configuration snapshot and a credential-free `fornax.json`. Put a fake `fornax-cli` on isolated PATH that exposes exact Skill list, Online/prod releases and downloaded package bytes. Run `fanloop config drift`: identical wrapper, source commit and file bytes return `in_sync` and exit 0. Fornax does not preserve Unix file modes, so a mode-only difference also returns `in_sync` and may include an informational detail. Missing or unreleased keys and byte or path changes return explicit drift with exit 1. CLI failure or unsafe download returns unavailable without copying credentials or diagnostics. Confirm Runtime/installer never creates `fornax.json` and ordinary Doctor/Flow calls never invoke the fake CLI.

## Gotchas

Snapshots are immutable; edit and commit the source repository. Default Dashboard is read-only. Author roots use the same category layout. No runtime provider selector exists. The Fornax mirror is checked only by explicit `config drift`; publication occurs only after Git main and local init readback. Capture configuration_manifest_sha256 with the frozen code candidate.
