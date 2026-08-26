# Event 只记录已经接受的 durable effect

Event Schema 3 使用稳定信封、typed payload 和可选 `caused_by_event_id`。Flow/Loop payload 用明确的 `from_step_id/to_step_id` 保存前后事实；Trace 和 Card 只保存各自已经接受的效果。只读、`--dry-run`、业务拒绝与可重建投影不产生 Event。原生运行时随本地事实变化自动记录 Event，因此不保留独立 `trace record`。
