---
name: example-analyze-materials
description: 在 example Workflow 的理解材料 Step 中，把用户材料整理成可供后续产出的最小事实摘要。
---

# 理解材料

读取用户给出的文件、链接和对话，只记录已经确认的事实。把结果写到 Requirement Root 下的
`.example/material-summary.md`，包含：

- 目标
- 输入材料
- 约束与责任边界
- 预期输出和完成标准
- 开放问题

非阻塞问题保留在开放问题中；缺少会改变 Step、责任人或输出的材料时，说明具体缺口并返回
`materials_incomplete`，不猜测答案。摘要完整时返回 `materials_analyzed` 和实际文件路径。
