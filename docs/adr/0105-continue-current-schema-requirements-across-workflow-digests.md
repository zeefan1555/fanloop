---
status: accepted
date: 2026-09-17
supersedes_in_part: ADR-0019, ADR-0059, ADR-0062, ADR-0092, ADR-0098, ADR-0099, ADR-0102
---

# 当前 Schema Requirement 跨 Workflow digest 继续运行

当前 Schema 的 Requirement 运行时按持久 Workflow ID 加载当前生产 Bundle。State、Output Registry、
Event 与 Card Projection 中的创建 Release version 和 Workflow `id + digest` 保持为不可变 provenance；
后续写入继续沿用该原始引用，不做隐式迁移。digest 或创建 Release 的差异本身不再阻断 Status、Report、
Card、Trace 或 Requirement Doctor。

当前 Step 在新 Bundle 中不存在时返回 `WORKFLOW_MISMATCH`。其他 State、Output、Condition、producer、
Event payload、因果链、失效集合、replay tail 或目标 Step 不兼容继续返回 `STATE_CORRUPT`。跨 digest
回放中的历史 Result 已由原 Bundle 准入，不再使用当前 `flow.yaml` 或 `loop.yaml` Route predicate 重判；
新 Result 仍在原子提交前按当前 Bundle 完整校验，不能借历史放宽绕过新门禁。

严格 `workflow.LoadRef` 保留给 Release artifact、Manifest、安装、打包和显式精确引用校验。运行时不增加
旧 Schema reader、migration、历史 Bundle catalog、fallback loader、配置开关或 Workflow 专用分支，
也不修改 Thrift IDL、持久 Schema、生产 Workflow YAML 或公开 JSON 结构。

本决策只取代所列 ADR 中“运行中 Requirement 必须由精确旧 Release/digest 解释”的运行时部分。
固定控制器仍可用于 Maintainer 候选隔离和流程连续性，但不再是当前 Schema Requirement 跨版本运行的
产品兼容前提；五文件真值、不可变 provenance 与 Release 完整性边界保持不变。
