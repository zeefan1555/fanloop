---
name: commonloop-dev-panorama
description: 在 commonloop-maintainer 的 panorama_card_published Condition 要求时生成并通过当前 Agent 通道展示自包含 Panorama，返回本次真实投递回执。
---

# 展示 Commonloop Panorama

仅当最新 `flow status` 的 `current.conditions[]` 包含 `panorama_card_published` 时执行。

1. 读取最新 `flow status`，再运行 `commonloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown`；只使用成功响应的 `data.content` 和 `data.snapshot_path`。
2. 以 `data.content` 为流程状态骨架，按当前 Human Step Prompt 读取有效 Outputs 指向的审核产物，把要求审核的正文、约束、验证与待决事项合并为一份无需聊天历史和外部链接也能理解的材料。不得把原始渲染结果或链接清单冒充完整审核材料。
3. 通过当前 Agent 通道只展示一次该材料，不检测或切换宿主，不调用 Botmux、Aiden、AIME 或其他通道。
4. 只接受当前通道真实返回的 `messageId` 或当前宿主提供的 Agent 交互 Event ID；两者都不可得时报告 blocked。不得复用前一 Step 或前一次进入本 Step 的回执。
5. 展示和回读成功后返回：

```json
{"condition_id":"panorama_card_published","output":{"type":"string","value":"<REAL_RECEIPT_ID>"}}
```

`data.snapshot_path` 只作为本次渲染证据，不代替真实投递回执。产物读取、渲染、展示或回读失败时上报 blocked，不提交 Result，不跨通道 fallback。
