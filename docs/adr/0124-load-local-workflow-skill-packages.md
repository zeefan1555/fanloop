---
status: accepted
date: 2026-10-01
supersedes_in_part: ADR-0120, ADR-0122
amends: ADR-0099, ADR-0109, ADR-0123
---

# Load Workflow Skill Packages From the Local Repository

The repository is the only Workflow configuration source. Each `skills/<id>/`
is a self-contained Skill package: `SKILL.md`, the five Workflow YAML files and
its bound `skills/<child-id>/` resources. Remove the Fornax wrapper
`workflows/<id>/`, remote publication/synchronization and `fanloop update`.
There is no provider, compatibility loader or fallback configuration copy.

Local installation binds an absolute repository path at
`$FANLOOP_DATA_HOME/config/current`. `FANLOOP_CONFIG_SOURCE` explicitly overrides
that binding. Flow, Doctor and Dashboard resolve the same source and validate
the same bundles. Changes are visible on the next invocation without rebuilding
the CLI. A missing or invalid source fails closed, regardless of old downloaded
files or the caller's working directory.
Only an unset override uses the binding; an explicitly empty or whitespace value
is invalid, including installation and controller pinning entrances.

Dashboard remains read-only by default. An explicit local source enables editing
that source. Shared locking, allowlisted paths, observed digests, complete Bundle
validation and atomic writes remain mandatory. The managed source binding may be
a symlink; package directories and resources may not contain symlinks or special
files. All five YAML files remain the sole workflow architecture truth.

The three retained packages are fanloop-maintainer, remnote-flashcards and
pstack-native. Their existing local resources are imported with recorded
provenance; Finder metadata is not configuration. Preserve their Step IDs, names,
order and executors. Configuration delivery becomes local editing, validation,
necessary behavior trials, Git delivery and actual loaded-path readback.
Bundled executables keep generated dependencies and runtime files outside the
validated source. The pstack tools install locked production dependencies in a
package-and-lockfile-keyed user cache, publish complete cache entries atomically,
and use Bun's native NODE_PATH resolution without changing either entrypoint.

This replaces ADR-0120's remote configuration truth and package wrapper, and
ADR-0122's remote update/bootstrap and isolated-only source restrictions. Keep
offline ordinary CLI operation, strict package validation, code/configuration
separation, immutable local CLI releases and precise candidate evidence. Keep
ADR-0109/0123's four-stage self-iteration and independent certification. Historical
Requirements remain with their original pinned controller and configuration;
neither State migration nor old-format support is introduced. Lark and RemNote
business integrations are unchanged.

The approved IDL change deletes only ConfigurationUpdateRequest,
ConfigurationUpdateResponse and OpsService.Update from idl/ops.thrift, including
the method's CLI annotations. Other Thrift contracts are unchanged.

Acceptance covers offline source loading from arbitrary working directories,
explicit overrides and rejection without fallback, immediate edit visibility,
all three complete packages, malformed or escaping packages, Dashboard read/write
safety, installation and Doctor paths, and the absence of fanloop update.

Decision: the user approved flat `skills/<workflow-id>/` packages and requested
implementation from latest master through the maintainer self-iteration flow.
The main agent approved requirements SHA256
`53152658c96a4472605e70bf1cc1bb73c3e454039198c3bb8fdfb22cfe6a081f`.
