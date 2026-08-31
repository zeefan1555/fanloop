---
name: fanloop-dev-implement
description: 在 Fanloop 源码 worktree 按确认 Spec/Tickets 实现最小改动，并维护 implementation-report.md 与唯一飞书研发实现报告。
---

# 实施修复

1. 确认源码位于独立 worktree，Issue Workspace 只保存需求、Spec、Tickets 和报告。
2. 按 ticket frontier 完成最小纵向实现。只在已确认的公开 Test Seam 且存在独立预期时使用 `fanloop-dev-tdd`；删除废弃路径，不加兼容层、投机配置或一次性抽象。
3. 同步行为契约、相关 ADR 和聚焦测试；运行聚焦测试与必要格式检查。完整 `./tests/run-unit`、`./tests/run-e2e` 由 Review 覆盖最终候选。
4. 自查 diff 并提交本地分支。工作树必须干净；记录当前 HEAD、main...HEAD、改动、ADR impact、聚焦命令、退出码和结果到 Issue Workspace 的 `implementation-report.md`。
5. 用包含 Requirement 身份的稳定标题查找飞书文档：零命中创建，唯一命中更新，多命中 blocked。把报告发布为唯一飞书研发实现报告，按 URL 语义回读正文非空、当前 HEAD 与实现结论一致。
6. 成功后上报 `implementation_completed`、`implementation_report_written=implementation-report.md`、`implementation_document_published=<URL>`。

不 push、不创建 PR、不合并。需求、方案或实现不成立时只选择对应回流 Condition；飞书写入或语义回读失败时保持 blocked。
