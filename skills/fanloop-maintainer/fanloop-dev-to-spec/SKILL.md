---
name: fanloop-dev-to-spec
description: 把已确认的 Fanloop CLI requirements.md 生成为 Issue Workspace 本地实施 Spec。用于方案设计 Step；不重新访谈或修改仓库。
---

# To Spec

读取已确认的 `requirements.md` 和相关仓库事实，不再提问。在 Issue Workspace
根目录写 `spec.md`，顶层章节固定为：背景、问题、目标、解法、CLI 预期的输入与返回。

`解法` 要覆盖已确认的 User Stories、Implementation Decisions、Testing Decisions、Out of Scope
和 Further Notes，不能遗失原有实施 Spec 信息。

`解法 / Testing Decisions` 只复制 `requirements.md` 中已确认的 Test Seams，不重新选择或新增。
逐项写明相关行为采用 `TDD` 或 `direct implementation`，以及 TDD 使用的独立正确预期。必要 Seam
决策缺失或与其他已确认内容冲突时返回需求澄清，不在方案阶段自行补决策。

`CLI 预期的输入与返回` 只写本次改动涉及的交互契约，每个关键正常或回流场景都必须包含：

- 实际叶子命令和完整 `Agent → CLI Request` JSON；
- 完整 `CLI → Agent Response` JSON；
- 会影响 Agent 下一步决策的 `effect`、`transition`、最新 `state.current`、有效 `outputs`、
  `invalidated_outputs`、`meta` 和 `_notice`。

动态 event ID、绝对路径、时间和 Workflow digest 可使用明确占位符，但不得用字段摘要、伪代码
或删减决策字段的片段冒充真实返回结构。

只落盘人或 Agent 通过当前 Workflow 批准的内容；缺少或冲突的决策返回需求澄清。不要发布外部
Issue，不修改仓库，
不把本地 Spec 放进 `.scratch/`、`docs/research/` 或 `docs/specs/`。

## 飞书技术方案产物

`spec.md` 完成后，使用当前宿主的 `lark-doc` 能力以 Markdown 发布技术方案：

1. 使用包含 Requirement 标题的稳定文档标题。同一 Requirement 始终更新同一份飞书文档。
2. 首次发布前按稳定标题查找：唯一命中则更新，零命中才创建，多命中则停止。创建结果不确定时先查找，禁止直接重复创建。
3. 创建或更新后使用返回 URL 回读文档，确认回读正文非空且五个顶层章节完整。
4. 成功时向当前 Step 返回：

```text
condition_id=spec_written
spec_path=spec.md
condition_id=technical_solution_document_published
technical_design_document_url=<已回读验证的飞书文档 URL>
```

飞书能力不可用、标题多命中、创建/更新失败或回读正文为空时返回
`progress_status=blocked` 和真实原因；不返回任何成功 Condition。
