---
status: accepted
date: 2026-08-18
amends: ADR-0046, ADR-0049
---

# 仅在新 Workflow 初始化前刷新 Release

正式 `fanloop-workflow` Skill 每次启动或继续 Requirement 时先执行本地
`flow status`，不得在读取已有 State 前自动执行 Release 更新。Status 返回已初始化
状态时，继续使用当前 Release 解释并推进该 Requirement；此前单独执行的 update 即使
失败，也不阻止后续本地 Flow 命令。

只有 Status 返回 `NOT_INITIALIZED` 且用户要启动新 Workflow 时，Skill 才在
`flow init` 前执行一次：

```bash
fanloop update --action update
```

更新成功后重新读取当前 Release 的 `fanloop-workflow/SKILL.md` 与 `ref/role.md`，
再执行 `flow init`；重新读取不得触发第二次更新。更新失败时保留当前安装、原样报告
错误并停止，不创建新 Workflow State。

这一生命周期边界取代 ADR-0046 的“每次推进会话在 Status 前刷新”及其会话级失败
门禁，并修订 ADR-0049 的 native update 调用时机。ADR-0009 的普通业务命令不隐式
更新、ADR-0019/0037 的运行中 Requirement 绑定 Release 支持边界、ADR-0024 的整包
校验与失败保留当前版本、ADR-0026/0032/0041 的安装和修复语义保持不变。

不新增 TTL、后台任务、配置项、第二套版本探测、兼容入口或 Workflow YAML 变化。
