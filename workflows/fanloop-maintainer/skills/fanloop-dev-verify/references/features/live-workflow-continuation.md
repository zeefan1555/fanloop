# Live Workflow continuation

## Sub-features

- Continue a current-schema Requirement after valid live Workflow or Skill edits.
- Preserve the creating CLI Release version and Workflow ID in every durable fact.
- Reject missing current Steps, incompatible Outputs, corrupt Events and new Results that violate current Routes.
- Reject old Storage schemas without migration or fallback.

## How to get to it (user POV)

Create and advance a Requirement, then edit its Workflow under `current/workflows` before the next CLI invocation. The
Requirement keeps its Workflow ID while Status, Prompt, Skill paths and new Route validation use the latest live files.

## Driving it with fanloop

Use one isolated install and a fresh Requirement Root. Run `flow init`, advance far enough to create historical Events,
edit the live Prompt and Skill, then use the same candidate for `flow status`, a dry-run and real `flow report`, Card
render, Trace status/render and `doctor --root`. In separate Roots, prove a removed current Step or incompatible
Output/Event facts fail closed, historical accepted Results are not re-decided, and a new Result that misses the latest
Route is rejected atomically. Present an old-schema Requirement and verify `STATE_SCHEMA_UNSUPPORTED` with no write.

## Gotchas

Do not hand-edit State or Events to manufacture the positive live-edit case. Workflow YAML and bound Skills must come
from the same live root on every invocation. There is no Workflow digest, packaged Workflow fallback or migration path;
Release integrity covers the CLI and global entrypoint only.
