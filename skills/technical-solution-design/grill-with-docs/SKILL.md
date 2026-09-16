---
name: grill-with-docs
description: 在技术方案每个 Step 开工前，结合已有文档追问并确认目标、约束、输入、输出和验收标准。
---

# Grill With Docs

1. 阅读当前 Step 上下文、已有 Outputs 和相关文档，不重复询问已有明确答案。
2. 使用 grilling 与 domain-modeling 的方法逐项澄清目标、边界、关键概念、输入、输出、风险和验收标准。
3. 把仍未确认的内容明确列为开放问题，不把建议自动视为决定。
4. 只有人在当前会话明确确认本 Step 范围后，提交 `step_scope_confirmed=confirmed`，并附 `source=human` 的原始确认 Evidence，选择 `start_current_step`。
5. 未确认时保持等待，不执行当前 Step 的业务 Prompt。
