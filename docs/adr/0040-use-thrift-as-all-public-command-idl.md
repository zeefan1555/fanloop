---
status: accepted
date: 2026-08-17
amended_by: ADR-0050
---

# 所有公开 CLI 命令使用 Thrift-first 契约

Fanloop 的 14 个公开叶子命令统一以模块化 Thrift IDL 作为方法、Request、Response、枚举、公共错误目录和命令元数据的唯一可编辑真值。IDL 按 `common`、`error`、`flow`、`trace`、`card` 与根级运维命令域拆分，并提供一个只负责聚合 include 图的生成入口。`common` 与 `error` 是基础类型库，不是 CLI 命令域。

每个公开命令对应一个 Thrift Service 方法。方法签名确定 Request、Response、Requirement Root 和 dry-run 运行参数；方法注解确定稳定 Command ID、摘要、读写风险、Requirement 范围、dry-run 能力和允许的错误码。生成步骤从 Service 与注解产生 Go CommandSpec 注册表；`schema list/describe`、Cobra 命令元数据和运行控制只消费该生成物，不再维护手写命令目录或第二套错误清单。Cobra 继续负责 typed flags、`--input`、stdin、帮助文本和调用本地 Runtime，不从 Thrift 生成整棵命令树。

`flow`、`trace`、`card` 与运维 Runtime 直接实现各自生成的 Service 接口。Thrift 在这里是进程内公共接口，不引入 Thrift Server、Client、Processor、网络 Transport 或 RPC 生命周期。现有统一 `ok/data/meta/_notice` JSON 信封继续由一个 Go 输出 Adapter 编码；Thrift 没有泛型 Response，不为每个命令复制一份信封类型。

所有命令共享一个全局 ErrorCode、ErrorType、ErrorSpec 和 PublicError。方法只声明本命令可能产生的 ErrorCode；运行时错误实例携带 code、message 和可选 details，CLI Adapter 根据全局 ErrorSpec 补齐 type、hint、retryable 与退出码。现有 `INTERNAL_ERROR`、`INTERNAL` 等重复错误码在本次直接切换中统一，废弃的字符串错误目录直接删除。

跨领域自然 JSON 使用一个递归 JsonValue union，并保留唯一的自然 JSON 编解码 Adapter。Flow Output、Card content 与 Schema JSON 可复用该类型，同时由各领域继续施加自身约束。时间字段用稳定的 RFC3339 字符串公共表示，不把 Go `time.Time` 暴露为另一套 IDL 真值。

Loop 不新增独立 Service。当前公开 CLI 没有 `loop` 命令；Loop 是 `flow report result` 的 RouteSelection、Transition 和 AvailableRoute 方向，仍由 `loop.yaml` 定义业务 Route。只有未来新增真实公开 `fanloop loop ...` 命令并通过独立 ADR 后，才增加 Loop Service。五份 Workflow YAML 的结构、引用、Condition 组合和推进/回流语义不因本决策改变。

生成工具继续固定当前 `thriftgo` 与 validator 版本，递归生成 include 图，并新增生成物新鲜度门禁。生成的 Request/Response、Service、validator 和 CommandSpec 注册表全部提交到版本库；生成后存在 diff 即构建失败。公共 CLI Contract、Schema examples、错误目录和 Cobra/IDL 一一对应测试构成主要验收面。

本次在一个发布边界直接删除手写公共 DTO、手写 CommandSpec 数据、分散 ErrorSpec、旧 PublicError 转换和重复 JSON 类型，不提供 alias、兼容 Adapter、双注册、Feature Flag 或旧 IDL fallback。隐藏的安装引导命令不是公开 Contract，不进入 Service 或 Schema 目录。

本决策全面取代 ADR-0025 的 Go CommandSpec IDL 真值，并把 ADR-0037 的 Thrift-first 约束从 Flow 扩展到所有公开命令。保留 ADR-0011 的统一 Agent JSON 信封、ADR-0012 的一个命令一个强类型 Request、ADR-0021 的稳定退出码集合、ADR-0038 的 Agent/CLI/Workflow Route 职责边界和 ADR-0039 的 Flow、Output、Trace、Card 存储隔离。
