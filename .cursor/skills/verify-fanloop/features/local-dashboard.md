# Local dashboard

## Sub-features

- `dashboard` loopback launch, read-only bound-source viewer and explicitly selected local-source editor.
- Live Stage/Job/Step topology with forward, terminal, self, and back Route projection.
- Exact Step, Prompt, Route, Condition, and Skill source navigation into complete files, including common controls on every governed Step.
- Allowlisted Workflow YAML and `SKILL.md` list, read, and refresh.
- Startup and request-level locking during concurrent local source reads or saves.

## How to get to it (user POV)

Run `fanloop dashboard`, open the printed `http://127.0.0.1:<port>` URL, choose a Workflow, inspect its graph, then
select a Step and one of its six source tabs or open a file directly. The viewer loads complete real files. Default mode reads the bound repository and is read-only. Set an absolute FANLOOP_CONFIG_SOURCE to edit that repository, including the current binding; saves directly update its files.

## Driving it with fanloop

Launch `dashboard` against an isolated installation and verify `/api/topology` Stage/Step counts, Route kinds and
endpoints against the isolated live YAML. In a fresh browser session select a known Step, visit all six tabs, and prove
each fragment opens the complete allowlisted file with matching exact source lines selected. In default read-only mode,
verify POST `/api/file` is rejected and hashes do not change. For explicit edit mode, use a separate verifier-owned writable
source copy, never the frozen candidate or read-only verification inputs. Save one allowlisted complete file, including
the package root SKILL.md. Read its `sha256` with GET `/api/file?path=<URL-encoded-path>` and send the full replacement
with `Content-Type: application/json` to POST `/api/file`:

```json
{"path":"<path returned by GET /api/files>","content":"<complete replacement file>","observed_sha256":"<sha256 returned by GET /api/file>"}
```

Read the saved file back. Prove stale, traversal, symlink and invalid-Bundle writes fail without partial changes. To test
editing the current binding, first bind the isolated installation to that writable copy, then explicitly select the same
copy with `FANLOOP_CONFIG_SOURCE`; do not select the frozen inputs. Run two Dashboard instances and a CLI on the same
writable source with separate data homes to verify shared source locking. Confirm frozen input hashes remain unchanged.

For a Workflow with common controls, verify every Step exposes both control Prompts, Common Conditions, and Common Skills
without adding synthetic graph edges. During a clean external edit that inserts lines before the selected fragment,
verify polling reselects and scrolls to the fragment's new exact range. At a 1280 px desktop viewport, verify Fit keeps
the graph at a readable scale and nodes remain legible; open a file directly, click the already-selected Step, and verify
the Step inspector and default source fragment are restored.

## Gotchas

The server stays in the foreground and binds only to loopback. It cannot create, delete, rename or publish configuration. Local edits are whole-file replacements only. HTTP polling reads local files without any remote configuration service.

A managed snapshot with changed bytes or missing configuration.json must fail before serving configuration, both at startup and after the server is running, including explicit selection of that same snapshot. Cover all startup/running × changed/missing × default/explicit combinations against files, file and topology APIs. Deleting metadata must never turn managed storage into an editable author source. Test this separately from write permission.
