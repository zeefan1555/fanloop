---
status: accepted
date: 2026-10-16
amends: ADR-0053-refresh-release-only-before-workflow-init, ADR-0132-git-workflow-configuration-center, ADR-0136-distribute-cli-from-github-releases, ADR-0137-take-workflow-configuration-from-fornax
amended_by: ADR-0139-update-config-from-github
---

# Refresh the CLI and Git configuration on a fresh init

A fresh non-dry-run `flow init` refreshes two things before it creates Requirement state, then continues with whatever is already installed if either refresh fails. Existing requirements are unchanged: init returns `ALREADY_INITIALIZED` before either refresh, and a requirement keeps the snapshot path recorded in `.fanloop/configuration-snapshot`.

## CLI release

When `current` is a symlink to `releases/<version>`, init asks GitHub for the latest release and compares stable versions. The installed version is already the latest, or newer than latest, when no release asset is downloaded and the process continues. A newer release is installed with the same checksum verification and atomic `current` switch as `fanloop update`. The new `current/bin/fanloop` is then re-executed with the original arguments. The re-executed process sets `FANLOOP_AUTO_UPDATE_REEXEC=1` so it does not refresh again. That variable is an internal one-shot guard, not a user setting.

`FANLOOP_NO_AUTO_UPDATE=1` (also `true` or `yes`) skips the check. A directory `current`, including a development install, is not a versioned release and is left untouched without a warning. Any other failure, including a timeout of 15 seconds, prints `warning: automatic update failed; checking for an installed fallback` and continues with the binary that is already running. The explicit `fanloop update` command is unchanged, including its same-version repair. This amends ADR-0053 where a failed refresh stopped initialization, and ADR-0136 where only `fanloop update` downloaded a GitHub Release. Dry-run does not refresh the CLI.

## Git configuration is the default

`second-brain-skills` `main` is again the configuration source for `flow init`, including when `~/.fanloop/fornax.json` exists. A fresh init fetches that ref, validates the complete `fanloop-workflow/` collection, and activates an immutable commit snapshot. The same commit reuses the installed snapshot. `FANLOOP_NO_AUTO_CONFIG=1` (also `true` or `yes`) skips the fetch and opens the installed snapshot. A Git or network failure does not replace `config/current`. When a verified snapshot is already installed, init prints `warning: automatic configuration pull failed; continuing with the installed snapshot` and uses it. When none is installed, init still returns `CONFIGURATION_FETCH_FAILED` (type=external, exit=1, retryable=true) and creates no Requirement. An invalid candidate still fails and leaves `current` in place. These warnings are not error codes and do not change a successful exit status.

`FANLOOP_CONFIG_SOURCE` still selects an author checkout and skips both remotes. Dry-run still does not activate a snapshot. This amends ADR-0132 only where a fetch failure with an existing snapshot used to fail the new init. It amends ADR-0137 where the presence of `fornax.json` made Fornax the source for `flow init`.

## Fornax stays opt-in

The Fornax client, snapshot format, `fanloop update config`, and `fanloop config drift` stay. They are not removed and they are not run by `flow init`. Removing them would discard the explicit sync and drift commands and the snapshots already installed from released packages. The smaller change is that `fornax.json` no longer switches the init source. `FANLOOP_USE_FORNAX=1` (also `true` or `yes`), together with `fornax.json`, selects the installed Fornax snapshot instead of the Git fetch. There is no Git-to-Fornax publish path and no automatic Fornax sync on init.
