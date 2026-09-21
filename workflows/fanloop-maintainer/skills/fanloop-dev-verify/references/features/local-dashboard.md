# Local dashboard

## Sub-features

- `dashboard` loopback launch and embedded editor UI.
- Live Stage/Job/Step topology with forward, terminal, self, and back Route projection.
- Exact Step, Prompt, Route, Condition, and Skill source navigation into complete files, including common controls on every governed Step.
- Allowlisted Workflow YAML and `SKILL.md` list, read, refresh, and save.
- SHA-256 conflict detection, bundle validation, and atomic replacement.

## How to get to it (user POV)

Run `fanloop dashboard`, open the printed `http://127.0.0.1:<port>` URL, choose a Workflow, inspect its graph, then
select a Step and one of its six source tabs or open a file directly. The editor always loads and saves the complete
real file. The page polls live topology and files and shows when an external change makes a dirty editor stale.

## Driving it with fanloop

Launch `dashboard` against an isolated installation and verify `/api/topology` Stage/Step counts, Route kinds and
endpoints against the isolated live YAML. In a fresh browser session select a known Step, visit all six tabs, and prove
each fragment opens the complete allowlisted file with the matching exact source lines selected. Save a valid Step-name
edit with the observed digest and verify both exact disk bytes and refreshed graph text. Then exercise invalid-bundle
rejection, an external edit while the editor is dirty, HTTP 409 stale protection, explicit refresh, traversal,
absolute-path, symlink, and empty-Skill rejection.

For a Workflow with common controls, verify every Step exposes both control Prompts, Common Conditions, and Common Skills
without adding synthetic graph edges. During a clean external edit that inserts lines before the selected fragment,
verify polling reselects and scrolls to the fragment's new exact range. At a 1280 px desktop viewport, verify Fit keeps
the graph at a readable scale and nodes remain legible; open a file directly, click the already-selected Step, and verify
the Step inspector and default source fragment are restored.

## Gotchas

The server stays in the foreground and binds only to loopback. It edits existing allowlisted files only; it does not
create, delete, rename, authenticate, expose remote access, mutate the graph, save partial YAML, or overwrite a dirty
editor after an external change. `/api/topology` is a read-only projection; `POST /api/file` is the sole write path.
