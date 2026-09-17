---
name: fanloop-dev-decision-receipt
description: 由执行子 Agent为 fanloop-maintainer 的主 Agent需求决定和最终候选决定记录可回读回执。
---

# Maintainer Decision Receipt

本 Skill 只记录主观决定，不代替测试、Review、候选身份、CI 或 PR 门禁。需求批准和最终候选验收必须由
执行子 Agent请求并记录主 Agent明确决定；执行子 Agent不得自批。

在 Requirement Root 的 `.scratch/decision-receipts.jsonl` 追加单行 JSON；创建时使用 `0600`，写入前
拒绝符号链接、非普通文件和损坏 JSONL。每条回执必须且只能包含：

- `schema_version=1`
- `receipt_id`：`decision-` 加 `idempotency_key` 的 SHA-256
- `idempotency_key`：`fanloop-maintainer:<step_id>:<decision_class>:<source_kind>:<source_id>`
- `step_id`、`decision_class`、`decision`、`decision_text`
- `actor_type=agent`、`actor_id`、`actor_name`
- `source_kind`、`source_id`、`evidence`
- `recorder_type=agent`、`recorder_id`、`recorder_name`、`recorded_at`

`source_kind` 只允许 `lark_message`、`host_message` 或 `host_turn`；`source_id` 必须是宿主提供的真实引用。
同一 `idempotency_key` 的相同重试复用旧回执，不同内容立即 blocked。使用同目录临时文件、fsync 和原子
rename 写入，再按 `receipt_id` 精确回读；只有唯一回读一致时返回 receipt ID。
