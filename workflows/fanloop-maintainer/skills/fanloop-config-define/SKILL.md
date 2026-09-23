---
name: fanloop-config-define
description: 为纯 Workflow YAML、Skill 指令和参考材料改造确认文件范围、现有能力与 live 交付目标；仅在 Maintainer Define 的配置 Route 使用。
---

# 配置范围

输入是用户目标和待修改的 Workflow/Skill。先检查现有 CLI 帮助或相关实现是否已经支持目标能力；例如 SDK 已支持 Power-up 时，只修正使用指引。

确认只涉及已有 Schema 下的 YAML、Skill 指令/参考材料和必要说明。新增可执行脚本、CLI 能力、IDL/Schema、Runtime 或测试基础设施不属于本 Skill，记录范围变化并回 Define。

将以下事实写入 `configuration_scope_recorded` 的对象：目标、允许文件、实际仓库与工作区、默认分支及基线提交、需更新的已有 live 根、目标文件修改前摘要、开始时间，以及本次用户授权来源。新文件的原摘要记为不存在。用户未明确只改 live 时，交付包含 PR 合码及 live 回读。

同时记录 `trial_plan`：是否改变执行行为及理由、需要试用时的 1–2 个最小场景。每个场景给出自然语言任务、可观察的成功标准、测试数据/目录、允许副作用和停止位置，不提前指定实现动作或正确路由答案。覆盖本次改变的成功行为及有必要验证的失败边界，不扩展成全产品回归。

改变执行规则、命令、路由、绑定、判断或完成标准时必须试用。只有全部 diff 都是错别字、排版等不影响行为的修正，才可记录 `required=false` 和具体理由；不因文件是 Markdown 就免测。用户未授权真实业务写入时使用临时仓库、隔离配置和专用测试对象；测试停在真实 PR/合码/全局 live 发布前。

范围与目标清楚就结束 Define；不启动代码需求访谈，不另造 Spec、Tickets 或代码 Verification Contract。后续步骤沿用本次 `maintenance_kind=configuration` 与范围，范围变化先回 Define。
