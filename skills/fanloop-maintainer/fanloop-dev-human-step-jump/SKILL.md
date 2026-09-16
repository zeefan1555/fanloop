---
name: fanloop-dev-human-step-jump
description: 在 fanloop-maintainer 任意运行中 Step 按 human 明确决定安全跳转，不伪造被跨过的完成事实。
---

# Human Step Jump

只在最新 Status 提供 `human_step_jump_requested`，且 human 明确指定以下九个 Step 之一时执行：

1. `bootstrap_techdesign`
2. `clarify_requirements`
3. `design_technical_solution`
4. `confirm_technical_solution`
5. `implement_code`
6. `review_code`
7. `execute_agent_acceptance`
8. `confirm_human_acceptance`
9. `handoff_merge_request`

目标严格晚于 source 时选 Flow，等于或早于 source 时选 Loop。目标含糊、未知、不唯一或未出现在 `available_routes` 时继续澄清，不写文件、不提交 Result。

读取当前 Outputs、Event 历史、Issue Workspace 报告和已登记飞书 URL，只复用与当前 branch、review_base、reviewed_head 一致的事实。按目标 Step 列出已复用、人工补充、仍缺失、前向跨过或回流失效的内容。

展示 source、target、方向、产物清单和影响后，要求 human 明确确认同一 target。用 `fanloop-dev-decision-receipt` 以 `decision_class=step_jump` 记录原始决定和不可变消息或 turn 引用。将完整记录追加到 Issue Workspace 的 `.scratch/human-step-jump.md`，并在 Requirement Root 写入 `.scratch/human-step-jump-context.md` 指向本次段落。

最终只提交 `human_step_jump_requested=jump_requested`、`human_step_jump_context_written`、`human_step_jump_recorded` 和已展示的 `panorama_presented`。不混入普通完成、通过、失败或跳过 Condition。
