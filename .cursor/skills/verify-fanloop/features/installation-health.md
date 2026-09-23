# Installation and health

## Sub-features

- Source build and isolated local installation.
- Release version, commit, binary checksum and global entrypoint integrity.
- Stable `current/` root, nested Release selector and synchronized published Workflow tree.
- Installation-level and Requirement-level Doctor checks.

## How to get to it (user POV)

Run `npm run build` (or `./scripts/build-local.sh`) to build without switching current, or
`npm run install:local` (or `./scripts/install-local.sh`) to install from the repository; npm scripts need Node.js/npm but no `npm install`. Inspect the
active installation with `fanloop version`, `fanloop doctor`, and `fanloop verify doctor`; add `--root` to diagnose an
initialized Requirement. Run `fanloop verify smoke` for the fixed isolated public-CLI journey and its evidence bundle.

## Driving it with fanloop

Install two clean commits with the same base `VERSION` into isolated `FANLOOP_DATA_HOME` and Skill roots, then run
`version` and `doctor`. Confirm each release version includes its commit prefix, both immutable directories remain,
`current/release` identifies the second commit, and `current/workflows` remains a real directory matching the explicitly selected published configuration. Corrupt only an isolated copy when proving Doctor rejection, then restore or discard that copy. Capture the global
current tree and selector before and after the run.

## Gotchas

Source builds can report a warning because they have no installed manifest. Local releases use
`VERSION-dev.<commit-or-content-digest>`; Production Workflow and Skill updates live in Fornax and do not change the CLI source revision. An installed candidate is not proven until Doctor reads the same isolated
data root and its `current/workflows` tree.
