---
status: accepted
date: 2026-08-25
amends: ADR-0055
---

# 并发运行隔离的 Requirement E2E Shard

`./tests/run-e2e` 继续从当前工作树构建一次真实 CLI，并保留完整 Requirement
lifecycle 与动态发现的每个 `workflows/*/workflow.yaml` 所对应的生产 Workflow Route
Matrix。ADR-0055 中的“依次完成”改为：构建成功后，lifecycle 与每个 Workflow Matrix 作为相互隔离的
shard 并发运行，全部使用同一个待测二进制与 source commit。

每个 shard 使用独立 Requirement Root、输出目录、fake Lark 日志、Trace 文件和进程日志，
不得共享可写 Requirement 状态。父进程等待全部已启动 shard 并聚合退出码；任一 shard
失败都使入口非零，但不能因此丢弃其他 shard 的报告或 audit。成功时终端只展示 shard
状态与报告路径；完整 Agent 请求、CLI 原始响应和状态变化继续保存在 per-shard 日志，失败时
展开对应日志。

入口仍在所有 shard 完成后验证执行前后源码状态一致，并保留 run root、source commit、dirty
状态和二进制 SHA-256。不得通过持久化缓存、跨运行复用二进制、改动路径过滤、风险分级或删减
断言换取速度；不并行单个 Workflow 内部的状态推进，也不新增依赖、配置开关、测试框架或第三个
公开测试入口。

本决策只修订 ADR-0055 的执行顺序和终端展示。ADR-0055 的真实 CLI、mock 外部系统、完整
覆盖、审计现场与源码状态边界，以及 ADR-0063 的 `targeted|e2e` 风险分档和最终 HEAD 验证
要求保持不变。
