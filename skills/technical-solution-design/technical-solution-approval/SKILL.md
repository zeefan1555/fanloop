---
name: technical-solution-approval
description: 发布完整飞书技术文档，等待人工终审并把反馈精确分类到最早受影响章。用于 technical-solution-design 的文档终审 Step；不得代替人批准或直接修改文档。
---

# 终审技术文档

## 产物列表

| 逻辑产物 | 承载与完整性要求 |
|---|---|
| 最终发布文档 | 稳定飞书文档 `<项目>｜技术文档`；与已通过审校的本地正文和图一致 |
| 人的审核结论与反馈 | 人的本轮明确回复；完整原文与影响分析保存在 Evidence |

读取 `technical-solution.md`、主架构图、正文引用的辅助图和最新
`.technical-solution/review.md`。先确认独立审校已通过，正文恰好包含第 0 至第 10 章。

使用当前宿主的 `lark-doc` 能力按稳定标题 `<项目>｜技术文档` 精确查找：唯一命中更新、零命中
创建、多命中阻塞。发布后用返回 URL 回读，确认十一章顺序、摘要行数、图文和关键表格完整；允许 `###`
按真实内容组织，禁止 `####`、手工 `1.1` 和附录章节。发布和 Agent 审校均不能代替人的终审。

向人展示已验证 URL、业务价值、核心问题、方案、关键取舍、结果、贡献边界、复用边界、规划、
审校结论和最新 Panorama，再等待本次进入该 Step 后的全新明确回复。

修改意见按最早受影响章分类为：`background_changed`、`goals_and_problems_changed`、
`business_constraints_changed`、`research_changed`、`overall_solution_changed`、
`key_modules_changed`、`technical_decisions_changed`、`delivery_changed`、`results_changed`、
`retrospective_changed`、`summary_changed`；只涉及最终标题、顺序或图文呈现时使用
`presentation_changed`。

- 明确批准：同时上报 `technical_solution_document_published`、`panorama_card_published` 与
  `technical_solution_approved`，流程结束；
- 明确修改：同时上报同一文档 URL、Panorama 与一项 feedback Condition；
- 含糊、沉默或继续讨论：继续等待。

Evidence 保存完整原始回复、飞书 URL、正式文档、架构图、审校报告、保留内容、失效产物和回流 Step。
