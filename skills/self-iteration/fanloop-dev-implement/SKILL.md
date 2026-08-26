---
name: fanloop-dev-implement
description: 实施 Fanloop CLI 修复。用于在维护者源码 worktree 按确认 Spec/Tickets 和 TDD 完成代码、聚焦测试与本地提交；不推送或创建 MR。
---

# 实施修复

1. 确认当前目录是独立源码 worktree、分支基于 bootstrap 记录的 `main` Commit；Issue Workspace 只存本地需求、Spec、Tickets 和证据，不写源码。
2. 按 ticket frontier 一次完成一个可独立验证的 Tracer Bullet。仅在 Ticket 声明的已确认公开 Test Seam 且存在独立正确预期时执行 `fanloop-dev-tdd`：一条 red 测试、一段最小实现、转绿后再继续；其余部分直接最小实现，不为满足流程制造测试。
3. 修根因，不顺手修无关问题；不保留废弃路径兼容层，不引入没有当前需求的配置或抽象。
4. 运行改动相关的聚焦测试和必要格式检查；最终由 `fanloop-dev-verify` 按风险执行 `targeted` 或 `e2e`。
5. 自查 diff 并提交到本地分支；使用仓库当前 Git 身份，不注入组织专属 trailer。
6. 不 push、不创建或更新 MR、不查询远端 checks、不自动 merge 或发布；这些外部写只属于最终 `fanloop-dev-mr-handoff`。
