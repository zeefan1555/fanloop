---
name: fanloop-config-edit
description: 按 Define 已确认范围修改 Workflow YAML、Skill 指令和参考材料；仅在 Maintainer Build 的配置 Route 使用。
---

# 修改配置

读取 `configuration_scope` 和本地源码工作区，仅修改已确认目标所需文件。同一目标补充关联说明时，在 delivery-record.md 追加原因、实际文件与修改前摘要即可，不机械回 Define；目标、验收、权限、副作用或类别扩大才回流。优先复用已有命令和绑定；Step id、名称、顺序与 executor 的实际差异必须记录并自动校验，不设 YAML 专属人工审批。

修改 YAML 时同步必要的 Condition、Route、Prompt 和 Skill 引用。编写 Skill 时按职责分开，具体操作只保留一份真值，调用处保留指针。

修改后立即做最小检查：diff、引用，命令改变时查对应 help；普通模板按 trial_plan 用真实代表性材料自检生成结果与关键边界，不新建临时测试框架。将结果和原始证据引用追加到同一 delivery-record.md，Certify 核验这些结果，不重复运行未失效的检查。

`configuration_changes_written` 记录实际文件、diff 摘要、候选文件摘要、工作区、本地配置源和记录路径。落盘不等于已提交或验收通过。保持引用在完整包内；不能创建第二份隐含配置真值。

不修改 Go/Node/Python 实现、可执行脚本、IDL、构建、代码测试或 Golden。若目标确实需要这些改动，用 `requirements_changed` 回 Define 重新分类，不继续以配置类别推进。
