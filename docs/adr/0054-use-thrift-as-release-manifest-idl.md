---
status: accepted
date: 2026-08-18
amends: ADR-0009, ADR-0024, ADR-0042
---

# 使用 Thrift 作为 Release Manifest IDL

Fanloop 的 `release.json` 统一以根级 `idl/release.thrift` 作为字段、稳定 field ID 和
局部字段约束的唯一可编辑真值。`thriftgo` 与 validator 生成
`internal/idl/releaseidl`；运行时文件继续使用自然 JSON，Schema Version 保持为 1。
现有手写 `Manifest`、`CLIRelease`、`Skill`、`Workflow` 和 `Asset` DTO 不再作为并行
契约来源。

`release.thrift` 直接复用 `ops.StateSchemaSupport`，不声明 Service、Error 或 enum。
生成 validator 负责 required、局部 pattern 和列表大小；Release/CLI 版本一致、State
Schema 读写关系、Skill/Workflow 唯一性、唯一默认版本、路径与文件名规则、摘要以及
四平台集合完整性继续由 `internal/release` 的通用 Validator 校验。

Go producer 与消费者统一服从生成类型。Node 和 shell 继续作为同一自然 JSON 的窄
Adapter，由跨运行时发布测试校验真实产物；本次不引入 JS codegen、JSON Schema、
RPC、通用 Manifest 框架、兼容 decoder、迁移或 fallback。

`idl/cli.thrift` include 该文件只为复用现有递归生成和新鲜度门禁，Release Manifest
不属于公开命令 Service。发布顺序、整包摘要校验、Doctor、原子切换、安装目录、
Workflow YAML、Durable Storage 和公共 CLI JSON Contract 均不改变。

本决策修订 ADR-0009 和 ADR-0024 中 `release.json` 字段真值的实现来源，但保留完整
配套 Release、只读 Doctor 与整包切换语义；修订 ADR-0042 的生成包集合，从包含
`yamlidl` 的八个领域包扩展为包含 `releaseidl` 的九个。ADR-0026、ADR-0032、
ADR-0041、ADR-0049、ADR-0053 的安装、修复、发布与更新边界保持不变。
