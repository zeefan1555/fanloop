# Installation and health

## Sub-features

- Source build and isolated local installation, including a fresh offline bootstrap with an empty real `current/workflows`.
- Release version, commit, binary checksum and both global public Skill entrypoints (`fanloop-workflow` and `create-fanloop-workflow`).
- Stable `current/` root, nested Release selector and an independently updated published Workflow tree.
- One-time migration from a managed legacy `current -> releases/<version>` link without changing its Workflow bytes.
- Installation-level and Requirement-level Doctor checks.

## How to get to it (user POV)

Run `npm run build` (or `./scripts/build-local.sh`) to build without switching current, or
`npm run install:local` (or `./scripts/install-local.sh`) to install from the repository; npm scripts need Node.js/npm but no `npm install`. Inspect the
active installation with `fanloop version`, `fanloop doctor`, and `fanloop verify doctor`; add `--root` to diagnose an
initialized Requirement. Run `fanloop verify smoke` for the fixed isolated public-CLI journey and its evidence bundle.

## Driving it with fanloop

First install without `FANLOOP_CONFIG_SOURCE`; confirm `version` works and `current/workflows` is a real empty directory, then run explicit `fanloop update` before `doctor`. Install two clean commits with the same base `VERSION` into isolated `FANLOOP_DATA_HOME` and Skill roots, then run
`version` and `doctor`. Confirm each release version includes its commit prefix, both immutable directories remain,
`current/release` identifies the second commit, all four configured Skill roots link both public Skills to that current Release, and `current/workflows` remains byte-identical across both offline installs. Run explicit `fanloop update` separately to prove the selected published configuration can change without rebuilding the CLI. Corrupt only an isolated copy when proving Doctor rejection, then restore or discard that copy. Capture the global
current tree and selector before and after the run.

## Gotchas

Source builds can report a warning because they have no installed manifest. Local releases use
`VERSION-dev.<commit-or-content-digest>`; Production Workflow and Skill updates live in Fornax and do not change the CLI source revision. Local installation must not require Fornax. An installed candidate is not proven until Doctor reads the same isolated
data root and its `current/workflows` tree.
