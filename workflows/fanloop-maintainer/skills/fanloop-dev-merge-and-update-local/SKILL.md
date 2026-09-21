---
name: fanloop-dev-merge-and-update-local
description: 只交付已认证候选；合并唯一 PR，校验 merge tree，并从精确 merge commit 更新源码 worktree 和本机 current。
---

# Merge And Update Local

只接受 certification-report.md 覆盖的同一 certification base/head、Git tree、candidate binary SHA-256
和验证资产摘要。

1. 读取可选 `delivery-record.md`，先回读当前分支到 main 的唯一 PR，并跨 open/merged 状态核对。多命中
   blocked。若 PR 已为 `MERGED`，校验记录一致后固定其非空 merge commit，跳过 base drift 和再次合并，
   上报 `delivery_pr_already_merged=merged`，直接从第 6 步继续合并事实核验与本地恢复；该路径不要求
   `delivery_main_unchanged`。
2. PR 尚未合并时执行 `git fetch origin`。若 `origin/main != certification_base`，不得合入、rebase 或修改
   候选；返回完整新 main SHA，交给 Workflow 以 `delivery_main_advanced` 回 Build。
3. main 未变化时，要求工作树 clean、HEAD 等于 certification_head、tree 与报告一致。零 PR 才 push/create，
   唯一 open PR 幂等更新。PR 描述包含背景、问题、解法、影响、验证、ADR impact、主 Agent结论和交付边界。
4. 通过 GitHub API 读取作用于 `main` 的 active Ruleset，从 require-status-checks 规则动态提取 required checks
   的精确名称集合；不得假定、补写或硬编码检查名。逐项回读这些检查在精确 certification_head 的结论，
   并同步、回读独立 Review 评论。产品失败回 Build；基础设施失败 blocked。已合并恢复也必须对原
   certification_head 重做这项核验。
5. 合码前再次 fetch；main 前进立即回 Build。否则执行
   `gh pr merge <PR_URL> --repo zeefan1555/fanloop --auto --squash --match-head-commit <CERTIFICATION_HEAD>`。
   禁止 `--admin`、直接 push main 或第二个 PR。
6. 回读 PR 为 MERGED、merge commit 非空、恰好一个 parent 且等于 certification_base，并可达
   origin/main。比较 `git rev-parse <merge>^{tree}` 与 `git rev-parse <certification_head>^{tree}`，必须相同。
7. 立即把 PR、checks、review receipt、认证 head/tree、merge commit/tree 写入 delivery-record.md。
8. 固定当前 Requirement 控制器。只把本 Requirement 的 clean 源码 worktree 切到 detached merge commit；
   不切换、重置、清理或删除其他 checkout。
9. 清除 data、config、Skill Root 和 Botmux 覆盖，从 merge commit 执行 `./scripts/install-local.sh`。
   全局 current 的 version commit 必须等于 merge commit，Doctor healthy，并运行 `fanloop verify smoke`。
10. 补全并回读 delivery-record.md。PR 已合并后的失败只重试同一 merge commit，不创建或合并第二个 PR。

全部通过后返回 `merge_request_published`、`remote_checks_passed`、`review_comment_synced`、`code_merged`、
`merge_tree_verified`、`source_repository_updated`、`local_cli_updated`、`post_merge_smoke_passed` 与
`delivery_record_written`。本 Skill 不发布远端制品。
