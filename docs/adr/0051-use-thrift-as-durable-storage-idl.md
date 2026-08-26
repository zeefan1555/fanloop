---
status: accepted
date: 2026-08-18
amends: ADR-0034, ADR-0039, ADR-0040, ADR-0042, ADR-0048
---

# 使用 Thrift 作为 durable storage IDL

Fanloop 自有的结构化 Requirement 文件统一以根级 `idl/storage.thrift` 作为字段、枚举、Event payload union、稳定 field ID 和 schema version 的唯一可编辑真值。`thriftgo` 与 validator 生成 `internal/idl/storageidl`；运行时领域模型只负责状态机和业务不变量，通过显式边界转换读写生成类型，不再通过手写 JSON tag 定义第二套落盘结构。JSON/JSONL 继续作为人类可读的文件编码，不引入 Thrift Server、Client、Processor、网络 Transport 或二进制协议。

Storage IDL 覆盖 `.fanloop/flow/state.json`、`.fanloop/output/state.json`、每行 `.fanloop/trace/events.jsonl`、`.fanloop/trace/config.json`、`.fanloop/card/projection.json` 和 `.fanloop/card/config.json`。Event payload 使用恰好一个成员的 Thrift union，并与 EventKind 精确对应。时间统一保存为 RFC3339 字符串；动态 Output 值复用 `common.JsonValue` 的唯一自然 JSON adapter。Trace Config 新增 schema version，并把 `trace_doc_url` 直接切换为 `trace_document_url`。

本次直接切换 State/Event Schema 10、Output Registry Schema 2、Card Projection Schema 3、Card Binding Schema 2 和 Trace Config Schema 1。旧手写落盘 encoder/decoder、旧 schema decoder、双读写、迁移 adapter、Feature Flag 和 fallback 全部删除。运行中的旧 Requirement 由其已安装旧 Release 解释或在切换前结束。

Storage IDL 与公开命令 IDL 是两个独立演进边界：`storage.thrift` 不复用 Flow、Trace 或 Card 的业务 DTO，只复用基础 `common.JsonValue`。这避免公开响应字段变化静默改写持久化格式。`storage.thrift` 不声明 Service，也不进入 12 个公开命令目录。

Markdown `trace/events.md` 不属于结构化 IDL。不可变 `card/<timestamp>.json` 是 `card.render` 返回并交给 Botmux 的原始 `common.JsonValue`，继续由 `card.thrift` 公共契约和 Lark Schema 约束，不增加会破坏直接投递的 Storage wrapper。Flow/Output/Event 原子提交、Card best-effort 投影、Trace 可重建投影和 Card 独立读取边界均保持不变。
