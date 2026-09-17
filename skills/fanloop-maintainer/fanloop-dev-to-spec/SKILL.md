---
name: fanloop-dev-to-spec
description: 把已确认的 Fanloop CLI requirements.md 生成为 Issue Workspace 本地实施 Spec。用于复杂 Build；不重新访谈或修改仓库。
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

完成后回读 `spec.md`，向 `build_until_verified` 返回本地路径和摘要；它只是辅助工件，不单独提交
Condition，也不要求发布飞书文档。
