---
name: technical-problem-approval
description: 对技术问题定义执行 Agent 独立复核或人工确认并留下完整审核证据。用于 technical-solution-design 的问题确认 Step；不得在本 Step 修改问题定义。
---

# 确认技术问题

先读取 `.technical-solution/problem.md` 并独立检查问题与目标是否具体、事实是否可验证、因果是否成立、范围是否完整且没有混入方案结论。确认无阻塞项且无需人作出新决定时，直接上报 `agent_approved=approved`，Evidence 记录复核理由和文件路径，不发布 Panorama。

不能独立批准时，再提供无需聊天历史也能理解的人工审核摘要：

- 要解决的 2–3 个核心问题；
- 支撑每个问题的关键事实与来源；
- 目标、非目标、约束和成功指标；
- 仍存在的开放问题及其是否阻塞后续推导；
- 明确选择：批准，或拒绝并说明必须修改的内容。

把上述内容组成一份自包含审核材料，通过当前 Agent 渠道展示。另按
`panorama_card_published` 绑定 Skill 原样展示 renderer 生成的 Panorama，不把审核正文二次拼入
Panorama。两者展示成功后才请求人的决定。

发现问题时只指出，不直接改写已提交材料。

进入人工路径后，仅在收到人的明确回复后报告结果：

- 明确批准：同时上报 `panorama_card_published` 与 `technical_problem_approved`；
- 拒绝或要求修改：同时上报 `panorama_card_published` 与 `technical_problem_rejected`，回到问题定义；
- 含糊、沉默或仅提供补充信息：继续等待，不推断批准。

人工路径的 `panorama_card_published` 输出本次 render 的精确 `panorama_snapshot_path`。Evidence 保存人的完整原始回复以及对应的 `.technical-solution/problem.md` 路径。
