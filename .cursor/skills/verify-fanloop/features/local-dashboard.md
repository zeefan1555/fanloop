# Local dashboard

## Sub-features

- `dashboard` loopback launch and embedded read-only viewer.
- Live Stage/Job/Step topology with forward, terminal, self, and back Route projection.
- Exact Step, Prompt, Route, Condition, and Skill source navigation into complete files, including common controls on every governed Step.
- Allowlisted Workflow YAML and `SKILL.md` list, read, and refresh.
- Startup synchronization and request-level read locking during concurrent configuration replacement.

## How to get to it (user POV)

Run `fanloop dashboard`, open the printed `http://127.0.0.1:<port>` URL, choose a Workflow, inspect its graph, then
select a Step and one of its six source tabs or open a file directly. The viewer loads complete real files. The page
polls synchronized topology/files; edits are published through Fornax.

## Driving it with fanloop

Launch `dashboard` against an isolated installation and verify `/api/topology` Stage/Step counts, Route kinds and
endpoints against the isolated live YAML. In a fresh browser session select a known Step, visit all six tabs, and prove
each fragment opens the complete allowlisted file with matching exact source lines selected. Verify there is no editable
save path; POST /api/file returns 405 and file hashes do not change. Check traversal, absolute path and symlink reads.
Run a parallel CLI refresh and prove requests never observe a partially installed tree.

For a Workflow with common controls, verify every Step exposes both control Prompts, Common Conditions, and Common Skills
without adding synthetic graph edges. During a clean external edit that inserts lines before the selected fragment,
verify polling reselects and scrolls to the fragment's new exact range. At a 1280 px desktop viewport, verify Fit keeps
the graph at a readable scale and nodes remain legible; open a file directly, click the already-selected Step, and verify
the Step inspector and default source fragment are restored.

## Gotchas

The server stays in the foreground and binds only to loopback. It cannot create, edit, delete, rename or publish
configuration. It releases the startup synchronization lock before serving and locks only each API read. HTTP polling
reads already synchronized local files, not Fornax on each request.
