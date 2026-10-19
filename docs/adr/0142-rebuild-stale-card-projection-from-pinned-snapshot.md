---
status: accepted
date: 2026-10-18
amends: ADR-0039-isolate-card-and-output-storage
superseded_in_part_by: ADR-0143-everything-is-condition
---

# Rebuild a stale Card projection from the pinned snapshot

An existing requirement keeps the Workflow directory recorded in `.fanloop/configuration-snapshot`. Later reads of that requirement use the pin after `config/current` moves. This decision applies that rule to every requirement-scoped Workflow load that was still following the global root: Card projection writes, Card projection loads, `card render`, and Doctor's requirement checks. `flow status`, `flow report`, `flow skills`, and `trace` already load through the pin. Doctor's installation check still reads the active configuration root.

`card render` still fails when `.fanloop/card/projection.json` is missing, corrupt, or bound to a Workflow that does not declare the stored Outputs. That part of ADR-0039 stays. A projection that decodes and validates against the pinned Workflow, but does not match committed State, is stale. State and the event log are the source of that projection. Non-dry-run `card render` rebuilds the projection from those committed facts under the pinned Workflow, then renders and writes the snapshot. Dry-run renders the same content and does not change the projection file or write a snapshot. The rebuild does not append a Trace event.

Doctor reports a stale projection as failed. Its hint is to run a non-dry-run `fanloop card render --root <requirement>`.
