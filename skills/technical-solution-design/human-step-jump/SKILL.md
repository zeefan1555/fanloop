---
name: human-step-jump
description: 在人要求跳转技术方案 Step 时，确认目标并展示跳过、失效和缺失依赖影响。
---

# Human Step Jump

1. 从当前 `available_routes` 选择人指定的 `jump_step_id`，不得猜测或拼造 Step ID。
2. 跳转前说明目标 Step、向前跳过的 Steps、将失效的目标及下游 Outputs，以及目标可能缺少的输入。
3. 等待人明确确认目标和影响。
4. 确认后只提交 `human_step_jump_requested=<jump_step_id>`，附 `source=human` 的原始确认 Evidence，并选择相同的 `jump_step_id`。
