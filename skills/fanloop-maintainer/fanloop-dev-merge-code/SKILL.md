---
name: fanloop-dev-merge-code
description: 在 Fanloop Agent 验收通过后，将精确 reviewed HEAD 通过唯一 GitHub PR 安全地 squash 合并到 main；不用于人工 MR 交接或通知。
---

# 合码

只在 `agent_acceptance_passed` 已被 Workflow 接受，且 `local-test-report.md`、`review-report.md`、
`acceptance-report.md` 都绑定当前干净 HEAD 时执行。任一候选事实漂移都回到实现，不沿用旧验证。

## 合并前检查

1. 要求当前分支不是 `main`，`origin` 精确指向 `zeefan1555/fanloop`；读取分支、HEAD 与三个报告。
2. 执行 `git fetch origin main`，要求候选相对最新 `origin/main` 为零 behind 且至少一条 ahead。
   base 漂移时不自行 rebase 或 merge，记录 `implementation_changes_requested` 并回到实现。
3. 用 `gh pr list --repo zeefan1555/fanloop --state all --head <branch>` 查找该分支的 PR。只能存在
   一个权威 PR；多个、错误 base、draft 或 head OID 不等于 reviewed HEAD 时停止。

## 创建并合并

1. `git push -u origin HEAD` 后，无 PR 时创建 `base=main` 的 PR；唯一 open PR 时更新标题和详细正文。
   正文包含背景、问题、解决方案、改动范围、本地测试、Agent 验收、ADR impact 和风险/非目标。
2. 回读 PR，要求 `baseRefName=main`、`headRefName=<branch>`、`headRefOid=<reviewed HEAD>`、
   `state=OPEN`、`isDraft=false`。不查询或等待远端 checks；本 Workflow 的合并门禁只使用当前
   reviewed HEAD 的本地验证、代码审查和 Agent 验收事实。
3. 使用以下形状合并，不使用 `--admin`、`--auto`、merge commit、rebase 或直接 push `main`：

   ```bash
   gh pr merge <PR URL> --repo zeefan1555/fanloop --squash --match-head-commit <reviewed HEAD>
   ```

4. 再次回读同一 PR，只有 `state=MERGED`、base/head/head OID 仍匹配且 `mergeCommit.oid` 非空才成功。
   `main` push 触发的 Release 是独立异步流程；本 Skill 不等待、不重跑或宣称发布成功。

## 结果

将 PR URL、source/base、reviewed HEAD、merged 时间和 merge commit 写入并回读
`acceptance-report.md`。成功后上报 `code_merged=<merge commit SHA>` 与
`acceptance_report_written=acceptance-report.md`。

候选或 base 漂移上报 `implementation_changes_requested` 与 `acceptance_report_written`。网络或
GitHub 暂时失败时，先回读 PR；只有能证明尚未产生歧义合并时，才上报
`code_merge_failed=failed` 与 `acceptance_report_written` 原地重试。不得发送 Botmux 消息、写
`handoff.json`、请求人工验收、自动发布或使用其他仓库/身份兜底。
