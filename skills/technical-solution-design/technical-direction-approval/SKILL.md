---
name: technical-direction-approval
description: 将前七章与架构图发布为飞书技术判断文档，等待人工审核并按最早受影响章回流。用于 technical-solution-design 的方案与决策审核 Step；不得代替人批准或修改章节。
---

# 审核方案与技术决策

## 产物列表

| 逻辑产物 | 承载与完整性要求 |
|---|---|
| 汇总的技术判断文档 | 稳定飞书文档 `<项目>｜技术判断`；完整承载第 1 至第 7 章、主架构图和引用的辅助图 |
| 审核结论与反馈 | 人的本轮明确回复；完整原文与影响分析保存在 Evidence |

读取第 1 至第 7 章和 `.technical-solution/architecture.mmd`，组装为只含以下七个正文标题的审核稿：

```markdown
# <项目>｜技术判断
## 1. 业务背景
## 2. 目标与问题定义
## 3. 业务特点与技术约束
## 4. 业界/业内方案调研
## 5. 总体方案设计
## 6. 关键模块设计
## 7. 核心技术决策与取舍
```

各 `##` 下允许 `###` 按真实内容组织，禁止 `####` 和手工 `1.1` 编号。

使用当前宿主的 `lark-doc` 能力按稳定标题 `<项目>｜技术判断` 精确查找：唯一命中更新、零命中
创建、多命中阻塞。发布后回读七章、主架构图和引用的辅助图。文档发布不代表人工批准。

向人展示已验证 URL、推荐方向、关键模块、核心取舍、主要风险和最新 Panorama，再等待全新明确
回复。反馈只选以下最早受影响章：`background_changed`、`goals_and_problems_changed`、
`business_constraints_changed`、`research_changed`、`overall_solution_changed`、
`key_modules_changed` 或 `technical_decisions_changed`。

- 明确批准：同时上报 `solution_document_published`、`panorama_card_published` 和
  `solution_direction_approved`；
- 明确修改：同时上报同一文档 URL、Panorama 和一项 feedback Condition；
- 含糊或继续讨论：继续等待。

Evidence 保存完整原始回复、飞书 URL、七个章节与架构图路径、保留内容、失效产物和回流 Step。
