---
name: fanloop-dev-implement
description: 执行子 Agent在 Fanloop 源码 worktree 直接完成实现、验证、独立整体 CR 和 implementation-report.md，并只向主 Agent汇报。
---

# 实施修复

1. 执行子 Agent确认源码位于独立 worktree，Issue Workspace 只保存需求、Spec、Tickets 和报告；不得回退其他人的并发改动。
2. 执行子 Agent按 ticket frontier 直接完成最小纵向实现，只向主 Agent回报。只在已确认的公开 Test Seam 且存在独立预期时使用 `fanloop-dev-tdd`；删除废弃路径，不加兼容层、投机配置或一次性抽象。
3. 用户可见 CLI、Workflow、安装、状态、投影或 Skill 行为变化时，读取相邻的 `fanloop-dev-maintain-verification/SKILL.md`，完成 Feature source audit 与 live pass并保留隔离公开 CLI 证据；验证资产需修改时一并修正，product gap 作为实现缺陷处理。
4. 同步行为契约、相关 ADR 和聚焦测试，运行聚焦测试与 `./tests/run-unit`。主 Agent发现缺口时可通过 follow-up 要求继续修复并重跑，但不接管实现。
5. 实现完成后派发一个不继承实现上下文的独立 Reviewer 审查全部累计 diff。P0/P1 由执行子 Agent修复，再由原 Reviewer复核；任何代码、测试、Workflow、验证资产或构建输入变化都使原全量验证失效。
6. 把 review_base、受测受审 tree、验证地图结论、命令、退出码、findings、修复和 ADR impact 写入 `implementation-report.md`。提交本地分支，证明最终 HEAD 的 tree 与报告一致且工作树 clean。
7. 逐项核验当前 Step 的全部必需 Skills、报告、HEAD 和 clean 状态后，上报 `implementation_completed=<完整 HEAD>` 与 `implementation_report_written=implementation-report.md`。

本 Step 不 push、不创建 PR、不合并。需求或方案不成立时选择对应回流 Condition；需要主 Agent决定时先报告并等待明确回复。
