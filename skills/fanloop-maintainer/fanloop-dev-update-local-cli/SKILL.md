---
name: fanloop-dev-update-local-cli
description: 在唯一 PR 合并后固定当前 Requirement 控制器，并把精确 merge commit 从干净 detached worktree 安装为本机 Fanloop current。
---

# 更新本地 CLI

只接收已回读 `MERGED` 的唯一 PR、非空 `mergeCommit.oid`、`acceptance-report.md` 与同一飞书验收交付报告。

1. 先从当前 Skill 目录运行 `scripts/pin-controller-release.sh <ABSOLUTE_REQUIREMENT_ROOT>`，固定本 Requirement 的 `bound-release-home` 控制器与四类 Skill Root。后续 Status、Result 和 Panorama 只用该固定控制器。
2. `git fetch origin main`，证明 merge commit 可达 `origin/main`；在临时目录创建该 commit 的干净 detached worktree，验证 `HEAD` 精确相等且 `git status --porcelain` 为空。
3. 从 detached worktree 清除全部 `FANLOOP_*` 与 `BOTMUX_CHAT_ID`、`BOTMUX_SESSION_ID` 覆盖后执行 `npm run install:local`。禁止从原开发 worktree、PR head、旧 Release 或脏目录安装。
4. 回读 `$HOME/.fanloop/current` 的真实目标、`bin/fanloop version` 和 `doctor`；只有 version commit 精确等于 merge commit 且 Doctor 为 `healthy` 才成功。
5. 把命令、退出码、merge commit、current 目标、version 和 Doctor 更新到 `acceptance-report.md`；按 Requirement 稳定标题更新唯一飞书验收交付报告，回读正文非空、merge commit 和安装结论一致。
6. 上报 `local_cli_updated`、`acceptance_report_written`、`acceptance_document_published` 后终止。失败保持 blocked；成功后不回滚全局 current。

不得修改 State、手写 Release、直接改 symlink 或跳过固定控制器。
