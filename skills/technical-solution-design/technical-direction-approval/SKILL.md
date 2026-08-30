---
name: technical-direction-approval
description: 对候选方案比较、关键取舍和总体架构方向执行 Agent 独立复核或人工确认。用于 technical-solution-design 的方案方向确认 Step。
---

# 确认方案方向

读取 `.technical-solution/problem.md` 和 `.technical-solution/proposal.md`，独立确认比较是否公平、推荐结论是否由事实推导、关键问题是否都有设计响应、风险是否透明。确认无阻塞项且无需人作出新决定时，直接上报 `agent_approved=approved`，Evidence 记录复核理由和文件路径，不发布 Panorama。

不能独立批准时，再提供自包含人工审核摘要：

- 已批准问题、目标和硬约束；
- 统一评价维度；
- 候选方案及同维度优缺点；
- 推荐方向、关键取舍和未决风险；
- 总体架构与问题—设计映射；
- 明确选择：批准方向，或拒绝并说明原因。

把上述内容连同最新 `flow status` 的完整 Stage/Job/Step 全景、有效 Outputs 和待决事项组成一份自包含审核材料，通过当前 Agent 渠道发送并回读成功。记录本次发送返回的真实 messageId 或 Agent 交互事件 ID；不得复用前一 Step 或前一次进入本 Step 的回执。材料发送成功后才请求人的决定。

不得在本 Step 擅自增加方案决策或改写上游文件。

进入人工路径后，仅在人的回复明确后报告一条结果：

- 批准当前方向：同时上报 `panorama_card_published` 与 `solution_direction_approved`；
- 问题不变但方向需重做：同时上报 `panorama_card_published` 与 `solution_direction_rejected`；
- 反馈改变问题、目标、非目标或硬约束：同时上报 `panorama_card_published` 与 `technical_problem_changed`；
- 含糊或尚需讨论：继续等待。

人工路径的 `panorama_card_published` 输出本轮真实发送回执。Evidence 保存人的完整原始回复和被审核的两个文件路径。
