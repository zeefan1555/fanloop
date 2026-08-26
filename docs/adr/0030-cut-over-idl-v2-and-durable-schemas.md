---
status: accepted
date: 2026-08-12
---

# IDL V2 与 durable schemas 在同一发布边界直接切换

Fanloop 将全部公开命令切换到命令同名的 Request/Response，并同时切换 Workflow Schema 3、State/Event Schema 3 与 CardSnapshot Schema 2。公开写结果明确区分调用方 `report`、业务 `effect/outcome`、命令后 `state` 和本地 `changes`。Flow 与 Loop 共享唯一 WorkflowState 投影；Trace 与 Card 只返回各自领域事实。旧字段、旧类型、旧 Event kind 和旧 Schema decoder 直接删除，不提供 alias、双读写、迁移 adapter、Feature Flag 或回退路径。Contract Golden、Schema 自校验、durable invariant、E2E 和 Release 生命周期共同构成发布门禁。
