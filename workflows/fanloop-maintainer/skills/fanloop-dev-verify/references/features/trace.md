# Trace

## Sub-features

- Trace document binding and local binding status.
- Local Event Markdown rendering.
- Remote Trace and Registry synchronization.

## How to get to it (user POV)

Use `trace bind`, `trace status`, `trace render` and `trace sync` on an initialized Requirement. Binding and remote sync are
explicit operations; Flow transitions do not silently perform them.

## Driving it with fanloop

Always cover local `trace status` and `trace render`: dry-run first, then render and compare the Markdown projection with
durable Events. Bind or sync only when the verification scope explicitly authorizes the exact document, identity and remote
write. After sync, read Status and every returned target receipt.

## Gotchas

External auth or network unavailability is `verified-unreachable` only after recording the attempted public path and concrete
missing prerequisite. A local fake can test the production transport seam but cannot prove a real remote write.
