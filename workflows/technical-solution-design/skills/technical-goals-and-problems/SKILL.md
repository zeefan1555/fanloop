---
name: technical-goals-and-problems
description: 从业务背景收敛不超过三个核心问题，并定义逐项对应的可验证目标。用于 technical-solution-design 的目标与问题定义 Step；不调研或选择方案。
---

# 定义目标与问题

读取 `.technical-solution/sections/01-business-background.md` 和
[What-Why-How 推导标准](../technical-solution-review/references/reasoning.md)。围绕同一业务边界完成：

1. 合并同源现象，只保留 2 至 3 个相互独立的核心问题；
2. 每个问题说明现状、影响、根因证据和为什么现在必须解决；
3. 为每个问题定义基线、目标、统计口径、观测窗口和验证方式；
4. 明确非目标，不把实现方式写成目标。

事实、推断、目标和待确认信息必须分开。没有可靠数字时使用可判定的行为标准并注明取证办法，
不得编造指标或为了突出新方案贬低旧系统。

## 产物列表

以下逻辑产物均写入 `.technical-solution/sections/02-goals-and-problems.md`：

| 逻辑产物 | 完整性要求 |
| --- | --- |
| 核心问题 | 只保留 2 至 3 个问题；每项包含现状、影响、根因证据和紧迫性。 |
| 目标与验收 | 每个问题都有对应目标、基线、口径、窗口和通过标准。 |
| 非目标 | 明确本次不解决的事项及理由。 |

片段不得出现 `#` 或 `##`；允许按真实内容使用 `###`，禁止 `####` 和手工 `1.1` 编号。
发现背景语义变化时上报 `background_changed`；否则逐项回读通过后上报
`goals_and_problems_defined` 和片段路径。
