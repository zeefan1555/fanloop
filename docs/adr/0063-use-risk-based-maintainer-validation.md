---
status: accepted
date: 2026-08-20
amends: ADR-0055, ADR-0059
---

# 自迭代使用按风险分档的本地验证

Fanloop 保留 `./tests/run-unit` 与 `./tests/run-e2e` 两个仓库测试入口，但维护者 Workflow
不再为每个变更在本地重复执行两者。Codebase MR gate 继续在最终 MR HEAD 上强制运行
`run-unit`，作为格式、生成物、静态检查和全部非 Requirement E2E 测试的确定性门禁。

`fanloop-maintainer` 的测试计划必须选择 `targeted` 或 `e2e`。`targeted` 只执行计划列出的
聚焦测试和必要格式检查；`e2e` 还必须从同一工作树运行 `run-e2e`。Step/Route/Condition
推进语义、Thrift IDL、durable state/storage/output/trace/card、发布/安装/更新/打包、测试入口、
缺少可靠聚焦 seam 或影响面不确定的变更强制使用 `e2e`。文档、自迭代 Skill、未改变推进
语义的 Prompt/SkillBinding 和具备完整聚焦 seam 的局部叶子行为可以使用 `targeted`。

测试资产变化后必须提交并推送，回到 MR gate 与 AI Code Review；只有覆盖最终 HEAD 的远端
`run-unit` 和 Review 才有效。不增加第三个入口、路径分类器、配置开关、兼容 alias 或 Runtime
专用分支。风险无法确定时直接升级为 `e2e`。

因此本决策修订 ADR-0055 中“任何产品或测试行为变更提交前本地运行两个入口”的条款：两个
入口继续存在，远端 `run-unit` 始终必需，本地 `run-e2e` 只在 `e2e` profile 必需。它也修订
ADR-0059 中 maintainer Test Stage 固定运行双入口的条款，保留四个 Test Step、测试资产回
MR 门禁与 AI Review、最终人工 MR 审核以及全部既有回流目标。ADR-0062 的当前 Bundle、digest
和无业务版本边界不变。

本次迁移自身修改生产 Workflow Route/Condition，仍按变更前有效门禁完整运行一次
`run-unit` 与 `run-e2e`。精确 YAML 前后片段、12 Step 零变化审查记录和验收条件见
`docs/specs/agile-maintainer-validation.md`。
