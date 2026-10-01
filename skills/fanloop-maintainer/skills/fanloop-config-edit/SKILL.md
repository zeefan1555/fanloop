---
name: fanloop-config-edit
description: 按 Define 已确认范围修改 Workflow YAML、Skill 指令和参考材料；仅在 Maintainer Build 的配置 Route 使用。
---

# 修改配置

读取 `configuration_scope` 与从已确认 Git 基线准备的本地源码工作区文件，只修改允许范围。优先复用已有命令和绑定，移除冲突或废弃的操作指引；Step id、名称、顺序与 executor 的实际差异必须记录并纳入自动校验，不设 YAML 专属人工审批。

修改 YAML 时同步必要的 Condition、Route、Prompt 和 Skill 引用。编写 Skill 时按职责分开，具体操作只保留一份真值，调用处保留指针。

`configuration_changes_written` 记录实际文件、diff 摘要、工作区及实际本地配置源。这里的完成是本地候选已落盘，不代表已提交或通过验收。保持相对引用在 skills/<workflow-id>/ 完整包内；不能创建第二份隐含配置真值。

不修改 Go/Node/Python 实现、可执行脚本、IDL、构建、代码测试或 Golden。若目标确实需要这些改动，用 `requirements_changed` 回 Define 重新分类，不继续以配置类别推进。
