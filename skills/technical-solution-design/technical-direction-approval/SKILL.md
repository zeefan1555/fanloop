---
name: technical-direction-approval
description: 对候选方案比较、关键取舍和总体架构方向执行人工确认。用于 technical-solution-design 的方案方向人工确认 Step；只记录人的结论并选择正确回流层级。
---

# 人工确认方案方向

本 Step 的方向决定只能由人作出。读取 `.technical-solution/problem.md` 和 `.technical-solution/proposal.md`，提供自包含审核摘要：

- 已批准问题、目标和硬约束；
- 统一评价维度；
- 候选方案及同维度优缺点；
- 推荐方向、关键取舍和未决风险；
- 总体架构与问题—设计映射；
- 明确选择：批准方向，或拒绝并说明原因。

把上述内容连同最新 `flow status` 的完整 Stage/Job/Step 全景、有效 Outputs 和待决事项组成一份自包含审核材料，通过当前 Agent 渠道发送并回读成功。记录本次发送返回的真实 messageId 或 Agent 交互事件 ID；不得复用前一 Step 或前一次进入本 Step 的回执。材料发送成功后才请求人的决定。

确认比较是否公平、推荐结论是否由事实推导、关键问题是否都有设计响应、风险是否透明。不得在本 Step 擅自增加方案决策或改写上游文件。

仅在人的回复明确后报告一条结果：

- 批准当前方向：同时上报 `panorama_card_published` 与 `solution_direction_approved`；
- 问题不变但方向需重做：同时上报 `panorama_card_published` 与 `solution_direction_rejected`；
- 反馈改变问题、目标、非目标或硬约束：同时上报 `panorama_card_published` 与 `technical_problem_changed`；
- 含糊或尚需讨论：继续等待。

`panorama_card_published` 输出本轮真实发送回执。Evidence 保存人的完整原始回复和被审核的两个文件路径。
