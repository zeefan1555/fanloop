---
status: accepted
date: 2026-08-17
amends: ADR-0044
---

# 将代码评审飞书文档注册为必填 Output

Workflow 10.0.0 继续使用 `fanloop-code-review` 执行真实 MR 审查，但每次形成
`Approve`、`Recommend` 或 `Block` Verdict 前，必须把本轮聚合代码评审报告发布为
一份飞书文档并回读确认正文非空。创建结果不确定、鉴权或权限缺失、发布失败、
回读失败时，当前 Step 保持 blocked，不提交 Verdict ConditionResult。

`condition.yaml` 新增原子 Condition `code_review_document_published`，输出
`code_review_document_url:url`。`review_code` 的 `Approve`、`Recommend` 正常 Route
与 `Block` 自动 Loop 都要求 Verdict Condition 和文档 Condition 同时成立。
`review_report.md` 继续作为发布源和 Evidence，`findings.json`、`verdict.json`
继续只作为 Evidence；飞书文档 URL 成为参与 Route 的当前 Output。

Card 继续按 Workflow 声明的 URL Output 通用投影，只为
`code_review_document_url` 增加“代码评审文档”显示名。不新增 CR 专用 Runtime、
飞书客户端、Output 类型、State/Event Schema 或存储路径。

本决策修订 ADR-0044 中“报告不成为必填 Output”及“Skill 不发布飞书文档”的
部分，保留三态聚合、阻塞语义和独立 Human Gate。ADR-0038 的原子 Condition、
OR-of-AND、Output 失效和通用 Runtime，以及 ADR-0039 的 Card/Output 隔离继续
有效。Workflow 10.0.0 直接替换源码与新 Release 中的 9.0.0 当前 Bundle；
运行中的旧 Requirement 继续由其已安装 Release 解释。

真实生产 YAML 的变更前后示例、影响说明与人工确认记录见
`docs/specs/code-review-feishu-artifact.md`。
