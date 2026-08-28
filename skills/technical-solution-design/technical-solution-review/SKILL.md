---
name: technical-solution-review
description: 以陌生评委视角独立审校技术方案的因果、取舍、架构完整性和可实施性。用于 technical-solution-design 的技术方案审校 Step；只写审校报告，不直接修改方案。
---

# 独立审校技术方案

把自己视为没有聊天上下文的首次读者。只读取：

- `.technical-solution/problem.md`；
- `.technical-solution/proposal.md`；
- `technical-solution.md`；
- `.technical-solution/architecture.mmd`。

不得直接修改这些输入。逐项检查：

- 标题和开头是否快速给出具体结论；
- 每个核心问题是否有事实、原因和对应设计；
- 候选方案是否使用相同维度，优缺点和取舍是否公平；
- 架构图是否完整覆盖边界、上下游、组件、依赖和主要数据/控制流；
- 图文、接口、数据模型、容量数字和术语是否一致；
- 异常、降级、安全、观测、迁移、发布、回滚和验证是否足以指导实施；
- 业务、技术和可复用价值是否回扣目标且有证据；
- 是否存在编造数据、跳过推导、隐藏风险或只能依赖聊天理解的内容。

## 审校产物

写入 `.technical-solution/review.md`。每条发现包含严重级别、位置、证据、影响、建议方向和回流层级：

- `problem`：问题、目标、非目标或约束本身发生变化；
- `direction`：核心模型、选型或总体架构方向需改变；
- `writing`：已批准方向不变，只需修正文档、图或实现细节；
- `minor`：不阻塞人工终审的优化项。

问题导致方案无法正确决策、实施或验证时视为阻塞。按最高层级只选择一条 Route：

- `problem`：上报 `technical_problem_changed`；
- `direction`：上报 `solution_direction_changed`；
- 仅有 `writing` 阻塞：上报 `technical_solution_review_failed`、`technical_solution_review_written` 和报告路径；
- 无阻塞：上报 `technical_solution_review_passed`、`technical_solution_review_written` 和报告路径。
