---
name: fanloop-dev-implement
description: 在 Fanloop 源码 worktree 按确认 Spec/Tickets 实现最小改动，完成全量本地验证和独立整体 CR，并维护 implementation-report.md。
---

# 实施修复

1. 确认源码位于独立 worktree，Issue Workspace 只保存需求、Spec、Tickets 和报告。
2. 按 ticket frontier 完成最小纵向实现。只在已确认的公开 Test Seam 且存在独立预期时使用 `fanloop-dev-tdd`；删除废弃路径，不加兼容层、投机配置或一次性抽象。
3. 同步行为契约、相关 ADR 和聚焦测试；运行聚焦测试、`./tests/run-unit` 与 `./tests/run-e2e`。
4. 使用一个不继承实现上下文的独立 Reviewer 审查全部累计 diff。P0/P1 由实现者修复，再由原 Reviewer 复核；任何代码、测试、Workflow 或构建输入变化都使原全量验证失效。
5. 把 review_base、受测受审 tree、命令、退出码、findings、修复和 ADR impact 写入 `implementation-report.md`。提交本地分支，证明最终 HEAD 的 tree 与报告一致且工作树 clean。
6. 成功后上报 `implementation_completed=<完整 HEAD>` 与 `implementation_report_written=implementation-report.md`。

不 push、不创建 PR、不合并。需求、方案或实现不成立时只选择对应回流 Condition。
