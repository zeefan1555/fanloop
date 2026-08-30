---
name: technical-overall-solution
description: 基于已批准问题和公平调研确定核心选型、架构全景与关键链路。用于 technical-solution-design 的总体方案 Step；不展开局部难点实现。
---

# 设计总体方案

读取已批准的 `01-background.md` 至 `04-research.md`，按以下顺序推导：

1. 核心思路：用一段话说明方案如何化解最关键矛盾；
2. 核心选型：只选择本问题需要的业务、物理、数据、存储或交互模型，并写清取舍；
3. 架构全景：覆盖系统边界、上下游、外部系统、底层依赖、中间件、框架和关键组件；
4. 核心链路：标明主要数据流、控制流、异常流及协议、数据类型、方向或有证据的量级；
5. 问题映射：每个核心问题对应设计决策，每个关键决策可回溯到问题、目标或约束。

创建 `.technical-solution/architecture.mmd`，用可编辑 Mermaid 横向或纵向分层。节点名称与正文一致，
箭头必须有含义；未知 QPS、延迟或容量标 `TBD`，不得编造。图要完整呈现周边支撑与依赖，重点
组件用分组或样式突出。

将核心思路、选型对比结论、架构图引用、关键链路和问题映射写入并回读
`.technical-solution/sections/05-overall-solution.md`。片段不得出现 Markdown 标题，不展开各难点的
实现步骤。

若发现已批准的背景、问题、目标或调研需要改变，只选择最早层级上报对应 feedback Condition。
否则同时上报 `overall_solution_designed`、片段路径、`architecture_diagram_written` 和图路径。
