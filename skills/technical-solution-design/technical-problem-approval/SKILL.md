---
name: technical-problem-approval
description: 对技术问题定义执行人工确认并留下完整审核证据。用于 technical-solution-design 的问题人工确认 Step；不得代替人批准，也不得在本 Step 修改问题定义。
---

# 人工确认技术问题

本 Step 的决定只能由人作出。先读取 `.technical-solution/problem.md`，再提供无需聊天历史也能理解的审核摘要：

- 要解决的 2–3 个核心问题；
- 支撑每个问题的关键事实与来源；
- 目标、非目标、约束和成功指标；
- 仍存在的开放问题及其是否阻塞后续推导；
- 明确选择：批准，或拒绝并说明必须修改的内容。

检查问题与目标是否具体、事实是否可验证、因果是否成立、范围是否完整且没有混入方案结论。发现问题时只指出，不直接改写已提交材料。

仅在收到人的明确回复后报告结果：

- 明确批准：上报 `technical_problem_approved`；
- 拒绝或要求修改：上报 `technical_problem_rejected`，回到问题定义；
- 含糊、沉默或仅提供补充信息：继续等待，不推断批准。

Evidence 保存人的完整原始回复以及对应的 `.technical-solution/problem.md` 路径。
