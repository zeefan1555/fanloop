---
status: accepted
date: 2026-09-17
amends: ADR-0055, ADR-0063, ADR-0078, ADR-0100, ADR-0101
---

# 退役本地 run-e2e 入口，保留 CI Requirement 验证

Fanloop 只保留一个本地开发测试入口：`./tests/run-unit`。开发实现运行受影响的聚焦测试和
`run-unit`；用户可见行为由隔离候选的 `fanloop verify`、Verification Skill 和无实现上下文的
Sub-agent 公开 CLI 验收覆盖，不再要求开发者重复执行 `./tests/run-e2e`。

原 `tests/run-e2e` 退役且不保留 alias。完整 Requirement lifecycle 与全部生产 Workflow Route
Matrix 继续作为远端 `requirement-e2e` required check，由 CI 私有脚本
`.github/scripts/run-requirement-validation` 执行。该脚本不是仓库公开测试入口，不进入本地 Maintainer
验证命令，也不包含已由隔离候选验收覆盖的 `verify smoke` shard。

任何候选源码、测试或验证资产变化都会使本地验证、Review、Sub-agent 验收与远端 CI 失效。合并仍要求
同一候选 SHA 的 `test (ubuntu)`、`test (macos)`、`requirement-e2e`、`install-doctor` 与
`governance` 全部通过。此次迁移不改变 Workflow YAML、Step、Route、Condition、Thrift、Runtime 或
产品行为。

用户于 2026-09-17 明确要求删除本地 E2E 入口并直接创建分支交付合并。
