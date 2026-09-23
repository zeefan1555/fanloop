---
name: fanloop-config-define
description: 为纯 Workflow YAML、Skill 指令和参考材料改造确认文件范围、现有能力与 live 交付目标；仅在 Maintainer Define 的配置 Route 使用。
---

# 配置范围

输入是用户目标和待修改的 Workflow/Skill。先检查现有 CLI 帮助或相关实现是否已经支持目标能力；例如 SDK 已支持 Power-up 时，只修正使用指引。

确认只涉及已有 Schema 下的 YAML、Skill 指令/参考材料和必要说明。新增可执行脚本、CLI 能力、IDL/Schema、Runtime 或测试基础设施不属于本 Skill，记录范围变化并回 Define。

将以下事实写入 `configuration_scope_recorded` 的对象：目标、允许文件、实际仓库与工作区、默认分支及基线提交、需更新的已有 live 根、目标文件修改前摘要、开始时间，以及本次用户授权来源。新文件的原摘要记为不存在。用户未明确只改 live 时，交付包含 PR 合码及 live 回读。

范围与目标清楚就结束 Define；不启动代码需求访谈，不另造 Spec、Tickets 或代码 Verification Contract。后续步骤沿用本次 `maintenance_kind=configuration` 与范围，范围变化先回 Define。
