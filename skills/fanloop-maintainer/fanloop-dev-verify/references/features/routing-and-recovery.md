# Routing and recovery

## Sub-features

- Flow advance, terminal completion, self-loop and back-edge recovery.
- OR-of-AND Condition alternatives and explicit Route selection.
- Downstream Output invalidation after recovery.

## How to get to it (user POV)

Read `data.state.current.available_routes` from Status. Submit one complete `when.any_of` alternative and the exact
`next_step_id`, `back_step_id` or terminal selection returned for that alternative.

## Driving it with fanloop

Use a fresh Root per route variant. Prove one forward transition, one permitted recovery and one rejected route. For a
back-edge, capture Outputs before and after and verify facts produced at or after the target Step are invalidated while
earlier facts remain. Route Matrix is supporting coverage, not a substitute for the public CLI proof.

## Gotchas

Do not infer routes from Step order. Different targets can share Condition facts and require explicit selection. A jump or
back-edge changes the current position but must not manufacture completion facts for skipped Steps.
