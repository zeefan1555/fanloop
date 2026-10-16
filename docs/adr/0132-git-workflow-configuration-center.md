---
status: accepted
date: 2026-10-10
supersedes_in_part: ADR-0124-load-local-workflow-skill-packages, ADR-0128
amends: ADR-0099, ADR-0120, ADR-0122
amended_by: ADR-0137-take-workflow-configuration-from-fornax, ADR-0138-refresh-cli-and-git-config-on-fresh-init
---

# Fetch immutable Workflow configuration from second-brain-skills

The only production source is git@github.com:zeefan1555/second-brain-skills.git,
refs/heads/main, under fanloop-workflow/<id>. New flow init calls fetch
that exact commit through system Git/SSH, validate the complete collection, and
atomically activate config/snapshots/<sha> through config/current under the
stable configuration lock. configuration.json records repository, ref, commit,
Workflow IDs and a manifest of relative paths, SHA256 and executable bits.
Snapshots are never edited in place. A failed fetch, validation or activation
preserves current and creates no Requirement State. Dry-run validates only a
temporary candidate. Existing flow status/report/skills calls remain offline.

Explicit FANLOOP_CONFIG_SOURCE means a repository root with this same category;
it bypasses fetch and supports authors. Dashboard is read-only unless an explicit
author source is set. Installation stays offline: an explicit clean local Git
commit seeds a snapshot; omission preserves existing configuration and permits
first installation before the first init. No update command, provider selection,
tags, releases, compatibility loader, snapshot cleanup or dependency is added.
State and YAML schemas are unchanged. ErrorCode 31 CONFIGURATION_FETCH_FAILED
is external, exit 1 and retryable; only Init adds it to its error list.

Production packages are removed from this code repository. Three packages use
Fanloop 7833b38; pstack-native uses its last complete 0d4594f tree, unchanged.
The user's 2026-10-10 explicit four-package migration instruction supersedes only
ADR-0128's pstack retirement/inventory decision. Its flow skills catalog remains.
The accepted standalone configuration commit is
5068777774ba1f1213e3b5a30bca8bce8ff3b3ac.

Keep immutable CLI releases, precise merge installation, independent review and
public CLI certification. Keep strict complete-package validation, source locks,
Dashboard atomic saves and the separation of code from configuration. Generic
runtime fixtures replace code tests coupled to production prose.

Acceptance: remote init and exact identity, idempotence, main advancement and
whole-collection deletion, offline existing flows, dry-run, invalid/missing and
symlink package rejection, failed activation preservation, concurrent callers,
offline seeded and fresh installation, author editing and default read-only view.
Approved requirements SHA256:
01f1293f7237f70a689481366f9877514cf434d42d55084ab934ecae9a809c94.
