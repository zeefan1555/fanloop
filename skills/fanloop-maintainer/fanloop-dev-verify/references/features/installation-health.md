# Installation and health

## Sub-features

- Source build and isolated local installation.
- Release version, commit, binary checksum, Workflow bundle and entrypoint integrity.
- Installation-level and Requirement-level Doctor checks.

## How to get to it (user POV)

Run `./scripts/build-local.sh` to build without switching current, or `./scripts/install-local.sh` to install. Inspect the
active installation with `fanloop version` and `fanloop doctor`; add `--root` to diagnose an initialized Requirement.

## Driving it with fanloop

Install with isolated `FANLOOP_DATA_HOME` and Skill roots, then run `version` and `doctor`. Confirm release version,
commit and Workflow digests identify the candidate. Corrupt only an isolated copy when proving Doctor rejection, then
restore or discard that copy. Capture the global current target before and after the run.

## Gotchas

Source builds can report a warning because they have no installed manifest. Skill and exemplar changes are live config and
do not change the CLI version. An installed candidate is not proven until Doctor reads the same isolated data/config roots.
