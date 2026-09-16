---
name: fanloop-dev-decision-receipt
description: 为 fanloop-maintainer 的需求、Step 跳转和最终验收记录真实 human 决定回执。
---

# Maintainer Decision Receipt

本 Skill 只记录主观决定，不代替测试、Review、候选身份、CI 或 PR 门禁。Developer 不得自批、改写或概括 human 的原始决定。

在 Requirement Root 的 `.scratch/decision-receipts.jsonl` 追加单行 JSON；创建时使用 `0600`，写入前拒绝符号链接、非普通文件和已损坏 JSONL。每条回执必须且只能包含：

- `schema_version=1`
- `receipt_id`：`decision-` 加 `idempotency_key` 的 SHA-256
- `idempotency_key`：`fanloop-maintainer:<step_id>:<decision_class>:<source_kind>:<source_id>`
- `step_id`、`decision_class`、`decision`、`decision_text`
- `actor_type=human`、`actor_id`、`actor_name`
- `source_kind`、`source_id`、`evidence`
- `recorder_type=agent`、`recorder_id`、`recorder_name`、`recorded_at`

`source_kind` 只允许 `lark_message`、`host_message` 或 `host_turn`；`source_id` 必须是宿主实际提供的不可变引用，不得用时间或随机值伪造。同一 `idempotency_key` 的完全相同重试复用旧回执；同 key 不同内容立即 blocked。

使用同目录临时文件、fsync 和原子 rename 写入，再按 `receipt_id` 精确回读并校验字段、内容和权限。只在唯一回读一致时返回 receipt ID。
