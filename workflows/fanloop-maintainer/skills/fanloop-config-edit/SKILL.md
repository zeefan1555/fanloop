---
name: fanloop-config-edit
description: 按 Define 已确认范围修改 Workflow YAML、Skill 指令和参考材料；仅在 Maintainer Build 的配置 Route 使用。
---

# 修改配置

读取 `configuration_scope` 与相关 live/仓库文件，只修改允许范围。优先复用已有命令和绑定，移除冲突或废弃的操作指引；保持未批准变更的 Step id、名称、顺序与 executor 不变。

修改 YAML 时同步必要的 Condition、Route、Prompt 和 Skill 引用。编写 Skill 时按职责分开，具体操作只保留一份真值，调用处保留指针。

`configuration_changes_written` 记录实际文件、diff 摘要、工作区及待更新 live 目标。这里的完成是修改已落盘，不是 PR 或 live 已更新。

不修改 Go/Node/Python 实现、可执行脚本、IDL、构建、代码测试或 Golden。若目标确实需要这些改动，用 `requirements_changed` 回 Define 重新分类，不继续以配置类别推进。
