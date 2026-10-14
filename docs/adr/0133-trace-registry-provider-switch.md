---
status: accepted
date: 2026-10-14
amends: ADR-0080
---

# Select the Trace projection provider in the Trace Registry

second-brain-agent ADR-0014 moves run records from Lark to a Notion table.
Step one runs a Notion `run-ledger` Skill beside the existing Lark projection.
Lark stays on until a separate change turns it off. That needs a switch that
leaves the IDL alone.

Every Trace Registry entry now has a required `provider`, set to `lark` or
`none`.

- **`lark`**: keeps the existing endpoint, field and CLI-log rules.
- **`none`**: only the key is allowed. Any Lark target, field mapping or CLI-log
  requirement is rejected.

When the Registry that `flow init` resolves has `provider: none`, init creates
no Trace or CLI-log document and binds nothing, so automatic `trace sync` never
runs. Some Requirements are bound by hand under a disabled Registry. For those,
`trace sync` reports the Registry target as skipped and does not call
`lark-cli`.

This change sets every entry to `lark`, so behaviour is unchanged. A Notion
provider in Go is separate later work and will need a contract review of
`idl/trace.thrift`.
