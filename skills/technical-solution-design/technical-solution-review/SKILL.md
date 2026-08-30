---
name: technical-solution-review
description: 以陌生评委视角审校九段式技术方案的因果、取舍、架构完整性和落地闭环。用于 technical-solution-design 的方案审校 Step；只写审校报告，不修改输入。
---

# 独立审校技术方案

把自己视为没有聊天上下文的首次读者。读取 `01-background.md` 至 `09-appendix.md`、
`technical-solution.md` 和 `.technical-solution/architecture.mmd`，不得修改这些输入。

逐项检查：

- 正文是否只有九个规定标题，没有 `###` 或 `1.1`，且三分钟内能进入总体方案；
- 背景、问题、目标、调研、总体方案、难点、收益和落地是否层层推导；
- 每个核心问题是否有事实、根因、目标、设计手段和验证闭环；
- 内部、业界、不改基线和推荐方案是否使用同一维度并公平披露优缺点；
- 架构图是否完整覆盖边界、上下游、组件、依赖、关键链路和有含义的箭头；
- 图文、接口、模型、容量数字、指标口径和术语是否一致；
- 异常、降级、安全、观测、迁移、发布、回滚和验证是否足以落地；
- 价值是否回扣目标且区分已测、预期和待验证，是否存在编造、跳步或隐藏风险。

将发现写入 `.technical-solution/review.md`。每条包含严重级别、位置、证据、影响、建议方向和最早
受影响层。分层只允许：`background`、`problem`、`objectives`、`research`、`overall_solution`、
`key_solutions`、`benefits`、`delivery`、`presentation`、`minor`。

任何导致错误决策、无法实施或无法验证的问题都算阻塞。只按最早受影响层选择一次结果：

- 有阻塞项：上报对应的 `background_changed` 至 `presentation_changed` 中一项，并同时上报
  `technical_solution_review_written`；Evidence 写明原始发现、保留内容、失效产物和回流 Step；
- 零阻塞项：同时上报 `technical_solution_review_passed` 与
  `technical_solution_review_written`。

`minor` 单独记录但不阻塞终审。不得用多个 feedback Condition 代替“最早层级”判断。
