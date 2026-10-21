# Workflow selection

## Sub-features

- Explicit Workflow selection for a new Requirement.
- Scenario routing through `entrypoints/fanloop-workflow/routes.yaml`.
- Rejection of missing or unknown Workflow IDs.
- The distributed fanloop-workflow entrypoint groups new default Roots under ~/fanloop/workspace/<workflow-id>/<requirement-root>; the CLI still receives an explicit absolute Root. Existing Roots continue at their original path.

## How to get to it (user POV)

Create an empty absolute Requirement directory, choose a documented scenario through the `fanloop-workflow` Skill, then
run `fanloop flow init --root ROOT --workflow ID --title TITLE`. Fanloop has no default Workflow and loads the selection
from the bound local repository's `fanloop-workflow/` directory.

## Driving it with fanloop

Read the frozen installed entrypoint and record its digest. Resolve its grouped default to an absolute isolated Root and run status, init dry-run, real init and status with a synthetic Workflow. Continue a separate previously initialized Root and prove neither directory moved. Do not infer an implicit CLI Root argument from the Skill convention.

Read `flow init --help`. Prove missing Workflow selection fails with `INVALID_ARGUMENT` at exit 2 and an unknown ID fails with
`WORKFLOW_NOT_FOUND` at exit 1 without exposing filesystem errors. Run init dry-run
with a real Workflow and confirm no State exists; perform the real init and read Status to confirm the selected Workflow ID
is bound and its Prompt/Skill paths come from the Requirement's pinned snapshot.

For an authoring failure matrix, copy the synthetic Workflow tree into a fresh isolated configuration root and mutate one
case at a time: malformed YAML syntax, an unknown field, an invalid Step executor, an unknown Condition, an unknown Prompt,
and an invalid Route target. Each `flow init` must return `WORKFLOW_INVALID`, exit `5`, identify the invalid file or
relationship, and leave `.fanloop/flow/state.json` absent. This automated load check replaces any YAML-specific human
approval; it does not replace product behavior trials for valid configuration.

## Gotchas

Selection only applies before initialization. An initialized Requirement keeps its persisted Workflow ID and configuration
snapshot; later config/current or author-source edits do not silently replace its semantics or Skill paths. Use the explicit
rebind command when adopting installed configuration, or a new Root for each selection variant.
