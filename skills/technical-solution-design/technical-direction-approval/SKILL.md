---
name: technical-direction-approval
description: 将前六个方案片段与架构图发布为飞书方案设计文档，等待人工审核并按最早受影响层回流。用于 technical-solution-design 的方案审核 Step；不得代替人批准或修改片段。
---

# 审核方案设计

读取 `01-background.md` 至 `06-key-solutions.md` 和 `.technical-solution/architecture.mmd`，组装为只含
“需求背景、核心问题、设计目标、方案调研、总体方案、难点解法”六个 `##` 语义章节的审核稿。
允许 `###` 按真实内容组织，但禁止 `####` 和手工 `1.1` 编号；主架构图必须嵌入“总体方案”且可
独立理解，片段明确引用的辅助图随所属章节发布。

`<项目>` 取最新 `flow status` 中的 Requirement 标题。使用当前宿主的 `lark-doc` 能力按稳定标题 `<项目>｜方案设计` 精确查找：唯一命中更新、零命中创建、
多命中阻塞；创建结果不确定时先重新查找。使用返回 URL 回读，确认正文非空、六个 `##` 顺序
正确、`###` 均归属当前语义章节、架构图存在且内容与本地片段一致。失败时报告 blocked，不返回
成功 Condition。

向人展示已验证 URL、推荐方向、关键取舍、主要风险和最新 Panorama，然后等待本次进入该 Step 后
的全新明确回复。收到修改意见时评估对整体的影响，只选最早受影响层：

| 最早变化 | Condition | 回流 Step |
|---|---|---|
| 需求背景 | `background_changed` | `frame_requirement_background` |
| 核心问题 | `problem_changed` | `analyze_core_problem` |
| 设计目标 | `objectives_changed` | `define_design_objectives` |
| 评价维度、候选或对比事实 | `research_changed` | `research_solution_options` |
| 核心选型、架构全景或关键链路 | `overall_solution_changed` | `design_overall_solution` |
| 局部手段或异常保障 | `key_solutions_changed` | `design_key_solutions` |

回流前向人展示反馈原文、最早受影响层、保留内容、失效产物和回流 Step。多层变化只选最靠上的
一层；不得把含糊表达推断为批准。

- 明确批准：同时上报 `solution_document_published=<已回读 URL>`、`panorama_card_published`、
  `solution_direction_approved`；
- 明确修改：同时上报同一文档 URL、`panorama_card_published` 和一项 feedback Condition；
- 尚需讨论：继续等待。

Evidence 保存人的完整原始回复、飞书 URL、六个片段与架构图路径以及影响分析。
