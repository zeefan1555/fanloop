---
name: update-fanloop-workflow
description: 已退役。编辑 second-brain-skills main 并通过 PR 合并，然后运行 fanloop update config。
---

# 已退役

这个 Skill 不再向 Fornax 发布。编辑 `zeefan1555/second-brain-skills` 的 `main`（`fanloop-workflow/<workflow-id>/`），通过 PR 合并，然后运行 `fanloop update config` 同步本地 Git 快照。

`FANLOOP_USE_FORNAX=1` 且 `fornax.json` 是普通文件时，`fanloop update config` 仍走 Fornax 同步。那条路径不是默认。
