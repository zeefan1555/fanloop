---
name: technical-problem-approval
description: 将业务背景、目标问题和业务技术约束发布为飞书业务问题文档，等待人工审核并按最早受影响章回流。用于 technical-solution-design 的问题与约束审核 Step；不得代替人批准或修改章节。
---

# 审核业务问题与约束

## 产物列表

| 逻辑产物 | 承载与完整性要求 |
|---|---|
| 汇总的业务问题文档 | 稳定飞书文档 `<项目>｜业务问题`；完整承载第 1 至第 3 章并在发布后回读 |
| 审核结论与反馈 | 人的本轮明确回复；完整原文与影响分析保存在 Evidence |

读取 `01-business-background.md`、`02-goals-and-problems.md` 和
`03-business-constraints.md`，组装为：

```markdown
# <项目>｜业务问题
## 1. 业务背景
## 2. 目标与问题定义
## 3. 业务特点与技术约束
```

各 `##` 下允许 `###` 按真实内容组织，禁止 `####` 和手工 `1.1` 编号。

使用当前宿主的 `lark-doc` 能力按稳定标题 `<项目>｜业务问题` 精确查找：唯一命中更新、零命中
创建、多命中阻塞。发布后用返回 URL 回读，确认三个章节顺序正确、内容与本地片段一致。文档发布
不代表人工批准。

向人展示已验证 URL、核心问题、目标、约束、开放项和最新 Panorama，然后等待本次进入该 Step 后
的全新明确回复。修改意见只选最早受影响章：

| 最早变化 | Condition | 回流 Step |
|---|---|---|
| 业务背景 | `background_changed` | `frame_requirement_background` |
| 目标与问题 | `goals_and_problems_changed` | `define_goals_and_problems` |
| 业务特点与技术约束 | `business_constraints_changed` | `define_business_constraints` |

- 明确批准：同时上报 `problem_document_published`、`panorama_card_published` 和
  `technical_problem_approved`；
- 明确修改：同时上报同一文档 URL、Panorama 和一项 feedback Condition；
- 含糊、沉默或继续讨论：继续等待。

Evidence 保存完整原始回复、飞书 URL、三个片段路径、保留内容、失效产物和回流 Step。
