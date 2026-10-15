# Installation and Health

## Sub-features

- Offline source build and installation; optional clean Git configuration import.
- GitHub Release install: platform archive, checksums.txt, release.json, the thin npm launcher, public Skill links, and `npm i -g --foreground-scripts --allow-scripts=fanloop`.
- `fanloop update` keeps a newer installed release instead of downgrading to `latest`. `fanloop doctor` warns with `export PATH=...` when that distribution install is not on PATH.
- `fanloop update` installs the CLI and then synchronizes released Fornax Workflow packages. `fanloop update config` synchronizes those packages, skips an unchanged released set, and prints each `id@version`.
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

Source builds may warn about a missing Release manifest, and first installation warns before the first configuration fetch. Broken bindings and explicit sources fail. Installed versions use `VERSION-dev.<commit-or-content-digest>`. Configuration
packages stay outside the immutable Release. Installation never fetches. With `fornax.json` and no `FANLOOP_CONFIG_SOURCE`, flow init reads the Fornax snapshot; otherwise it uses the Git configuration center.

A no-source-context verifier validates the distributed install command receipt,
binary digest, binding and public CLI readbacks. It must not claim it executed the
installer unless frozen installation resources were explicitly distributed.
