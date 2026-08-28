---
name: technical-solution-writing
description: 将已批准的问题和方案方向写成可实施、可评审的正式技术方案及总体架构图。用于 technical-solution-design 的技术方案写作 Step；不得引入未经批准的新事实或方向。
---

# 写作正式技术方案

读取已批准的 `.technical-solution/problem.md` 和 `.technical-solution/proposal.md`。先搭建结论先行、总分总的结构，再补足实现所需细节；不重复推翻已批准结论。

## 架构图

创建 `.technical-solution/architecture.mmd`，使用可编辑的 Mermaid 表达完整总体架构。图中必须包含：

- 系统边界、上游、下游和外部依赖；
- 分层及关键组件；
- 主要数据流和控制流；
- 有意义的连线标签，例如协议、数据类型、方向或有证据的量级。

没有可靠数据时标 `TBD`，不得编造 QPS、延迟或容量。将该图嵌入并在正文中解释，确保图和文字使用相同组件名。

## 正文

写入 `technical-solution.md`。标题表达具体结论，开头在短篇幅内给出背景、冲突、问题和答案。正文至少覆盖：

1. 背景、业务特点、当前方案和核心问题；
2. 目标、非目标、约束和成功指标；
3. 候选方案、评价维度、推荐结论与取舍；
4. 总体架构及关键链路；
5. 关键模块的职责、接口、数据模型、依赖和交互；
6. 容量、性能、可用性、一致性、异常与降级；
7. 安全、权限、隐私和合规；
8. 可观测性、验证方案、迁移、发布、回滚和里程碑；
9. 业务价值、技术价值、可复用价值；
10. 风险、开放问题和附录。

不适用的部分用一句话说明原因，不制造空洞章节。区分已验证事实、估算和待确认项；所有关键决策可回溯到已批准问题。

写入后回读两个文件，检查链接、名称和结论一致。成功时同时上报 `technical_solution_written`、`technical_solution_path`、`architecture_diagram_written` 和 `architecture_diagram_path`。问题基线变化时上报 `technical_problem_changed`；方案方向变化时上报 `solution_direction_changed`。
