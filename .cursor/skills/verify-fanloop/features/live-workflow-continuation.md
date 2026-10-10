# Live Workflow continuation

## Sub-features

- Continue a current-schema Requirement after valid live Workflow or Skill edits.
- Preserve the creating CLI Release version and Workflow ID in every durable fact.
- Reject missing current Steps, incompatible Outputs, corrupt Events and new Results that violate current Routes.
- Reject old Storage schemas without migration or fallback.
- Pin a controller with Python 3 standard-library shared source locking; failed new pins remove only their own controller home.
- Reuse existing controllers with their own configuration binding after clearing external source overrides.

## How to get to it (user POV)

Create and advance a Requirement, then edit the bound local Workflow package before the next CLI invocation. The
Requirement keeps its Workflow ID while Status, Prompt, Skill paths and new Route validation use the latest live files.

## Driving it with fanloop

Use one isolated install and a fresh Requirement Root. Copy the distributed packages or synthetic fixtures into a separate
verifier-owned writable repository with complete packages under `fanloop-workflow/`, then set the absolute `FANLOOP_CONFIG_SOURCE`
to that copy. Keep the frozen candidate and read-only verification inputs unchanged throughout the run.
Run `flow init`, advance within the approved synthetic scope to create historical Events, and edit a Prompt and Skill only
in the writable copy. Use the same candidate binary for `flow status`, a dry-run and real `flow report`, Card render,
Trace status/render and `doctor --root`; the next invocation must read the edits without publication or reinstallation.
In separate Roots, prove a removed current Step or incompatible
Output/Event facts fail closed, historical accepted Results are not re-decided, and a new Result that misses the latest
Route is rejected atomically. Present an old-schema Requirement and verify `STATE_SCHEMA_UNSUPPORTED` with no write.

## Gotchas

Do not hand-edit State or Events to manufacture the positive live-edit case. Workflow YAML and bound Skills must come
from the same live root on every invocation. There is no Workflow digest, packaged Workflow fallback or migration path;
Release integrity covers the CLI and global entrypoint only.
