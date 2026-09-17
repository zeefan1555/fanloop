# Cross-release continuation

## Sub-features

- Continue a current-schema Requirement after its Workflow digest changes.
- Preserve the original Release version and Workflow digest in every durable fact.
- Reject missing Steps, incompatible Outputs, corrupt Events and new Results that violate current Routes.
- Keep Release and installation integrity checks digest-strict.

## How to get to it (user POV)

Create a Requirement with an older Fanloop release whose State Schema is still supported, advance it far enough to contain
a historical Result, then open the same Root with the current candidate. The creator and candidate must have the same
Workflow ID and different Bundle digests.

## Driving it with fanloop

Use an isolated creator install only to create the fixture. Use the current candidate for every tested command and assertion:
run `flow status`, a dry-run and real `flow report`, Card render, Trace status/render, and `doctor --root`. Compare State,
Output Registry, Events and Card Projection before and after to prove the old Release version and digest remain unchanged.
In separate Roots, verify missing current Step returns `WORKFLOW_MISMATCH`; incompatible Output/Event facts return
`STATE_CORRUPT`; historical Route drift is accepted; and a new Result that misses the current Route is rejected atomically.

## Gotchas

Do not hand-edit a Requirement to manufacture the positive live case. Build a creator from the same Schema with a harmless
Workflow content change so its digest differs naturally. Do not use the creator CLI for candidate assertions. Digest
tolerance applies only to Requirement runtime resolution and historical Route predicates; artifact Doctor, Manifest,
installation, packaging, current Step, Output ownership and new Result validation stay strict.
