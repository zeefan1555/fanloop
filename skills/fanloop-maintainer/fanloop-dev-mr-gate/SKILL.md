---
name: fanloop-dev-mr-gate
description: 在最终 Handoff 中确定 final pair，发布唯一 GitHub PR，校验 required checks 并同步已有 Review 结论。
---

# GitHub PR 门禁

只接受整体 Review、本地验证、Agent 验收和 human 验收覆盖的同一 `review_base` / `reviewed_head`。不重做前序整体 Review。

1. `git fetch origin`，确认工作树 clean、`HEAD == reviewed_head` 且 merge-base 等于 review_base。
2. `origin/main == review_base` 时上报 `handoff_main_unchanged`。若只有 main 前进，在当前 Step 合入精确 SHA；冲突使用 `resolving-merge-conflicts`。合入后对 `reviewed_head..final_head` 和 `final_base...final_head` 做独立 CR，运行受影响聚焦测试与 `./tests/run-unit`，写入 `handoff-integration-report.md`；只在 human 明确确认未改变原批准需求后上报集成四项 Condition。
3. 用 `gh pr list --head <branch> --base main` 查找 PR。零命中才 push 并创建，唯一命中幂等更新，多命中 blocked；不创建第二个 PR。
4. 回读 PR target=`main`、source branch、base SHA、head SHA 和 merge-base，必须分别等于 final base、final head、final base。
5. 使用 GitHub CLI 回读当前 main Ruleset 和 final head 的 required checks。`test (ubuntu)`、`test (macos)`、`requirement-e2e`、`install-doctor`、`governance` 必须全部成功；pending 留在当前 Step，确定失败上报 `remote_checks_failed`。
6. 把既有 `review-report.md` 结论同步到 PR 评论并回读；已集成 main 时同步集成 CR。交接前再次 fetch 并校验 final pair；main 再前进则重复集成短门禁。

本 Skill 不 approve、不 merge、不发布、不更新全局 CLI。
