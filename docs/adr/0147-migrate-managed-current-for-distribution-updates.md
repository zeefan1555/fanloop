---
status: accepted
date: 2026-10-26
amends: ADR-0136, ADR-0140, ADR-0146
---

# Migrate managed current directories at distribution entrypoints

AGE-163 authorizes npm postinstall and `fanloop update` to convert a validated native installation directory into the distribution symlink layout. The directory must contain exactly the four managed links `bin`, `entrypoints`, `release.json`, and `release`. The first three retain the native installer's targets. `release` must select a direct real child of a real `releases` directory. Unknown entries, including old live Workflow data, are refused rather than hidden.

After release validation and Skill-link preparation, create a sibling symlink and atomically exchange it with `current`. macOS uses `RENAME_SWAP`; Linux uses `RENAME_EXCHANGE`, through the existing `golang.org/x/sys` dependency. The sibling `current.backup-*` is now the original directory with its original inode and links. Keep it indefinitely. A failed exchange removes only the prepared symlink and leaves current untouched. Do not use a two-rename fallback. Serialize distribution activation with a data-root lock and revalidate under that lock.

The Go layout package owns the native directory validator and distribution activation. npm invokes the verified downloaded binary's hidden `__activate-current` command only when current is a directory. It does not add a native Node dependency. Ordinary symlink installation and native development installation retain their existing layouts.

Same-version repairs retain the replaced release at `releases/.backup-<version>-*` instead of deleting it. If Skill, configuration or activation fails, atomically exchange that backup with the repaired release to restore the original release inode, binary and user bytes. npm delegates this restore to the same verified native binary. Keep both trees after restoration. The shared lock lives in `releases`, so a read-only data-root parent cannot prevent restoring writable release children. Revalidate the activation target after taking the lock; a native rollback during the wait must not produce a dangling current.

For directory migrations, public update synchronizes configuration using the staged new binary before activation. Configuration failure therefore leaves the original current directory unchanged. It does not re-execute a second sync after activation. npm validates and writes its PATH profiles before directory activation so a profile failure cannot leave a partially migrated installation. Its existing best-effort configuration sync remains non-fatal.

The no-downgrade rule also reads directory installations. If a newer release is installed, update keeps that directory and version. Fresh flow initialization continues to leave development directories offline; only explicit public update migrates them. Native installation and distribution activation share the current lock so a native installation cannot roll back a release selected by a concurrent migration. This decision does not change storage schemas, Workflow configuration, production data, or existing published release assets.

The user approved AGE-163 on October 26, 2026 at 14:05 SGT. Migration and failure-preservation requirements are tracked in the Go and npm managed-current regression tests.
