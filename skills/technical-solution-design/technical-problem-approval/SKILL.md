---
name: technical-problem-approval
description: 将需求背景、核心问题和设计目标发布为飞书问题定义文档，等待人工审核并按最早受影响层回流。用于 technical-solution-design 的问题审核 Step；不得代替人批准或修改片段。
---

# 审核问题定义

读取 `01-background.md`、`02-problem.md`、`03-objectives.md`，组装为只含以下三个正文标题的审核稿：

```markdown
# <项目>｜问题定义
## 1. 需求背景
## 2. 核心问题
## 3. 设计目标
```

`<项目>` 取最新 `flow status` 中的 Requirement 标题。正文不得出现 `###` 或 `1.1` 编号。使用当前宿主的 `lark-doc` 能力发布：按稳定标题
`<项目>｜问题定义` 精确查找，唯一命中则更新，零命中才创建，多命中立即阻塞；创建结果不确定时
先重新查找，不得重复创建。使用返回 URL 回读，确认正文非空、三个标题顺序正确且内容与本地片段
一致。失败时报告 blocked，不返回成功 Condition。

向人展示已验证 URL、三个核心结论、开放项和最新 Panorama，然后等待本次进入该 Step 后的全新
明确回复。不得把沉默、“看过”“继续讨论”或补充材料解释为批准，也不得自行改写输入。

收到修改意见时先做整体影响分析，只选最早受影响层：

| 最早变化 | Condition | 回流 Step |
|---|---|---|
| 业务形态、现状架构、演进诉求 | `background_changed` | `frame_requirement_background` |
| 现状评估、瓶颈、根因、取舍 | `problem_changed` | `analyze_core_problem` |
| 指标、约束、非目标 | `objectives_changed` | `define_design_objectives` |

在回流前向人明确展示：反馈原文、最早受影响层、继续保留的内容、将失效的全部下游产物、回流
Step。反馈同时触及多层时只选表中最靠上的一层。

- 明确批准：同时上报 `problem_document_published=<已回读 URL>`、`panorama_card_published`、
  `technical_problem_approved`；
- 明确修改：同时上报同一文档 URL、`panorama_card_published` 和一项 feedback Condition；
- 含糊或仍在讨论：继续等待。

Evidence 保存人的完整原始回复、飞书 URL、本地三个片段路径和影响分析，不使用摘要替代原文。
