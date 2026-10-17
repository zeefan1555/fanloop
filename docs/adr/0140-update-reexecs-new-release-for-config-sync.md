---
status: accepted
date: 2026-10-17
amends: ADR-0139-update-config-from-github, ADR-0138-refresh-cli-and-git-config-on-fresh-init
---

# fanloop update re-executes the new release for configuration sync

`fanloop update` switches `current` to the downloaded GitHub Release and then synchronizes Workflow configuration. After a changed release, that sync runs in the new binary.

## Release switch

After printing `updated <version>`, a changed release (`!Unchanged`) whose process does not already have `FANLOOP_UPDATE_REEXEC=1` executes `<DATA>/current/bin/fanloop` with argv `update config` and environment `FANLOOP_UPDATE_REEXEC=1` via `syscall.Exec`. The new process's exit code is the command's exit code. `execUpdatedBinary` is the injected function; production uses `syscall.Exec`.

A same-version repair (`Unchanged`) and a process that already has `FANLOOP_UPDATE_REEXEC=1` keep the in-process sync. `fanloop update config` never re-executes. `FANLOOP_CONFIG_SOURCE` is still refused by whichever process performs the sync.

If exec returns an error, the command prints `warning: could not start the updated binary (<err>); synchronizing with the running binary` to stderr and falls back to the in-process sync. That sync result decides the exit code.

This amends ADR-0139 where `fanloop update` performed the Git sync in the process that downloaded the release. It amends ADR-0138 only where that ADR says the explicit `fanloop update` command is unchanged: same-version repair stays in-process; a changed release now hands configuration sync to the new binary. `flow init` still uses `FANLOOP_AUTO_UPDATE_REEXEC`.

## npm postinstall

After `switchCurrent` succeeds, the npm installer runs `<dataHome>/current/bin/fanloop update config` unless `FANLOOP_NO_AUTO_CONFIG` is `1`, `true`, or `yes`, or `FANLOOP_CONFIG_SOURCE` is set. The child inherits stdio and the environment, with a 120 second timeout. A non-zero exit or a timeout prints `fanloop install: configuration sync failed; run "fanloop update config"` and the postinstall still exits 0. The CLI is already installed.

## Shipped 0.6.4 cannot be fixed by a later release

A 0.6.4 binary that runs `fanloop update` downloads the new release, switches `current`, and then calls the Fornax sync compiled into 0.6.4. That call `lstat`s `<DATA>/fornax.json` and exits 1. `Update` does not execute anything from the new tree. No file added to a later release archive, and no change to the GitHub latest payload, runs inside that process. The one-shot failure remains for that invocation. The recovery is a second `fanloop update config` on the new `current`, an npm install of a release whose postinstall syncs configuration, or `fanloop flow init`, which already re-executes through `RefreshCLI`.
