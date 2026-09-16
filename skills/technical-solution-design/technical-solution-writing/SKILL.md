---
name: technical-solution-writing
description: 将十一个已确认章节组装为按读者理解路径展开的正式技术文档。用于 technical-solution-design 的文档组装 Step；不得重新推导或静默修改上游结论。
---

# 组装正式技术文档

## 产物列表

| 逻辑产物 | 承载与完整性要求 |
|---|---|
| 完整技术文档 | `technical-solution.md`；完整承载 00 至 10 章、主架构图、章节明确引用的辅助图与证据 |

开始前读取[推导标准](../technical-solution-review/references/reasoning.md)和
[范文与标准](../technical-solution-review/references/exemplars.md)。范文只用于学习结论、推导、对比和
图文组织，不是当前项目证据，也不能覆盖当前十一章契约。

读取以下已确认产物：

```text
.technical-solution/sections/00-summary.md
.technical-solution/sections/01-business-background.md
.technical-solution/sections/02-goals-and-problems.md
.technical-solution/sections/03-business-constraints.md
.technical-solution/sections/04-research.md
.technical-solution/sections/05-overall-solution.md
.technical-solution/sections/06-key-modules.md
.technical-solution/sections/07-decisions.md
.technical-solution/sections/08-delivery-and-risk.md
.technical-solution/sections/09-results.md
.technical-solution/sections/10-retrospective-and-roadmap.md
.technical-solution/architecture.mmd
```

组装 `technical-solution.md`，结构必须精确为：

```markdown
# <能表达核心结论的项目标题>
## 0. 摘要
## 1. 业务背景
## 2. 目标与问题定义
## 3. 业务特点与技术约束
## 4. 业界/业内方案调研
## 5. 总体方案设计
## 6. 关键模块设计
## 7. 核心技术决策与取舍
## 8. 落地路径与风险控制
## 9. 结果收益
## 10. 复盘与后续规划
```

允许 `###` 按项目真实内容组织，但必须归属当前 `##`；禁止 `####` 及更深标题、手工 `1.1` 式
子编号、空模板标题和附录章节。总体架构图嵌入第五章并解释边界、上下游、组件、依赖和箭头含义；
辅助图只组装章节明确引用且与正文一致的文件。

按用户用途调整表达重心，但不改变章节：

- 晋升：突出业务价值、技术深度、个人 owner 范围、影响力和规划能力；
- 分享：突出问题抽象、方案复用、关键经验、适用边界和踩坑清单；
- 融合：同时保留两类证据，仍按评委或听众的理解路径而非时间线组织。

本 Step 可以去重、调整同一章节内的呈现顺序和过渡句，但不得新建因果关系、补造选型理由、改变
事实状态、目标承诺或贡献归属。发现依据缺失或语义冲突时，按最早受影响章上报
`background_changed` 至 `summary_changed` 中的一项；仅标题、顺序或图文呈现问题留在本 Step 修复。

写入后回读验证：恰好一个项目标题、十一个规定 `##` 且顺序正确、所有章节非空、没有附录、所有
`###` 合法、关键事实有证据状态、图文链接有效。成功时上报 `technical_solution_written` 和
`technical-solution.md` 路径。
