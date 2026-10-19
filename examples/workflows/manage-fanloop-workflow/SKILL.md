---
name: manage-fanloop-workflow
description: 管理 Fanloop Workflow 配置的四节点 Workflow Bundle，覆盖分析、构建、人工审核、个人 Fornax 发布与本地同步。
---

# manage-fanloop-workflow

本包是 loop_cli `manage-loop-workflow` 在 Fanloop 中的对应 Workflow。四阶段与原有五个管理 Skill 保持原职责，并通过普通 `panorama_presented` Condition、Route operand 和包内 `manage-fanloop-panorama` Skill 明确选择全景图能力。Runtime 不注入全景图节点。发布只写入个人 Fornax 空间 `7543941556223377409`，key 前缀为 `fanloop.workflow.`。
