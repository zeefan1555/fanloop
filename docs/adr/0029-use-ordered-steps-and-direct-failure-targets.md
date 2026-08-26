---
status: accepted
---

# Workflow 使用有序 Step、全局 Step ID 和直接失败目标

Fanloop Workflow 由有序 Stage 和 Step 定义正常路径；同一时刻最多一个活动 Step，通过后进入顺序中的下一 Step。`step_id` 在整个 Workflow 内唯一，Stage 只分组和展示。每个 Step 的 `on_failure` 将调用方报告的 `failure_name` 映射到唯一 `return_to_step_id`；调用方不能提交目标，也不声明通用 Route。执行方由 Step 的 `executor` 唯一决定，Agent/Human 共用一行 `instruction`。这一限制放弃 DAG、并行和条件成功分支，换取从上到下可读、可恢复、可审计的第一版模型。本决策取代 ADR-0007、ADR-0017 和 ADR-0018，并修订 ADR-0019 的旧兼容与迁移部分。
