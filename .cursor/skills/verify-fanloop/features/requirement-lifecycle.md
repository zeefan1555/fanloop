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

Status is a fresh control-plane read, not a cache. An older CLI may create a real cross-release fixture. Every tested drive
must enter through the current candidate; for a bound Root, verify its automatic dispatch to the exact known older
controller before State decoding. Executing only that older binary directly does not prove the candidate entrypoint.
Record both candidate and selected controller identities, actual configuration pin and before/after business facts. M1 keeps
State/Event 16; version relabeling or same-Schema continuation does not prove later incompatible-Schema continuity.
Human Steps in other Workflows require a real human decision. In the `code-change` Workflow, the execution Sub-agent
owns the control loop and requests the Main Agent's requirement decision and final candidate decision.
