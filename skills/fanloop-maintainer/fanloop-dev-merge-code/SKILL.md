---
name: fanloop-dev-merge-code
description: 发布唯一 GitHub PR，校验 candidate_head 的 Ruleset 与 required checks，并自动 squash 合并后更新验收交付报告。
---

# 合并 MR

只接收同一 `candidate_head` 的本地验证、Review、隔离安装与 Sub-agent 验收通过事实。当前工作树必须干净且 HEAD 不漂移。

1. 验证 origin 为 `zeefan1555/fanloop`、当前分支非 main；幂等 push candidate_head。查找该 head 分支到 main 的 PR：零命中创建，唯一命中更新，多命中 blocked；不得创建第二个 PR。
2. 回读 PR 的 base、head、head OID、draft/state 和 main Ruleset。required checks 必须在精确 candidate_head 成功，至少包含 Ubuntu/macOS test、requirement-e2e、install-doctor 与 governance；等待中保持 blocked，确定实现失败回流。
3. 全部门禁成功后执行：

~~~bash
gh pr merge "$PR_URL" \
  --repo zeefan1555/fanloop \
  --auto \
  --squash \
  --match-head-commit "$CANDIDATE_HEAD"
~~~

禁止 `--admin`、直接 push main、merge commit、rebase、人工端到端验收或绕过 required checks。只有回读 `state=MERGED`、base/head 仍匹配且 `mergeCommit.oid` 非空才成功。

把 PR URL、candidate_head、Ruleset、checks、mergedAt 和 merge commit 更新到 `acceptance-report.md` 与同一飞书验收交付报告；语义回读非空且事实一致后上报 `code_merged`、`acceptance_report_written`、`acceptance_document_published`。候选漂移回实现；平台暂时失败先回读 PR，只有确认没有歧义合并时才上报 `code_merge_failed` 原地重试。
