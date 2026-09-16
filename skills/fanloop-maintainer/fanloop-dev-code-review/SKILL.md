---
name: fanloop-dev-code-review
description: 核验实现阶段的完整测试与独立审查证据，发布 Code Review 报告并冻结 review_base/reviewed_head。
---

# Code Review

读取 requirements.md、Spec/Tickets、相关 ADR、完整 diff 和 `implementation-report.md`，核验实现阶段记录的 review_base、implementation_head、完整验证和独立 Reviewer 结论。

1. 只接受 Approve 或 Recommend，且聚焦测试、`./tests/run-unit`、`./tests/run-e2e` 全部通过并覆盖最终 HEAD。本 Step 不重复运行测试或重做整体 CR。
2. fetch origin，要求 `origin/main == review_base`、`HEAD == implementation_head`、工作树 clean，且 `git merge-base review_base HEAD == review_base`。
3. 把 Verdict、本地验证和候选身份定稿到 `review-report.md`，按稳定标题发布并语义回读唯一飞书 Code Review 报告。
4. Result 前再次核验候选身份。通过时上报 `code_review_approved` 或 `code_review_recommended`、`local_validation_passed`、`review_report_written`、`review_base_frozen`、`reviewed_head_frozen` 与 `code_review_document_published`。

证据缺失或审查 Block 上报 `code_review_blocked`；验证失败上报 `local_validation_failed`；身份漂移上报 `candidate_changed`。三者都回实现。
