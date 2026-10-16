---
name: manage-fanloop-workflow
description: 管理 Fanloop Workflow 配置的四节点 Workflow Bundle，覆盖分析、构建、人工审核、个人 Fornax 发布与本地同步。
---

# manage-fanloop-workflow

本包是 loop_cli `manage-loop-workflow` 在 Fanloop 中的对应 Workflow。四阶段保持原职责，绑定六个 Skill（含 workflow-design-principles）。全景图由 Runtime `pending_actions` 投递，本包不声明全景图 Condition。发布只写入个人 Fornax 空间 `7543941556223377409`，key 前缀为 `fanloop.workflow.`。
