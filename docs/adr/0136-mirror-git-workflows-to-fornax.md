---
status: accepted
date: 2026-10-15
supersedes_in_part: ADR-0124-load-local-workflow-skill-packages
amends: ADR-0132-git-workflow-configuration-center
preserves: ADR-0124-automate-workflow-yaml-validation, ADR-0125-make-ordinary-skill-maintenance-lightweight
---

# Mirror Git Workflow packages to personal Fornax

second-brain-skills `main` remains the only production Workflow source. New `flow init` fetches and activates its immutable Git commit; existing Requirements stay offline. Fornax is a post-merge personal-space mirror for discovery and audit. Runtime, installation, Doctor, Dashboard and Flow never load or fall back to Fornax.

Every package follows Loop's archive contract. The ZIP root contains a generated wrapper `SKILL.md` and exactly one `<workflow-id>/` directory. That directory contains the five YAML files, optional `trace.yaml`, and complete `skills/**`; the Git package entry `SKILL.md` is not copied inside it. Wrapper metadata records Workflow ID, exact `fanloop.workflow.<snake_id>` key, source repository, source commit and stable SemVer. Archive validation rejects unsafe paths, duplicate or special entries, unexpected roots, missing YAML or Skills, ID/key mismatches and changed bytes. ZIP builds still record the executable bit for files that are executable in Git. Fornax `skill install` does not preserve Unix file modes. `tools/workflow-package compare --archive --directory` and `fanloop config drift` compare relative paths and file bytes only. A mode-only difference does not fail; it may be reported as an informational note.

The mirror inventory is discovered from the complete Git configuration collection and must equal the credential-free `${FANLOOP_DATA_HOME:-$HOME/.fanloop}/fornax.json` Workflow-to-key map. That file stores only personal workspace and keys. It never stores AK/SK, JWT, download token, Skill ID or version. Runtime and installer do not create it. The publication Skill updates it only during a separately authorized personal-space publication.

`fanloop config drift` is a read-only public command. It compares the active managed Git snapshot with the latest updated Online/prod released record for each exact key through the user's logged-in `fornax-cli`, downloads into private temporary storage, validates the package, and compares wrapper plus every relative path and file byte. Unix file modes are ignored because Fornax does not preserve them. A mode-only difference stays `in_sync` and may appear as an informational detail. `in_sync` exits zero; explicit `drifted` or `unavailable` results exit one. Doctor remains offline because it also supplies ordinary command notices.

Publication order is fixed: validate the Git candidate, merge the second-brain-skills PR to `main`, run a fresh local `flow init` and read back the exact commit, rebuild from that clean commit, create/commit/release in the personal workspace recorded by `fornax.json`, confirm `released`, download and compare, then require `config drift` to report `in_sync`. Unknown write outcomes are read back before retry. Any other workspace is outside this design.

The initial migration creates new snake-key Skills and does not reuse or delete the three legacy hyphen-key Skills. The active package set follows Git main. The deleted `playbook-flow` package is not restored.

This partially supersedes ADR-0124-load-local-workflow-skill-packages only where it forbids remote publication and a Fornax wrapper. It amends ADR-0132 only by adding the post-merge mirror. Git main, immutable snapshots, offline existing Requirements, strict package validation, code/config separation, and no provider, fallback or compatibility loader remain unchanged.

The Main Agent approved requirements SHA-256 `7f75fd5fe305f43d152296714feee3f3d880f08519cb5a111cef715ed6e32bf7` and the exact `idl/ops.thrift` ConfigDrift contract before implementation.
