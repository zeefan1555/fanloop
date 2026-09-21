# Requirement lifecycle

## Sub-features

- Uninitialized, running and completed Requirement states.
- Init, latest Status and current Prompt/Skill projection.
- Durable Event history and final terminal state.

## How to get to it (user POV)

Create an empty absolute Root, then run `flow status`. A missing directory returns `INVALID_ARGUMENT`; an existing empty
Root returns `NOT_INITIALIZED`. Initialize once, then repeat the control loop:
`status -> execute current Prompt/Skills -> report progress/result -> status` until terminal.

## Driving it with fanloop

Capture the initial error, init dry-run, real init and first Status. Treat init dry-run as preserving business state, not as
writing no files: `.fanloop/log/cli.jsonl` may record the invocation. At every Step store the current Step ID, Conditions,
available Routes, Prompt and absolute Skill paths. Progress through a small approved journey, then use `trace status` and
`trace render` and compare them with `.fanloop/trace/events.jsonl` to prove the completed Status, State and Event tail agree.

## Gotchas

Status is a fresh control-plane read, not a cache. An older CLI may create a real cross-release fixture, but every tested
drive and assertion must use the current candidate. Historical Requirement roots or a different CLI release cannot prove the
candidate. Human Steps in other Workflows require a real human decision. In `fanloop-maintainer`, the execution Sub-agent
owns the control loop and requests the Main Agent's requirement decision in `define_verification_contract` and final
candidate decision in `certify_candidate`.
