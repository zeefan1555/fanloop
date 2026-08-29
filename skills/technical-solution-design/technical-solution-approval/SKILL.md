---
name: technical-solution-approval
description: 对完整技术方案包执行最终人工确认，并将拒绝精确分类到问题、方向或写作层。用于 technical-solution-design 的技术方案人工确认 Step；不得代替人作终审决定。
---

# 人工终审技术方案

本 Step 的最终决定只能由人作出。读取问题定义、已批准方向、正式方案、架构图和最新审校报告，提供无需聊天历史的终审摘要：

- 要解决的问题、目标与非目标；
- 推荐方向和关键取舍；
- 总体架构、关键模块和主要风险；
- 验证、迁移、发布与回滚安排；
- 审校结论及仍保留的非阻塞项；
- 明确选择：批准，或拒绝并指出变化层级。

把上述内容连同最新 `flow status` 的完整 Stage/Job/Step 全景、有效 Outputs 和待决事项组成一份自包含审核材料，通过当前 Agent 渠道发送并回读成功。记录本次发送返回的真实 messageId 或 Agent 交互事件 ID；不得复用前一 Step 或前一次进入本 Step 的回执。材料发送成功后才请求人的决定。

不得把“看过”“继续”“可以讨论”等含糊表达解释为批准，也不得在本 Step 直接修改输入文件。

仅在人的回复明确后报告一条结果：

- 批准当前完整方案：同时上报 `panorama_card_published` 与 `technical_solution_approved`，流程结束；
- 问题和方向不变，只需修改正文、架构图或实现细节：同时上报 `panorama_card_published` 与 `technical_solution_rejected`；
- 核心模型、选型或总体架构方向变化：同时上报 `panorama_card_published` 与 `solution_direction_changed`；
- 问题、目标、非目标或硬约束变化：同时上报 `panorama_card_published` 与 `technical_problem_changed`；
- 含糊或尚需讨论：继续等待。

`panorama_card_published` 输出本轮真实发送回执。Evidence 保存人的完整原始回复，以及本轮审核的五个文件路径。
