---
name: fanloop-dev-merge-code
description: 在 Ruleset、CI 和两机器人验收通过后，对精确 candidate_head 启用 GitHub 自动 squash 合码。
---

# 自动合码

只接收 acceptance-report.md 中同一 candidate_head 的 Eval 通过、Ruleset ID、全部必需 CI checks 和
两机器人验收事实。PR 必须唯一、base=main、open、非 draft，head OID 精确匹配；当前工作树只读。

## 执行

回读 PR 和 main Ruleset 后执行：

~~~bash
gh pr merge "$PR_URL" \
  --repo zeefan1555/fanloop \
  --auto \
  --squash \
  --match-head-commit "$CANDIDATE_HEAD"
~~~

禁止 --admin、直接 push main、merge commit、rebase、第二个 PR、人工端到端验收或绕过 required
checks。命令返回后持续回读同一 PR；只有 state=MERGED、base/head/head OID 仍匹配且 mergeCommit.oid
非空才成功。自动合并仍在等待 checks 时保持 blocked，不把已启用 auto-merge 记为已合并。

## 结果

把 PR URL、candidate_head、Ruleset ID、checks、mergedAt 和 merge commit 追加并回读
acceptance-report.md，随后上报 code_merged 与 acceptance_report_written。候选或 base 漂移回到实现；
平台暂时失败时先回读 PR，只有能证明未发生歧义合并才上报 code_merge_failed 原地重试。
