# Installation and health

## Sub-features

- Source build and isolated local installation.
- Release version, commit, binary checksum, Workflow bundle and entrypoint integrity.
- Installation-level and Requirement-level Doctor checks.

## How to get to it (user POV)

Run `./scripts/build-local.sh` to build without switching current, or `./scripts/install-local.sh` to install. Inspect the
active installation with `fanloop version`, `fanloop doctor`, and `fanloop verify doctor`; add `--root` to diagnose an
initialized Requirement. Run `fanloop verify smoke` for the fixed isolated public-CLI journey and its evidence bundle.

## Driving it with fanloop

Install two clean commits with the same base `VERSION` into isolated `FANLOOP_DATA_HOME` and Skill roots, then run
`version` and `doctor`. Confirm each release version includes its commit prefix, both immutable directories remain, and
current identifies the second commit. Corrupt only an isolated copy when proving Doctor rejection, then
restore or discard that copy. Capture the global current target before and after the run.

## Gotchas

Source builds can report a warning because they have no installed manifest. Local releases use
`VERSION-dev.<commit-or-content-digest>`; Skill and exemplar changes are live config and do not change a dirty build's
content-derived version. An installed candidate is not proven until Doctor reads the same isolated data/config roots.
