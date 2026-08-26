---
status: accepted
date: 2026-08-18
amends: ADR-0012, ADR-0040
---

# 用叶子命令 Help 取代公开 Schema 发现入口

Fanloop 的每个公开叶子命令直接在 `--help` 中给出完整 Request JSON 模板、
等价 typed flags、字段与互斥约束、Requirement Root、dry-run、执行效果和下一步。
空 Request 也明确显示 `{}`。根命令和命令组只负责导航，不复制叶子字段契约；
隐藏安装入口不属于公开 Contract。

静态 Help 只描述稳定 Request 结构。当前 Requirement 的 Step、Condition、
OutputSpec 与 Route 继续由最新 `flow status` 投影，Agent 从 Status 选择事实与
RouteSelection，CLI 按当前 State 和五份 Workflow YAML 校验。Help 不硬编码生产
Workflow ID，也不成为运行时校验输入。

公开 `schema list`、`schema describe` 与专用 JSON Schema 生成域在一个发布边界
直接删除，公开叶子命令从 14 个收敛为 12 个。对应 Ops Service 方法、DTO、Runtime、
测试、Goldens 和无人使用的 JSON Schema 依赖一并删除，不保留 alias、fallback、
Feature Flag 或双注册。

删除 Schema 不改变 Thrift-first 真值。模块化 Thrift Service、生成 Request/Response、
静态 validator、CommandSpec、公共 ErrorSpec、JsonValue、统一结果信封和领域 Runtime
继续保留；Cobra 继续负责命令路径、typed flags、输入传输、运行控制与 Help。Help
不复制完整 Response Schema 或 Error catalog。

本决策修订 ADR-0012 中“`schema list` 和 `schema describe` 是两个独立命令”的部分，
保留一个逻辑命令一个强类型 Request、typed flags/`--input` 二选一以及运行控制不进入
Request 的边界。它修订 ADR-0040 中 14 个公开叶子命令及 Schema 消费 CommandSpec 的
部分，保留所有剩余命令的 Thrift-first、生成 Service/validator/CommandSpec、Cobra
Adapter 与直接切换约束。完整行为契约与验收条件见
`docs/specs/agent-cli-help.md`。
