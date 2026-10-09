# fanloop-remnote

Cursor Agent Plugin for two RemNote jobs that used to run inside Fanloop:

- **remnote-flashcards** — draft, review, write, and retrospect learning cards.
- **remnote-direct-maintainer** — change the separate RemNote Direct CLI/plugin repository.

The skills call the installed `remnote-cli` (local WebSocket to the RemNote plugin). They do not call `fanloop flow` and they do not ship an MCP server.

`remnote-flashcards` follows four steps: locate the HOME Concept tag, draft and review, preview then apply, then a short retrospect. `remnote-direct-maintainer` follows define, build, verify, and deliver. Pass/fail checks from the Fanloop workflow files are checklists in those two skills.

## Prerequisites

- Node.js `>= 22.13`.
- `remnote-cli` installed and paired. The pairing token stays in `~/.config/remnote-cli/connection.json`. Do not copy that file into the plugin or the git repo.
- RemNote running, with the remnote-direct SDK Bridge plugin loaded. From the remnote-direct repo, serve the built plugin on localhost:

  ```bash
  node scripts/serve-dist.js --root dist --port 8090
  ```

  That server binds to `127.0.0.1`. In RemNote, load the SDK Bridge from that local URL. `remnote-cli status` should then show the knowledge base you intend to touch.
- On a host without FUSE, start the RemNote AppImage with extraction and the sandbox disabled:

  ```bash
  APPIMAGE_EXTRACT_AND_RUN=1 /path/to/RemNote.AppImage --no-sandbox
  ```

- Optional, only for the flashcard retrospect when there is one finding: Notion CLI `ntn`, already logged in (`ntn login`). Headless hosts set `NOTION_KEYRING=0` before login. An integration API key is not the path.
  - `FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID` overrides the data source id.
  - `FANLOOP_REMN_EXPERIENCE_DATABASE_ID` overrides the database id.
  - Defaults are in `skills/remnote-flashcards/references/notion-experience.md`.

## Install locally in Cursor

Cursor loads Agent Plugins from `~/.cursor/plugins/local/<plugin-name>/` when local plugin imports are allowed (Teams and Enterprise admins control this). Reload the window after installing (`Developer: Reload Window`), then open Customize and confirm the skills listed below.

Current Cursor docs skip a symlink whose target sits outside `~/.cursor/plugins/local`. Copy the plugin directory:

```bash
mkdir -p ~/.cursor/plugins/local
rm -rf ~/.cursor/plugins/local/fanloop-remnote
cp -a /path/to/fanloop/plugins/fanloop-remnote ~/.cursor/plugins/local/fanloop-remnote
```

A symlink is the right shape only when its target stays inside that folder. A symlink from `~/.cursor/plugins/local/fanloop-remnote` to a checkout elsewhere is skipped:

```bash
ln -s /path/to/fanloop/plugins/fanloop-remnote ~/.cursor/plugins/local/fanloop-remnote
```

This folder is not published to a marketplace.

## Try it

1. Run `remnote-cli status` and confirm the plugin, knowledge base, and write switch. Run one `remnote-cli` command at a time. A second command fails with `CLI port occupied`.
2. In Cursor, ask for flashcards from a short source you actually want to learn. The agent should search with `--cards-only`, draft, and run `flashcards apply` **without** `--apply` first.
3. The preview reports `validated` and `planHash` only, not the cards. Read the plan file. Only then ask it to apply, reusing the same idempotency key.
4. A `partial` or `unknown` result means stop and read back. Do not ask it to replay with a new key.
5. An empty Chinese `search` is not proof a Rem is missing. RemNote's Chinese search cache can be stale. Reload that cache before treating the miss as real.
6. A term-card plan's `title` creates the theme. The combo plan omits `title`. After that apply, `move-note` places the combo last inside the theme, and `object-remove-tag` clears the combo's own tag. The theme's HOME tag indexes it. Do not leave the combo on the daily note. Aliases are `remnote-cli sdk rem object-get-or-create-alias-with-text --target-id <ID> --args-json '[["<alias>"]]'`, one call per card. `update --add-aliases` depends on the SDK version and fails on SDK 0.0.46.
7. Reads and writes stay on `remnote-cli`. Desktop control is only for a control the CLI cannot perform, such as the Chinese search-cache reload button.

For a Direct code change, name that repository and ask for `remnote-direct-maintainer`. Card creation does not go through that skill.

Wiki edits stay off unless you ask for them. Desktop control stays off unless the CLI has no command for that control.

## Check the package

From the fanloop repo:

```bash
python3 plugins/fanloop-remnote/scripts/validate-plugin.py
```

That checks `plugin.json` against the Agent Plugins 1.0.0 schema, parses every skill frontmatter, and resolves relative links. It uses the Python standard library only (no PyYAML). Links that resolve outside the plugin directory fail. It does not talk to RemNote.

## Layout

```text
plugins/fanloop-remnote/
  plugin.json
  README.md
  scripts/validate-plugin.py
  scripts/plugin.schema.json
  skills/
    remnote-flashcards/          # card workflow checklist, writing rules, references
    remnote-cli/                 # command boundary and FlashcardsPlan
    remnote-structure/           # HOME folders and Concept tags
    remnote-verify/              # draft review and readback
    remnote-wiki/                # only when you ask for the Wiki
    control-remnote/             # desktop fallback when the CLI cannot see UI state
    remnote-direct-maintainer/   # define / build / verify / deliver checklist
    remnote-direct-define/
    remnote-direct-build/
    remnote-direct-verify/
    remnote-direct-deliver/
```

Fanloop panorama cards are not in this plugin. They need `fanloop card render`. Each step instead sends a short status in chat. The production workflow packages under `skills/remnote-flashcards/` and `skills/remnote-direct-maintainer/` are unchanged.
