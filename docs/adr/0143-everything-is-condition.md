---
status: accepted
date: 2026-10-20
supersedes: ADR-0137-move-panorama-delivery-to-runtime-action
supersedes_in_part: ADR-0119-require-panorama-condition-in-every-workflow, ADR-0130-deliver-done-panorama-after-terminal-route, ADR-0142-rebuild-stale-card-projection-from-pinned-snapshot
---

# Everything is condition

Fanloop Runtime no longer owns Panorama delivery or any other business gate. It initializes the first configured Step, validates the Conditions and Routes declared by the selected Workflow, persists accepted reports, and projects state. It does not inject fixed nodes, `pending_actions`, delivery acknowledgments, or reserved Condition IDs.

`panorama_presented` and `panorama_card_published` are ordinary Condition IDs. A Workflow that needs a Panorama declares its Condition, adds it to the applicable Route combinations, and binds a delivery Skill through `prompt.yaml`. A Workflow that omits such a Condition starts and advances without rendering or acknowledging a Panorama. `card render --view panorama` remains a general renderer that any Workflow Skill may call.

This removes the Runtime-owned `panorama_delivery_required` action, `flow action acknowledge`, the release-level `fanloop-panorama-delivery` Skill, snapshot receipt identity, and the `CONDITION_RESERVED` error. The existing example package demonstrates the opt-in model with `panorama_presented` and `example-panorama`. Production Workflow packages live in second-brain-skills and migrate separately by adding the same Condition/Route/Skill pattern only where Panorama remains desired.

No compatibility layer or fallback path remains. State and Event schema versions advance from 15 to 16. Event kind value 7, EventPayload field 7, and FlowState field 5 are retired and must not be reused. Requirements written with schema 15 are not migrated by this change.

## Contract review

The user explicitly authorized replacing the fixed Panorama state-machine gate with Workflow-owned Conditions. The main Agent reviewed and accepts these Thrift changes before implementation:

- `idl/flow.thrift`: retire `FlowState` field 5; remove `ActionType`, `PendingAction`, `ActionAcknowledgeEffect`, `FlowActionAcknowledgeRequest`, `FlowActionAcknowledgeResponse`, and `FlowService.AcknowledgeAction`; remove `CONDITION_RESERVED` from command error catalogs.
- `idl/storage.thrift`: advance `FLOW_STATE_SCHEMA_VERSION` and `EVENT_SCHEMA_VERSION` to 16; retire `EventKind` value 7 and `EventPayload` field 7; remove `ActionType` and `FlowActionAcknowledgedPayload`.
- `idl/error.thrift`: retire error code value 32 and remove its ErrorSpec.

Generated Go contracts are updated only through `scripts/generate-cli-idl.sh`.
