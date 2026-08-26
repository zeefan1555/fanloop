# State 是唯一运行状态源

在重设计后的存储模型中，`.fanloop/state.json` 是恢复 Fanloop Workflow 的唯一状态源。`events.jsonl` 是持久审计历史而不是重放日志，`events.md` 是可重建的人类投影；只有真实投递重试流程出现后才增加外部投递状态。行为等价阶段继续保留 `.prd-flow`，后续迁移也不再声称文件系统能够提供多文件原子事务。
