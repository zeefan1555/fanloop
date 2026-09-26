---
name: example-draft-result
description: 在 example Workflow 的形成草案 Step 中，只依据已确认的材料摘要生成可审核结果。
---

# 形成草案

读取 `.example/material-summary.md`，把结果写到 `.example/result.md`。结果应直接满足摘要中的目标和
完成标准，并让每个关键判断能回到输入材料或明确的人类决定。

发现输入事实已经变化时返回 `materials_changed`，回到理解材料。材料未变但草案仍不满足完成标准时，
返回 `draft_needs_revision` 并说明具体原因。草案可供审核时返回 `draft_written` 和实际文件路径。
