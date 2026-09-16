# Progress and result

## Sub-features

- `flow report progress` execution facts.
- `flow report result` typed Condition results, Evidence and Route selection.
- JSON input, typed flags, dry-run and structured rejection.

## How to get to it (user POV)

Read the latest Status and the target report leaf `--help`. Use only the current Step ID, listed Conditions and one listed
Route. Prefer `--input @file` or stdin for complex requests.

## Driving it with fanloop

Build the request only from Status. Snapshot durable files, run the report with `--dry-run`, and prove no State/Event change.
Run the real request, capture its effect and event ID, then read Status and durable files. Also submit one invalid current
fact and prove its stable error code and lack of durable mutation.

## Gotchas

Evidence does not participate in Route matching. Output types and constraints come from Status, not from memory. CLI
transcripts contain complete unredacted input and output, so verification uses synthetic values only.
