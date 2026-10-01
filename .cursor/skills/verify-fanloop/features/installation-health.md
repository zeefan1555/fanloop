# Installation and Health

## Sub-features

- Offline source build and installation bound to one complete local repository.
- Release version, commit, binary checksum and both public Skill entrypoints.
- Stable `current/` and nested Release selector, separate from `config/current` source binding.
- Failed installation preserves the active Release and previous source binding.
- Installation-level and Requirement-level Doctor checks.

## How to get to it (user POV)

Run `npm run build` or `./scripts/build-local.sh` to build without activation.
Run `npm run install:local` or `./scripts/install-local.sh` to install and bind that
repository. Inspect `fanloop version`, `fanloop doctor` and `fanloop verify doctor`.

## Driving it with fanloop

Use isolated data and Skill roots. Install two clean commits, check each immutable
Release, then check `current/release` and the absolute `config/current` binding.
Prove no new `current/workflows` copy exists. The four Skill roots expose only
`fanloop-workflow` and `create-fanloop-workflow`. Run from an unrelated cwd with the
override unset to prove the persisted source is used.

An invalid or missing explicitly supplied source must reject installation without
changing current or the previous binding. Test user-owned path conflicts and checksum
failures. Remove or corrupt only an isolated bound source and prove Doctor fails.
Test an unset source separately from an explicitly empty or whitespace override;
only the unset variant may select the default repository or persisted binding.
Local source edits must be visible without reinstalling. Capture the global installation
before and after; it must remain unchanged.

## Gotchas

Source builds may warn about a missing Release manifest, but missing Workflow source
must fail. Installed versions use `VERSION-dev.<commit-or-content-digest>`. Configuration
packages stay outside the immutable Release. No remote configuration service is used.

A no-source-context verifier validates the distributed install command receipt,
binary digest, binding and public CLI readbacks. It must not claim it executed the
installer unless frozen installation resources were explicitly distributed.
