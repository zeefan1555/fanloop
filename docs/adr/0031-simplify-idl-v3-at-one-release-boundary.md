---
status: accepted
date: 2026-08-12
---

# IDL V3 在一个发布边界删除重复公共事实

Fanloop 直接切换到 IDL V3：`flow.move` 政名为 `flow.report`，Request 删除 `report/details` 包装，所有公共写响应删除 Store 文件清单，Loop 使用 `category`，Trace 和 Card 只返回调用方可消费的领域结果。State/Event 同时切换到 Schema 4；Workflow Schema 3 与 CardSnapshot Schema 2 保持不变。公共模型删除规则快照、digest 和内部路径不会删除 durable Event、Manifest 与 CardSnapshot 中负责审计和完整性校验的事实。旧命令、旧字段和旧 Schema decoder 不保留 alias、迁移或双读写。
