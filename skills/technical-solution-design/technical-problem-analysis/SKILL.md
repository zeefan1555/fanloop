---
name: technical-problem-analysis
description: 从已确认背景中的可验证事实推导现状瓶颈、根因和关键取舍。用于 technical-solution-design 的核心问题 Step；不定义目标或设计方案。
---

# 分析核心问题

开始前读取 [What–Why–How 推导标准](../technical-solution-review/references/reasoning.md)。

读取 `.technical-solution/sections/01-background.md`，围绕同一业务边界完成：

1. 现状评估：已有方案的优势、适用条件和已出现的事实差距；
2. 常见方案：说明现状和常见做法为何不足，并保留其适用条件；
3. 瓶颈定位：通过指标、调用链、Profile、代码或配置定位主要消耗和失败点；
4. 根因分析：解释机制，不把现象改写成原因，并提炼多个现象背后的共同技术特点；
5. 关键取舍：指出延迟、可用性、一致性、成本、复用性等指标间真正存在的冲突。

按事实差距和独立根因收敛核心问题，同源现象合并。每个问题明确具体判断、可追溯证据、证据为何
支持判断，以及现有机制如何形成差距。优先写可量化差距；没有数据时写明缺口和验证办法。
不要为了突出新方案而贬低旧系统。

将结果写入并回读 `.technical-solution/sections/02-problem.md`。片段不得出现 `#` 或 `##`；允许按
真实内容使用 `###`，但禁止 `####` 及手工 `1.1` 式标题。使用加粗的问题结论、证据表和因果链
表达；不得写目标值、候选方案或解决手段。

若调查改变了背景语义，停止并上报 `background_changed`，Evidence 写明变化、保留内容、失效产物
和回流 Step。否则完成时上报 `core_problem_defined` 和片段路径。
