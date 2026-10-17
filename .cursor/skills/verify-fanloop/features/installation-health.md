# Installation and Health

## Sub-features

- Offline source build and installation; optional clean Git configuration import.
- GitHub Release install: platform archive, checksums.txt, release.json, the thin npm launcher, public Skill links, and `npm i -g --foreground-scripts --allow-scripts=fanloop`.
- `fanloop update` keeps a newer installed release instead of downgrading to `latest`. `fanloop doctor` warns with `export PATH=...` when that distribution install is not on PATH.
- `fanloop update` installs the CLI. After it switches to a different release, that new release performs the configuration sync from second-brain-skills main. A same-version repair synchronizes in the running process. `fanloop update config` does that Git sync without re-executing, skips an unchanged commit, and prints the full sha plus workflow ids. npm postinstall runs `update config` after switching `current` unless `FANLOOP_NO_AUTO_CONFIG` or `FANLOOP_CONFIG_SOURCE` is set; a sync failure does not fail the install. `FANLOOP_USE_FORNAX=1` with `fornax.json` keeps the Fornax sync and its `id@version` output.
- Release version, commit, binary checksum and both public Skill entrypoints.
- Stable `current/` and nested Release selector, separate from `config/current` source binding.
- Failed installation preserves the active Release and previous source binding.
- Installation-level and Requirement-level Doctor checks.

## How to get to it (user POV)

Run `npm run build` or `./scripts/build-local.sh` to build without activation.
Run `npm run install:local` or `./scripts/install-local.sh` to install, preserving the current snapshot. Use --config-source to import a clean committed configuration repository. Inspect `fanloop version`, `fanloop doctor` and `fanloop verify doctor`.

## Driving it with fanloop

Use isolated data and Skill roots. On macOS invoke the documented installer and pinner with /bin/bash, including no output-directory argument and no configuration source. Install two clean commits, check each immutable
Release, then check `current/release` and the absolute `config/current` snapshot binding and configuration.json identity.
Prove no new `current/workflows` copy exists. The four Skill roots expose only
`fanloop-workflow` and `create-fanloop-workflow`. Run from an unrelated cwd with the
override unset to prove the persisted source is used.

An invalid or missing explicitly supplied source must reject installation without
changing current or the previous binding. Test user-owned path conflicts and checksum
failures. For both default binding and explicit managed snapshot selection, run doctor and verify doctor against healthy, modified-manifest and missing-manifest cases. Include a syntactically valid manifest whose commit_sha differs from the immutable snapshot directory SHA; diagnostics, flow readers and Dashboard must reject it. Both diagnostic commands must reject damaged snapshots; an ordinary author repository without configuration.json remains valid. Remove or corrupt only isolated sources.
Test an unset source separately from an explicitly empty or whitespace override;
only omission preserves the existing snapshot or bootstraps without configuration.
Explicit author-source edits must be visible without reinstalling. Capture the global installation
before and after; it must remain unchanged.

## Gotchas

A 0.6.4 binary that runs `fanloop update` switches to the new release, then prints `fanloop update: lstat …/fornax.json: no such file or directory` once and exits 1, because that sync still runs 0.6.4's own code. Run `fanloop update config` once more. `npm i -g <tgz>` syncs configuration during postinstall, and `fanloop flow init` re-executes the updated binary.

Source builds may warn about a missing Release manifest, and first installation warns before the first configuration fetch. Broken bindings and explicit sources fail. Installed versions use `VERSION-dev.<commit-or-content-digest>`. Configuration
packages stay outside the immutable Release. Installation never fetches. Fresh flow init fetches the Git configuration center even when `fornax.json` exists. `FANLOOP_USE_FORNAX=1` selects the installed Fornax snapshot. A versioned `current` is checked against the latest GitHub Release on a fresh non-dry-run init; failure keeps the installed binary.

A no-source-context verifier validates the distributed install command receipt,
binary digest, binding and public CLI readbacks. It must not claim it executed the
installer unless frozen installation resources were explicitly distributed.
