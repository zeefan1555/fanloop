---
name: technical-solution-review
description: 以陌生评委视角审校九个语义章节的因果、取舍、证据、适用边界、架构完整性和落地闭环。用于 technical-solution-design 的方案审校 Step；只写审校报告，不修改输入。
---

# 独立审校技术方案

## 产物列表

| 逻辑产物 | 承载与完整性要求 |
|---|---|
| What–Why–How 与范文对照报告 | `.technical-solution/review.md`；覆盖当前方案的重要判断，定位三问、上下游承接、证据验证及所选范文的对应位置 |
| 问题清单 | 同一 `review.md`；每项有位置、证据、影响、严重级别和最早受影响层；无问题时明确记录，不制造发现凑清单 |
| 审校结论 | 同一 `review.md`；分别给出局部闭环、全文推导、证据与验证结论，并按既有规则决定通过或最早层回流 |

三项可以由同一报告承载。先冷读完整方案及其引用图、附录，记录产物缺项，再回查 01–09 源片段。
源材料齐备而成文遗漏或呈现不清时，归为 `presentation_changed`；源片段自身缺项时，才按最早
受影响的生产步骤回流。报告路径与审校通过是两个既有事实，报告存在不代表可以通过。
核对依据为同一 Release 中[背景](../technical-background-framing/SKILL.md)、[问题](../technical-problem-analysis/SKILL.md)、
[目标](../technical-objective-setting/SKILL.md)、[调研](../technical-solution-research/SKILL.md)、[总体方案](../technical-overall-solution/SKILL.md)、
[难点](../technical-key-solutions/SKILL.md)、[收益](../technical-solution-benefits/SKILL.md)、[落地](../technical-solution-delivery/SKILL.md)、
[成文](../technical-solution-writing/SKILL.md)各自的“产物列表”，相对路径以本 Skill 目录解析。

## 执行

先读[推导标准](references/reasoning.md)，按重要判断检查 What–Why–How，并分别判断局部论证、全文推导、
证据与验证。固定三问标题、章节齐全或文字相似均不能代替这些检查。

读取[图文范文与标准](references/exemplars.md)，按导读查看完整正文和本地图片，按其中的对照表记录范文与当前文档的具体差距。
相对路径以本 `SKILL.md` 所在目录解析。范文材料或看图能力缺失时报告缺口，补齐前不提交审校 Result，
也不把材料故障当作当前方案的内容反馈。

把自己视为没有聊天上下文的首次读者。先冷读 `technical-solution.md` 和正文明确引用的图及附录，
定位每个重要判断的 What、Why、How 与上下游关系，再读 `01-background.md` 至 `09-appendix.md`、
`.technical-solution/architecture.mmd` 和引用证据核对来源。上游材料不能替最终正文补答案；不得修改这些输入。

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

按推导标准的表格记录重要判断与范文对照，并给出局部论证、全文推导、证据与验证三项独立结论，
全部写入 `.technical-solution/review.md`。每条发现包含严重级别、位置、证据、影响、建议方向和最早
受影响层。分层只允许：`background`、`problem`、`objectives`、`research`、`overall_solution`、
`key_solutions`、`benefits`、`delivery`、`presentation`、`minor`。

任何导致错误决策、无法实施或无法验证的问题都算阻塞。只按最早受影响层选择一次结果：

- 有阻塞项：上报对应的 `background_changed` 至 `presentation_changed` 中一项，并同时上报
  `technical_solution_review_written`；Evidence 写明原始发现、保留内容、失效产物和回流 Step；
- 零阻塞项：同时上报 `technical_solution_review_passed` 与
  `technical_solution_review_written`。

`minor` 单独记录但不阻塞终审。不得用多个 feedback Condition 代替“最早层级”判断。
