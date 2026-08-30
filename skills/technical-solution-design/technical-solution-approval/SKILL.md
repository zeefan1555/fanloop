---
name: technical-solution-approval
description: 对完整技术方案包执行 Agent 独立复核或最终人工确认，并将人工拒绝精确分类到问题、方向或写作层。用于 technical-solution-design 的技术方案确认 Step。
---

# 终审技术方案

读取问题定义、已批准方向、正式方案、架构图和最新审校报告，独立检查完整性、因果、取舍、风险与验证闭环。确认无阻塞项且无需人作出新决定时，直接上报 `agent_approved=approved` 并结束，Evidence 记录复核理由和五个文件路径，不发布 Panorama。

不能独立批准时，再提供无需聊天历史的人工终审摘要：

- 要解决的问题、目标与非目标；
- 推荐方向和关键取舍；
- 总体架构、关键模块和主要风险；
- 验证、迁移、发布与回滚安排；
- 审校结论及仍保留的非阻塞项；
- 明确选择：批准，或拒绝并指出变化层级。

把上述内容组成一份自包含审核材料，通过当前 Agent 渠道展示。另按
`panorama_card_published` 绑定 Skill 原样展示 renderer 生成的 Panorama，不把审核正文二次拼入
Panorama。两者展示成功后才请求人的决定。

不得把“看过”“继续”“可以讨论”等含糊表达解释为批准，也不得在本 Step 直接修改输入文件。

进入人工路径后，仅在人的回复明确后报告一条结果：

- 批准当前完整方案：同时上报 `panorama_card_published` 与 `technical_solution_approved`，流程结束；
- 问题和方向不变，只需修改正文、架构图或实现细节：同时上报 `panorama_card_published` 与 `technical_solution_rejected`；
- 核心模型、选型或总体架构方向变化：同时上报 `panorama_card_published` 与 `solution_direction_changed`；
- 问题、目标、非目标或硬约束变化：同时上报 `panorama_card_published` 与 `technical_problem_changed`；
- 含糊或尚需讨论：继续等待。

人工路径的 `panorama_card_published` 输出本次 render 的精确 `panorama_snapshot_path`。Evidence 保存人的完整原始回复，以及本轮审核的五个文件路径。
