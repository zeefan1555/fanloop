# Requirement lifecycle

## Sub-features

- Uninitialized, running and completed Requirement states.
- Init, latest Status and current Prompt/Skill projection.
- Durable Event history and final terminal state.

## How to get to it (user POV)

Run `flow status` on a new absolute Root. After `NOT_INITIALIZED`, initialize once, then repeat the control loop:
`status -> execute current Prompt/Skills -> report progress/result -> status` until terminal.

## Driving it with fanloop

Capture the initial error, init dry-run, real init and first Status. At every Step store the current Step ID, Conditions,
available Routes, Prompt and absolute Skill paths. Progress through a small approved journey, then verify the completed
Status, State and Event tail agree.

## Gotchas

Status is a fresh control-plane read, not a cache. Historical Requirement roots or a different CLI release cannot prove the
current candidate. Human Steps require a real human decision; a verification run stops before them unless explicitly scoped.
