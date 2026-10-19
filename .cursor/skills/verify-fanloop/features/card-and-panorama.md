# Card and panorama

## Sub-features

- Current and panorama views.
- Markdown and Lark JSON formats.
- Immutable snapshots and renderer-owned content.

## How to get to it (user POV)

Run `fanloop card render --root ROOT --view current|panorama --format markdown|lark-json`. Use `--dry-run` when the
caller only needs presentation content and no local snapshot.

## Driving it with fanloop

For a Workflow that declares a Panorama Condition, render when that Condition's bound Skill requires it. Capture response
content, projection and snapshot count. Verify the panorama identifies the current Stage/Job/Step and that a real render
creates a new immutable snapshot while dry-run does not. Use returned content directly as the first user-visible message
for that configured gate; any human question must follow below it.

## Gotchas

Rendering is separate from Botmux delivery. A valid Lark payload is not proof that it was sent. Do not rebuild or summarize
renderer-owned content when exact presentation is part of the contract. A stale projection is rebuilt from committed
events under the requirement's pinned snapshot by a non-dry-run `card render`; dry-run leaves the projection file unchanged.
Missing or corrupt projections still fail.
