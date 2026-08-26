---
status: accepted
date: 2026-08-18
amends: ADR-0038, ADR-0040, ADR-0042
---

# 使用 Thrift 作为 Workflow YAML IDL

Fanloop 的五份 Workflow YAML 统一以根级 `idl/yaml.thrift` 作为字段、枚举、稳定
field ID 和各文件 schema version 的唯一可编辑真值。`thriftgo` 与 validator 生成
`internal/idl/yamlidl`；`go.yaml.in/yaml/v3` 继续负责 YAML 语法解析，并直接严格解码
到生成类型。手写 Workflow 领域模型只保留规范化运行时结构与行为，不再携带 YAML
tag、YAML Document DTO、重复枚举值或数字 schema version。

`yaml.thrift` 只覆盖不可拆分的 `workflow.yaml`、`flow.yaml`、`condition.yaml`、
`loop.yaml` 和 `prompt.yaml`。它不声明 Service，不进入公开命令目录，也不定义通用
YAML AST。`defaults.json`、Markdown、Card JSON 和 Requirement 持久化文件不属于该
IDL。

字段级类型、枚举和局部约束由生成类型与 validator 约束；Prompt/Condition/Step
引用、ID 唯一性、Flow 可达性、前进与回流方向、OR-of-AND 互斥和 Route 歧义等
跨文件图不变量继续由通用 `internal/workflow/validate.go` 校验。CLI 不承载任何生产
Workflow、Step、Condition 或 Route 的专用分支。

生成类型只作为 YAML authoring 边界。Loader 在严格解码后显式规范化为现有
Workflow 领域模型，Runtime、Status、Trace、Card 和持久化继续消费领域模型及各自
投影。`yaml.thrift` 不复用 `flow.thrift` 或 `storage.thrift` 的 DTO；作者配置、公开
命令和持久化格式保持独立演进，只在显式边界转换。

本次迁移不修改五份生产 YAML 的任何字节、schema version、引用、Route 或推进语义，
也不修改公共 CLI JSON Contract。Workflow 11.0.0 的规范化 digest 必须继续为
`sha256:eff6b09cfc0406db29117d38703bb25bedeb1d8d89d6994b1ca0eb7f9d8220fe`。
不保留手写 YAML Document decoder、重复 YAML tag、兼容 adapter、双 decoder、Feature
Flag 或 fallback。

本决策修订 ADR-0038 中 Workflow YAML 字段和枚举的可编辑来源，但保留五文件架构、
原子 Condition、OR-of-AND、显式 RouteSelection 和通用 Runtime 语义；修订 ADR-0042
的生成包集合，从七个领域包扩展为包含 `yamlidl` 的八个。ADR-0040 的公开命令
Thrift-first 契约保持不变，`yaml.thrift` 只是无 Service 的内部 authoring IDL。
