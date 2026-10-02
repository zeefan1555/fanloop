---
name: remnote-direct-define
description: Define an authorized RemNote Direct change and its impact-based acceptance.
---

# Define

Read the selected repository's root/package AGENTS.md, affected ADRs and existing
commands. Resolve its actual Git root, baseline and dirty paths; do not infer it
from a parent vault or scan private application data. Preserve unrelated changes.

Use one local `delivery-record.md` outside tracked source for the run. Record:
repository root, baseline, owned files, authorization, accepted outcome, impact,
command/check list, allowed test side effects, Git destination and installation
baseline. Keep source/build IDs and raw evidence references, not secrets or KB text.

## Impact

Select all affected rows from behavior and dependencies, not only directory names:

| Impact | Required acceptance |
| --- | --- |
| documentation | Affected commands, references and instructions remain accurate; no gratuitous KB writes |
| cli_read | Real supported reads/output and relevant failure cases |
| cli_write | Exact target/write/readback, identity/history preservation, replay and partial/unknown recovery |
| plugin_protocol | Both ends' protocol, SDK catalog, authentication, permissions, serialization and lifecycle |
| installation | Existing install path, actual command/artifact resolution and restart persistence |

Mixed work takes the union. A CLI-only directory change can still affect protocol
or installation. A read/help change does not automatically require card creation.

Choose the smallest real success and failure scenarios that cover the change.
Two-card tests are a reusable write smoke, not proof for Wiki, TODO, images or all
SDK methods. Determine current capability support; unavailable required behavior
blocks or requires a newly approved scope, never a silent exemption.

Read Git delivery policy from repository instructions and actual remote rules.
Every completed validated change must be committed and pushed to the recorded
destination. Do not override branch protection, assume PR/direct-push policy, or
publish packages/marketplace releases without authorization.

Runtime cases record the exact authorized KB and isolated fixture scope. Do not
hard-code a sample KB, day, HOME tag or test title. No deletion of user content for
cleanup, no direct database access, and no pairing secrets in logs/screenshots.
Record current CLI resolution and plugin asset/version identity when installation
may change; these are the local restore baseline, not a database rollback promise.
