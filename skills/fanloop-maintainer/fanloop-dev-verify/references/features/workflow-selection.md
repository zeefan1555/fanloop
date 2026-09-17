# Workflow selection

## Sub-features

- Explicit Workflow selection for a new Requirement.
- Scenario routing through `entrypoints/fanloop-workflow/routes.yaml`.
- Rejection of missing or unknown Workflow IDs.

## How to get to it (user POV)

Create an empty absolute Requirement directory, choose a documented scenario through the `fanloop-workflow` Skill, then
run `fanloop flow init --root ROOT --workflow ID --title TITLE`. Fanloop has no default Workflow.

## Driving it with fanloop

Read `flow init --help`. Prove missing Workflow selection and an unknown ID fail with structured errors. Run init dry-run
with a real Workflow and confirm no State exists; perform the real init and read Status to confirm the selected Workflow ID
and digest are bound.

## Gotchas

Selection only applies before initialization. An initialized Requirement keeps its persisted Workflow ID and digest as
provenance, while a supporting newer CLI resolves runtime semantics from its current Bundle with that ID. Changing live
Skill content does not change the persisted Workflow reference. Use a new Root for each selection variant.
