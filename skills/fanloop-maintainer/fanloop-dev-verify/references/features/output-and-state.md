# Output and state

## Sub-features

- Flow State, Output Registry and producer ownership.
- Atomic State/Output/Event commits.
- Current-fact retention and downstream invalidation.

## How to get to it (user POV)

Outputs appear through Status after accepted reports. Local facts live under `.fanloop/flow`, `.fanloop/output` and
`.fanloop/trace`; users inspect them through Status and Doctor rather than editing them.

## Driving it with fanloop

After init and each accepted report, capture Status plus `flow/state.json`, `output/state.json` and `events.jsonl`. Confirm
Workflow bindings agree, each Output names its producer Step, Result Event changes match the registries, and rejected or
dry-run requests leave durable bytes unchanged.

## Gotchas

State is the runtime truth, but Outputs are stored in their own registry. Never add missing values by hand. Path Outputs are
Requirement-root-relative and must resolve inside the Root.
