---
name: remnote-direct-verify
description: Verify the frozen candidate against the actual change, not a fixed demo.
---

# Verify

Confirm the commit, source and build digests still match Build. Execute the defined
cases using the candidate's public CLI, with an isolated PATH/install location when
needed. Existing guarded repository integration suites are development entries;
daily smoke uses the installed `remnote-cli` command directly.

Documentation/read-only cases do not acquire write permissions or create cards.
For runtime cases, verify real plugin/protocol/catalog/SDK and the exact authorized
KB using status/capabilities. Read permission and authenticated write permission
are distinct. A manifest or successful reload is not proof of effective permission.

Only plugin-impacting work loads candidate plugin assets. Use the recorded static
asset service and desktop installation path, preserve the previous installation,
and inspect runtime identity rather than a cached installer manifest. Do not
publish to a marketplace or expand supported platforms. Preserve action-time
permission confirmations required by the computer-control skill.

Use the existing native baseline plus the affected capability's own cases:

- Card/task writing: preview has no side effects; apply returns known IDs; exact-ID
  readback proves native cards/text/parent/Tag as applicable. Same-plan/key replay
  preserves Rem IDs, native card IDs, body and review history.
- Wiki/index work: flat and grouped indexes, new/existing pages and entries, exact
  target parent, ambiguous/foreign targets, no duplicate owner and zero unintended
  cards. Do not validate every feature with two cards.
- Connection/protocol work: authentication, wrong KB, permissions, timeout and
  disconnect; relevant desktop lifecycle must not hang or require force quit.
- Read/serialization work: supported result shapes and explicit failures; a read
  regression can remain read-only.
- Installation work: actual executable/assets, cold start and normal restart,
  without dependence on a temporary terminal or development server.

For CLI-only composition, record that the same plugin artifact served acceptance.
Current supported capabilities bound the test set. Do not invoke known unsupported
native transactions, pretend rollback works, or run an unrelated broad suite that
requires unsupported operations. List genuinely untested capabilities explicitly.

Unknown or partial writes stop further writes. Preserve exact plan/key/IDs in
private recovery storage, inspect persisted results, and never blindly replay,
change keys or delete/recreate. A local installation restore is not a KB rollback.
Return source defects to Build; authorization/environment failures stay blocked.

Record each case as passed/failed or not_applicable with its impact-based reason.
Unavailable required cases are failed/blocked, not not_applicable. Raw command
results and native readback must support the verdict; fixture simulations prove
only their stated scope. Measure cold connection, relevant operation and replay
separately from planning and environment wait, with workload/mode recorded.
