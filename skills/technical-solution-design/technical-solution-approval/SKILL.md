---
name: technical-solution-approval
description: 发布完整飞书技术方案，等待人工终审并把反馈精确分类到九个最早受影响层。用于 technical-solution-design 的方案终审 Step；不得代替人批准或直接修改方案。
---

# 终审技术方案

读取 `technical-solution.md`、`.technical-solution/architecture.mmd` 和最新
`.technical-solution/review.md`。先确认审校已通过，正文只有九个规定结构且图文一致。

`<项目>` 取最新 `flow status` 中的 Requirement 标题。使用当前宿主的 `lark-doc` 能力按稳定标题 `<项目>｜技术方案` 精确查找：唯一命中更新、零命中创建、
多命中阻塞；创建结果不确定时先重新查找。发布 `technical-solution.md` 后使用返回 URL 回读，确认
正文非空，九个标题按序且无 `###`、`1.1`，架构图与关键表格存在。失败时报告 blocked，不返回
成功 Condition。

向人展示已验证 URL、方案结论、关键取舍、收益、落地风险、审校结论和最新 Panorama，然后等待
本次进入该 Step 后的全新明确回复。修改意见按最早受影响层分类：

| 层级 | Condition | 回流 Step |
|---|---|---|
| 需求背景 | `background_changed` | `frame_requirement_background` |
| 核心问题 | `problem_changed` | `analyze_core_problem` |
| 设计目标 | `objectives_changed` | `define_design_objectives` |
| 方案调研 | `research_changed` | `research_solution_options` |
| 总体方案 | `overall_solution_changed` | `design_overall_solution` |
| 难点解法 | `key_solutions_changed` | `design_key_solutions` |
| 方案收益 | `benefits_changed` | `evaluate_solution_benefits` |
| 落地规划 | `delivery_changed` | `plan_solution_delivery` |
| 仅排版、措辞、图文呈现 | `presentation_changed` | `write_technical_solution` |

回流前向人展示反馈原文、最早受影响层、保留内容、失效产物和回流 Step。多层变化只选最靠上的
一层；沉默、“看过”或继续讨论不算批准。

- 明确批准：同时上报 `technical_solution_document_published=<已回读 URL>`、
  `panorama_card_published` 与 `technical_solution_approved`，流程结束；
- 明确修改：同时上报同一文档 URL、`panorama_card_published` 与一项 feedback Condition；
- 尚需讨论：继续等待。

Evidence 保存人的完整原始回复、飞书 URL、正式方案、架构图、审校报告和影响分析。
