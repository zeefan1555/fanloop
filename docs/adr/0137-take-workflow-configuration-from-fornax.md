---
status: accepted
date: 2026-10-15
supersedes_in_part: ADR-0136-mirror-git-workflows-to-fornax
amends: ADR-0132-git-workflow-configuration-center, ADR-0136-distribute-cli-from-github-releases, ADR-0019-pin-running-requirements-to-workflow-versions
---

# Take Workflow configuration from Fornax releases

When `fornax.json` exists, released Fornax packages are the production Workflow source and the Git checkout is a mirror. This supersedes the statement in ADR-0136-mirror-git-workflows-to-fornax that second-brain-skills `main` is the only production source, and the statement in ADR-0136-distribute-cli-from-github-releases that `fanloop update config` syncs that Git branch. ADR-0132 still applies when `fornax.json` is absent and when `FANLOOP_CONFIG_SOURCE` is set: those paths keep fetching or reading Git.

`fanloop update` still installs the latest GitHub Release of the CLI and does not downgrade a newer local version. It then synchronizes configuration. `fanloop update config` synchronizes configuration without touching the CLI. Both commands discover every configured `{workspace_id, key_prefix, access}` source, keep the latest Online/prod release of each matching key, and download it with the user's logged-in `fornax-cli`. A listed key whose suffix is not `[a-z0-9_]+` is skipped with a warning. An empty Skill id still rejects the set. The program does not embed AK/SK and does not call the Fornax API with built-in keys.

A read_write source wins when the same workflow id is also released from a read_only source. A duplicate id inside one source, or a tie across sources, rejects the whole set and leaves `config/current` unchanged. Downloaded package names `loop-workflow-<id>`, `fanloop-workflow-<id>`, bare `<id>`, and the full Skill key `loop.workflow.<snake_id>` or `fanloop.workflow.<snake_id>` are accepted. The workflow id comes from `workflow.yaml`. A leading `#!` is installed as mode 0555. A read_only package that fails validation is skipped with a warning. A read_write package that fails validation rejects the set.

The snapshot directory name is the sha256 hex of the sorted `key@version` lines. The same digest with different bytes is refused, so an existing snapshot is not overwritten. Activation swaps `config/current` only after the whole set has been validated. A download failure, a read_write validation failure, an activation failure, or a receipt failure leaves the previous `current` in place. The receipt at `$FANLOOP_DATA_HOME/cache/fornax-workflows.json` lets an unchanged released set and an unchanged tree skip the download. The sync holds the existing `.workflows.lock` for the whole operation.

`flow init` uses the active Fornax snapshot when `fornax.json` exists and `FANLOOP_CONFIG_SOURCE` is unset. It does not replace that snapshot with a Git fetch. A requirement writes `.fanloop/configuration-snapshot` before its first commit. Later reads of that requirement use the pinned directory after `config/current` moves. A requirement without the pin still follows `current`.

`fanloop config drift` still reports the existing `in_sync`, `drifted`, and `unavailable` results. The released package becomes the reference, with `local_behind` and `local_modified`, in the follow-up that changes the status enum.
