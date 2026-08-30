---
status: accepted
date: 2026-08-24
amends: ADR-0035, ADR-0066
amended_by: ADR-0090, ADR-0091
---

# Maintainer 在同一 Botmux Session 复用唯一 Requirement Root

Release-bound `fanloop-dev-workflow` 在为新维护请求生成 issue slug、执行 `fanloop update`
或 `flow init` 之前，先读取当前 `BOTMUX_SESSION_ID`，并在
`~/fanloop/issues/*/.fanloop/card/config.json` 中精确匹配已持久化的 `session_id`。

- 唯一命中：配置文件所在目录是唯一 Requirement Root，立即运行 `flow status`；不更新
  Release、不重新选择 Workflow、不初始化新目录。运行中 Requirement 的后续反馈继续更新同一
  `requirements.md`，必要时使用现有 `requirements_changed` Route 回流。
- 零命中：才允许生成新 Root，并沿 ADR-0053 保留的新 Requirement update/init 边界启动。
- 多命中：停止并报告冲突，不猜测应该复用哪个 Root，也不创建第三个 Root。
- 唯一命中已完成 Requirement：同样不创建新 Root；新维护需求必须使用新话题/Session。

本决策复用 ADR-0035 已有 Card Binding，但不把 Binding 变成 Flow State 或全局 Registry，不修改
Schema、写入时机或不可隐式改绑语义。查找和选择只是 Agent Skill 协议；CLI Runtime 不新增
`locate`、Registry、索引、迁移或 fallback。ADR-0066 的三阶段八步、唯一人工门禁、本地验证与
MR 交接边界不变；ADR-0059 的 Workflow 选择与 ADR-0053 的“只在初始化前更新”边界保留。

本决策不保证跨 Session 自动找回 Requirement，也不合并已经分叉或有未提交产品变更的
Worktree。当前 Session 内误建且源码 Worktree clean 的重复 Requirement 在需求事实合并后直接
移除，不保留兼容或迁移层。张菲帆通过 `om_x100b678abaa41ca0b3018e94e33bb91` 批准本决策进入实现。
