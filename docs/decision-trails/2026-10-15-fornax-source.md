# Fornax as source, 2026-10-15

Done when all of these are true on a real tree, not on a summary:

1. A downloaded `loop-workflow-<id>` directory whose wrapper SKILL.md has only name and description materializes to a tree `workflow.LoadFrom` accepts, with workflow id taken from `workflow.yaml`, and a shebang script mode `0555`.
2. `~/.fanloop/fornax.json` is a list of `{workspace_id, key_prefix, access}` sources. The same workflow id in a read_write source and a read_only source keeps the read_write package. Two claims inside one source, or a tie across sources, reject the whole set.
3. `fanloop update` and `fanloop update config` install the latest Online/prod release for every configured prefix through the user's `fornax-cli`, with no embedded key. The snapshot directory name is the sha256 of the sorted `key@version` lines. Failure leaves `config/current` unchanged. A second identical run prints `already up to date` and does not download. A requirement initialized on snapshot A still loads A after `current` moves to B.
4. Drift reports `in_sync`, `local_behind`, `local_modified`, or `unavailable`, with the released package as the reference.
5. Leaf `--help` lists that command's error codes. `update-fanloop-workflow` edits only the personal read_write source. `scripts/mirror-fornax-to-git` copies a snapshot into a checkout path. A new ADR supersedes ADR-0136's statement that Git is the only production source.

Live Fornax is not available in this run. Network discovery is proven with a fake `fornax-cli` client. The downloaded sample tree is proven by loading the file that was uploaded.

| decision | evidence | verdict |
| --- | --- | --- |
| Skip a second design contest. The user and loop_cli ADR-0132 already fixed discovery, the receipt, the lock, and atomic replace. | docs/decision-trails/2026-10-15-fornax-source.md | INCONCLUSIVE |
| Do not port panorama-as-runtime-action. Fanloop already delivers panorama through the workflow condition. A second `flow action acknowledge` command would add IDL, state, and a second delivery path. | loopcli log commits 6c9797b and 3d3e19a; docs/adr/0119-require-panorama-condition-in-every-workflow.md | INCONCLUSIVE |
| Do not port live workflow reconciliation. Running requirements will stay on the snapshot they were initialized with, so a later config swap does not rewrite their steps. | loopcli log commits a1aafe8 and 2d5730c; docs/adr/0019-pin-running-requirements-to-workflow-versions.md | INCONCLUSIVE |
| Restore executable bits by treating a leading `#!` as mode 0555. Fornax does not return Unix modes, and loop packages have no mode list. | internal/fornaxmirror/accept.go | INCONCLUSIVE |
| Reuse the existing `.workflows.lock` for update and update config. A second lock can race with flow init on the same `current` symlink. | internal/workflow/configuration.go | INCONCLUSIVE |
| The uploaded loop-workflow-pstack-native tree loads after the wrapper SKILL.md is copied into the workflow directory. workflow id is pstack-native. worktree-audit.sh is mode 0555. | internal/fornaxmirror/accept_test.go TestMaterializeRealLoopCLISample | VERIFIED |
