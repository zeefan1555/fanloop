---
name: technical-objective-setting
description: 把已确认的背景与核心问题转化为可验证指标、设计约束和非目标。用于 technical-solution-design 的设计目标 Step；不调研或选择方案。
---

# 定义设计目标

读取 `01-background.md` 与 `02-problem.md`，只定义方案必须达到的结果边界：

1. 核心指标：每项写基线、目标值、统计口径、观测窗口和验证方式；
2. 设计约束：一致性、安全、资损、兼容边界、资源、工期、依赖和组织约束；
3. 非目标：本次明确不解决的事项及原因；
4. 取舍顺序：指标冲突时谁优先，以及允许牺牲到什么边界。

目标必须逐项回扣核心问题，避免“提升性能”“增强可用性”等不可验收表达。无法给出数值时使用
可判定的行为标准并注明待补数据，不编造指标。

将结果写入并回读 `.technical-solution/sections/03-objectives.md`。片段不得出现 Markdown 标题，
使用目标表、约束列表和非目标列表；不得出现候选方案、技术选型或架构图。

若发现背景或问题需要改变，只选择最早层级上报 `background_changed` 或 `problem_changed`，Evidence
写明反馈原文或新事实、保留内容、失效产物与回流 Step。否则上报 `design_objectives_defined` 和
片段路径。
