# Card and panorama

## Sub-features

- Current and panorama views.
- Markdown and Lark JSON formats.
- Immutable snapshots and renderer-owned content.

## How to get to it (user POV)

Run `fanloop card render --root ROOT --view current|panorama --format markdown|lark-json`. Use `--dry-run` when the
caller only needs presentation content and no local snapshot.

## Driving it with fanloop

Render immediately after init and after a transition. Capture response content, projection and snapshot count. Verify the
panorama identifies the current Stage/Job/Step and that a real render creates a new immutable snapshot while dry-run does
not. Use returned content directly as the first user-visible message after entering a Step; any human question must follow
below it, and the final reply must not repeat the panorama.

## Gotchas

Rendering is separate from Botmux delivery. A valid Lark payload is not proof that it was sent. Do not rebuild or summarize
renderer-owned content when exact presentation is part of the contract.
