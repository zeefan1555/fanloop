---
status: superseded by ADR-0029
---

# 一个公开写命令只提交一个事实

最终公开入口收敛为 `flow init/status/update/approve/migrate`、`loop feedback`、`trace bind/status/sync/render` 和 `card render`。自动 Event 取代 `trace record`，重复别名与迁移入口合并，Panorama 变成 Card 的视图参数。一次写命令只提交一个动作结果、反馈或绑定变化，并统一返回 `current/completed/next/card/changes`；下一 Route 和独立 Card 指令只能由绑定的 Fanloop Workflow 计算。
