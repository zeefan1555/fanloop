# Card and panorama

## Sub-features

- Current and panorama views.
- Markdown and Lark JSON formats.
- Immutable snapshots and renderer-owned content.

## How to get to it (user POV)

Run `fanloop card render --root ROOT --view current|panorama --format markdown|lark-json`. Use `--dry-run` when the
caller only needs presentation content and no local snapshot.

## Driving it with fanloop

Render immediately after init and after a transition while `pending_actions` contains `panorama_delivery_required`.
Capture response content, projection and snapshot count. Verify the panorama identifies the current Stage/Job/Step and
that a real render creates a new immutable snapshot while dry-run does not. Use returned content directly as the first
user-visible message, then `fanloop flow action acknowledge` that same action id and snapshot path before any business
progress or result. Any human question must follow below the panorama. The in-step retrospect panorama is not the Done
100% delivery after a successful terminal route.

## Gotchas

Rendering is separate from Botmux delivery. A valid Lark payload is not proof that it was sent. Do not rebuild or summarize
renderer-owned content when exact presentation is part of the contract. A stale projection is rebuilt from committed
events under the requirement's pinned snapshot by a non-dry-run `card render`; dry-run leaves the projection file unchanged.
Missing or corrupt projections still fail.
