---
name: technical-solution-review
description: 以陌生评委或听众视角审校十一章技术文档的业务价值、技术判断、取舍、证据、贡献、复用边界与规划。用于 technical-solution-design 的文档审校 Step；只写审校报告，不修改输入。
---

# 独立审校技术文档

## 产物列表

| 逻辑产物 | 承载与完整性要求 |
|---|---|
| 审校报告 | `.technical-solution/review.md`；包含章节完整性、推导、证据、贡献边界、复用边界和最终结论 |
| 问题清单 | 同一报告；每项有位置、证据、影响、严重级别和最早受影响章 |

先冷读 `technical-solution.md`、主架构图和正文引用的辅助图，再核对 00 至 10 章源片段。上游材料
不能替最终正文补答案，不得修改任何输入。随后读取[推导标准](references/reasoning.md)和
[范文与标准](references/exemplars.md)进行交叉检查。

逐项检查：

- 不懂业务的读者是否能在五分钟内理解背景、核心问题、方案和结果；
- 是否讲清楚为什么值得做、为什么现在做，而不只是列做了什么；
- 问题是否限制为 2 至 3 个，并与目标、约束、方案、结果逐项闭环；
- 调研是否只比较 2 至 3 类真实方案，维度一致且说明不能直接照搬的原因；
- 总体方案和 2 至 3 个关键模块是否有清晰边界、机制、权衡、异常和选择理由；
- 技术决策是否列出真实备选、未选原因、代价和重评条件；
- 落地是否覆盖里程碑、灰度、风险、协作、回滚和踩坑；
- 结果是否回扣目标并区分已测、估算、目标和待确认；
- 晋升内容是否区分个人与团队贡献，分享内容是否给出复用方式和适用边界；
- 复盘是否包含正确判断、弯路和有里程碑与风险的后续规划；
- 正文是否恰好包含第 0 至第 10 章，摘要为 5 至 8 行，允许 `###` 按真实内容组织，但没有附录、
  `####` 或手工 `1.1` 标题；
- 图文、接口、模型、指标口径、证据状态和术语是否一致。

每条发现写明严重级别、位置、证据、影响、建议方向和最早受影响层。分层只允许：
`background`、`goals_and_problems`、`business_constraints`、`research`、`overall_solution`、
`key_modules`、`technical_decisions`、`delivery`、`results`、`retrospective`、`summary`、
`presentation`、`minor`。

任何导致错误决策、无法实施、无法验证或无法支撑目标读者判断的问题都算阻塞：

- 有阻塞项：上报对应的一项 feedback Condition，并同时上报 `technical_solution_review_written`；
- 零阻塞项：同时上报 `technical_solution_review_passed` 与 `technical_solution_review_written`。

只选择最早受影响层。Evidence 写明原始发现、保留内容、失效产物和回流 Step；`minor` 单独记录但
不阻塞终审。
