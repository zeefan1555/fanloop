---
name: fanloop-dev-code-review
description: 对最终本地候选运行风险验证并独立审查，更新研发实现报告后冻结 candidate_head。
---

# Code Review

固定比较点为 bootstrap 记录的 main Commit。读取 requirements.md、Spec/Tickets、相关 ADR、仓库规则、完整 main...HEAD diff 与 `implementation-report.md`，记录 reviewed HEAD。

1. 审查批准决策、五份生产 YAML、Test Seam、实现、删除项、发布/安装路径和 ADR 一致性；任何 Thrift diff 必须能定位到编码前的精确人工批准。
2. 运行计划中的聚焦命令，并从同一工作树执行 `./tests/run-unit` 与 `./tests/run-e2e`。全部命令、退出码、报告路径、测试前后源码状态和 reviewed HEAD 必须一致；测试改变源码或候选变化即失效。
3. 把 findings、验证证据、ADR impact 与结论更新到 `implementation-report.md`。
4. 按稳定标题更新同一飞书研发实现报告；语义回读正文非空、reviewed HEAD、findings 和结论一致。
5. 无阻断项且工作树干净时上报 `review_passed`、`implementation_report_written`、`implementation_document_published` 与 `candidate_head_frozen=<完整 SHA>`。

需求冲突回 `requirements_changed`；方案或公开 Seam 不可靠回 `technical_solution_changes_requested`；普通实现阻断回 `review_failed`。三种回流都必须先更新本地报告和飞书文档。任何修复形成新 HEAD 后，旧验证与 Review 全部失效。
