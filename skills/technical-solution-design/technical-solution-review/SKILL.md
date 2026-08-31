---
name: technical-solution-review
description: 以陌生评委视角审校九个语义章节的因果、取舍、证据、适用边界、架构完整性和落地闭环。用于 technical-solution-design 的方案审校 Step；只写审校报告，不修改输入。
---

# 独立审校技术方案

把自己视为没有聊天上下文的首次读者。读取 `01-background.md` 至 `09-appendix.md`、
`technical-solution.md`、`.technical-solution/architecture.mmd` 和正文明确引用的辅助图，不得修改
这些输入。

逐项检查：

- 正文是否保留九个规定的 `##` 语义章节；允许 `###` 按真实内容组织，但不能出现 `####` 或手工
  `1.1`，且三分钟内能进入总体方案；
- 背景、问题、目标、调研、总体方案、难点、收益和落地是否层层推导；
- 每个核心问题是否有事实、根因、目标、设计手段和验证闭环；
- 内部、业界、不改基线和推荐方案是否使用同一维度并公平披露优缺点；
- 架构图是否完整覆盖边界、上下游、组件、依赖、关键链路和有含义的箭头；
- 图文、接口、模型、容量数字、指标口径、证据状态和术语是否一致，事实是否有可追溯来源；
- 接口、数据与存储、状态与并发、性能与容量、稳定性与观测、安全与合规、兼容与迁移是否按适用性
  展开，并对不适用项给出事实理由；
- 异常、降级、恢复、观测、迁移、发布、回滚和验证是否足以落地；
- 价值是否回扣目标，是否明确适用边界、方案代价，并区分已测、估算、目标和待确认，是否存在
  编造、跳步或隐藏风险。

将发现写入 `.technical-solution/review.md`。每条包含严重级别、位置、证据、影响、建议方向和最早
受影响层。分层只允许：`background`、`problem`、`objectives`、`research`、`overall_solution`、
`key_solutions`、`benefits`、`delivery`、`presentation`、`minor`。

任何导致错误决策、无法实施或无法验证的问题都算阻塞。只按最早受影响层选择一次结果：

- 有阻塞项：上报对应的 `background_changed` 至 `presentation_changed` 中一项，并同时上报
  `technical_solution_review_written`；Evidence 写明原始发现、保留内容、失效产物和回流 Step；
- 零阻塞项：同时上报 `technical_solution_review_passed` 与
  `technical_solution_review_written`。

`minor` 单独记录但不阻塞终审。不得用多个 feedback Condition 代替“最早层级”判断。
