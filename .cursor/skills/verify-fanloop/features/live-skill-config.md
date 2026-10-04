# Local Workflow Skill Packages

## Sub-features

- One complete package at `skills/<workflow-id>/`: root SKILL.md, five YAML files and package-local `skills/**` resources.
- Absolute source binding at `$FANLOOP_DATA_HOME/config/current`; explicit `FANLOOP_CONFIG_SOURCE` overrides it.
- No source search, copied live tree, remote update or fallback.
- Strict full-package validation and shared source-directory locking across data homes.

## How to get to it (user POV)

Install from the repository with `./scripts/install-local.sh`. Run `fanloop doctor` and
`flow init/status` from an unrelated working directory. Their paths must identify
the bound source's packages, not the binary or the invocation directory.

## Driving it with fanloop

Bind source A, invoke from another cwd, then explicitly select source B and verify its
Prompt and Skill paths. Edit a local prompt and resource without rebuilding; the next
invocation must read it while the binary digest remains unchanged. Unset the override
and confirm A is used. A bad explicit source or missing bound source must fail without
reading another checkout or old `current/workflows` bytes.

Reject explicitly empty/whitespace and relative sources, symlinked roots/packages/resources, missing SKILL.md, missing
YAML, unknown root files, invalid YAML and missing bound Skills without business State
writes. Preserve nested resources and executable scripts. Only the two public entry
Skills may appear in global Skill roots. `fanloop update` must be an unknown operation.
Run with no remote helper or network credentials.

Use `scripts/validate-configuration.py` for the fixed source's Route Matrix and complete
configuration manifest; it does not execute business Skill actions. Repeat `--workflow ID`
to run only affected matrices; no selection means all, duplicate IDs run once and an
unknown ID fails before any matrix. Verify the manifest still lists all source files
and reports unchanged source. Prompt prose alone needs strict loading and a relevant
example, not matrices or manually rebuilt parser-negative suites.

## Gotchas

Workflow content is read from the repository on every invocation, independently of the
immutable CLI Release. Existing Requirements retain their Workflow ID. A pinned
self-iteration controller uses its own complete source snapshot.
