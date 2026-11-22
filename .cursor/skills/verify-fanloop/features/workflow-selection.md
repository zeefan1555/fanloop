# Workflow selection

## Sub-features

- The production scenario map is empty; daily product flows live in `second-brain-agent/plugins/`.
- Rejection of missing or unknown Workflow IDs.
- The distributed fanloop-workflow entrypoint groups new default Roots under ~/fanloop/workspace/<workflow-id>/<requirement-root>; the CLI still receives an explicit absolute Root. Existing Roots continue at their original path.

## How to get to it (user POV)

New product work does not create a Requirement. Continue an existing Root with its pinned controller. Synthetic fixtures still exercise explicit Workflow validation in isolation.

## Driving it with fanloop

Read the frozen installed entrypoint and record its digest. Prove the source `routes.yaml` has zero scenarios and release validation accepts that empty map. Use a synthetic explicit author source for the existing init/status regression; never present it as a production scenario. Continue a separate previously initialized Root and prove its directory and pinned source did not move.

Read `flow init --help`. Prove missing Workflow selection fails with `INVALID_ARGUMENT` at exit 2 and an unknown ID fails with
`WORKFLOW_NOT_FOUND` at exit 1 without exposing filesystem errors. Run init dry-run
with a real Workflow and confirm no State exists; perform the real init and read Status to confirm the selected Workflow ID
is bound and its Prompt/Skill paths come from the Requirement's pinned configuration source.

For an authoring failure matrix, copy the synthetic Workflow tree into a fresh isolated configuration root and mutate one
case at a time: malformed YAML syntax, an unknown field, an invalid Step executor, an unknown Condition, an unknown Prompt,
and an invalid Route target. Each `flow init` must return `WORKFLOW_INVALID`, exit `5`, identify the invalid file or
relationship, and leave `.fanloop/flow/state.json` absent. This automated load check replaces any YAML-specific human
approval; it does not replace product behavior trials for valid configuration.

## Gotchas

Selection only applies before initialization. An initialized Requirement keeps its persisted Workflow ID and absolute
configuration pin; later config/current updates do not replace that binding. A Requirement pinned to an explicit author
repository still reflects content edits in that same repository. Use config rebind to adopt an installed snapshot, or a new
Root for each selection variant.
