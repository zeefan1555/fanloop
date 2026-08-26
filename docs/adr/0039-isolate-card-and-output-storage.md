---
status: accepted
date: 2026-08-16
---

# 隔离 Card 投影与当前 Output 存储

Requirement 本地数据直接切换为四个职责目录：`.fanloop/flow/state.json` 只保存流程游标、当前执行事实、Requirement、Release、Integration 和最后 Event 指针；`.fanloop/output/state.json` 保存当前仍有效的 RegisteredOutput；`.fanloop/trace/` 保存不可变 Event 与 Trace 投影；`.fanloop/card/projection.json` 保存 Card 当前展示所需的独立投影，非 dry-run 渲染仍生成不可变 `.fanloop/card/<timestamp>.json` 快照。

`flow init`、被接受的 `flow report progress` 和 `flow report result` 先提交 Flow State、Output Registry 与 Event，再使用已经校验并提交后的当前事实更新 Card Projection。Card Projection 更新和远端发送仍是 best-effort：失败只产生 warning，不回滚已经成立的 Flow；自动发送只消费本轮成功渲染返回的精确 `snapshot_path`。Output Registry 属于恢复执行所需的本地事实，必须与 Flow State/Event 在同一次 Requirement lock 提交中写入，不能 best-effort。

`card render` 只读取 `.fanloop/card/projection.json` 和绑定的 Workflow Bundle，不读取 `.fanloop/trace/events.jsonl`，也不向 Trace 追加 `card.rendered` Event。Card 不展示“最近一次 Result”，不保存 latest Result、ConditionResult、Transition、accepted/invalidated OutputChanges 或历史 Evidence；当前 Step 的 Summary/Evidence 和当前有效 Output 由 Card Projection 保存。Card Projection 不存在、损坏或绑定错误时直接失败，不回退读取 State/Event。

Doctor 分别校验 Flow State、Output Registry、Event/Trace、Card Projection、Card Binding 和 Card 快照。Trace 损坏不得使 Card 域检查自动跳过；Card 快照只按文件名和自身 JSON/Schema 校验，不再依赖 Trace Event 对账。这里不新增 Card Event Log、投递回执、重试队列或 public Card update 命令。

该切换使用 State/Event Schema 9、Output Registry Schema 1 和 Card Projection Schema 1。旧 State 内嵌 `outputs`、`card.rendered` Event、从 Event 反查 latest Result 以及对应 decoder/adapter 直接删除，不提供双读写、迁移、Feature Flag 或 fallback。运行中的旧 Requirement 继续由其绑定的旧 Release 解释。

本决策取代 ADR-0013 中“Card 快照必须由审计 Event 记录路径与摘要”的部分，取代 ADR-0034 中 State 内嵌 Output 和 Card 只有原始快照的布局，取代 ADR-0038 中 State 直接保存 RegisteredOutput 以及 Panorama 从 State/Event 投影的部分；保留不可变快照、State 当前事实、Event durable effect、本地 Flow 提交先于 best-effort 投影、renderer 与 sender 分离，以及五份 Workflow YAML 的架构真值。
