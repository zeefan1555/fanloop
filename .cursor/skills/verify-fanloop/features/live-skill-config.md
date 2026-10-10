# Workflow Configuration Snapshots

## Sub-features

- second-brain-agent main is the single source, with complete skills/fanloop-workflow packages.
- New flow init fetches, validates and activates an immutable commit snapshot.
- Existing status/report/skills are offline; explicit repository source bypasses refresh.
- Complete path/content/executable-bit identity, shared locking, no partial activation.

## How to get to it (user POV)

Install into isolated data and Skill roots without configuration. Run flow init from an unrelated directory. Read Doctor, config/current/configuration.json and Status Skill paths; they identify the same commit snapshot. Explicit FANLOOP_CONFIG_SOURCE selects an author repository root.

## Driving it with fanloop

Use a local Git fixture through Git's url.insteadOf for repeatable remote-main tests, plus one authorized real GitHub main smoke. Initialize two independent Roots at the same SHA; confirm snapshot reuse. Advance main and delete a resource/package; the new snapshot must exactly match the new collection. Existing paths remain readable. Block Git and prove existing status/report/skills succeed while a fresh init returns CONFIGURATION_FETCH_FAILED, external, exit 1, retryable, with no State or current change. Invalid and symlink packages fail before activation. Dry-run validates without State or current change. Exercise concurrent inits and failed activation. Hold an existing Root State lock while a valid report waits; a refresh may prepare its candidate but must not switch the binding until that report completes. The report must succeed against one snapshot.

Use an explicit author source to prove immediate edit visibility and invalid-source rejection. scripts/validate-configuration.py validates the fixed full collection, records its complete manifest, supports affected --workflow selection and checks unchanged source. Unknown IDs fail before matrices.

## Gotchas

Snapshots are immutable; edit and commit the source repository. Default Dashboard is read-only. Author roots use the same category layout. No manual update, tag or provider selector exists. Capture configuration_manifest_sha256 with the frozen code candidate.
