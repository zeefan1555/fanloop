---
status: accepted
date: 2026-10-14
amends: ADR-0133
---

# Turn off the Lark Trace projection in production

The Notion run ledger 「Fanloop 运行」 now records every run, and its test write
passed. second-brain-agent ADR-0014 said Lark would be switched off in a
separate change once that happened. This is that change.

- **Production profile**: the default and the `fanloop-maintainer` override are
  now `provider: none`. `flow init` no longer creates a Trace or CLI-log docx,
  and `trace sync` never upserts a Base row for production bindings.
- **Test profile**: stays `lark`. It is only reached through an explicit
  `trace bind --registry test` and keeps the Lark projection path covered by
  tests.
- **Existing bindings**: a Requirement already bound under a disabled Registry
  still decodes even when it carries a CLI-log document. `trace sync` reports
  the Registry target as skipped.

To turn Lark back on, set `provider: lark` and restore the endpoint and field
mapping from commit `ee793e6`.
