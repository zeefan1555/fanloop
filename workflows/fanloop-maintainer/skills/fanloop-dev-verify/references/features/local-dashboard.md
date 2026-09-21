# Local dashboard

## Sub-features

- `dashboard` loopback launch and embedded editor UI.
- Allowlisted Workflow YAML and `SKILL.md` list, read, refresh, and save.
- SHA-256 conflict detection, bundle validation, and atomic replacement.

## How to get to it (user POV)

Run `fanloop dashboard`, open the printed `http://127.0.0.1:<port>` URL, choose a Workflow file, edit it, and save.
The page polls live files and shows when an external change makes a dirty editor stale.

## Driving it with fanloop

Launch `dashboard` against an isolated installation and use its loopback HTTP API to list and read one YAML plus one
Skill. Save a valid edit with the observed digest, modify the file externally, verify the next read changes, then prove
an old digest returns HTTP 409. Exercise traversal, absolute-path, symlink, invalid-YAML, and empty-Skill rejection.

## Gotchas

The server stays in the foreground and binds only to loopback. It edits existing allowlisted files only; it does not
create, delete, rename, authenticate, expose remote access, or overwrite a dirty editor after an external change.
