---
status: accepted
date: 2026-10-15
supersedes: ADR-0081, ADR-0119
supersedes_in_part: ADR-0104, ADR-0130
superseded_by: ADR-0143-everything-is-condition
---

# 将 Panorama 投递改为 Runtime Required Action

Panorama 是所有 Workflow 共用的节点进入能力，不再由业务 YAML Condition 和重复 Skill 表达。`flow init`
以及 accepted `advanced`、`looped`、`jumped`、`completed` Event 创建一个持久
`panorama_delivery_required` action；`progress`、`started`、rejected request 与 dry-run 不创建。
同 Step 回流或跳转重入仍产生新的 action。completed State 在 `current` 消失后继续暴露最终 action。

`flow init`、`flow status` 和真实 Result 响应在 `FlowState.pending_actions` 返回 action ID、触发 Event ID、
可选 Step ID，以及 CLI Release 中唯一 `fanloop-panorama-delivery` Skill 的绝对路径。Agent 先按该 Skill 在
`botmux`、`local_agent`、`aime` 或 `aiden` 的唯一宿主通道原样投递 renderer 内容，再使用
`flow action acknowledge` 提交同一个 action ID 与本次 non-dry-run render 的精确 snapshot path。
ack 成功前业务 progress/result 返回 `REPORT_NOT_ALLOWED`；Runtime 不调用远端发送工具，也不自动重试。

Pending truth 只从不可变 Event 历史恢复：init 或打开 action 的 Result Event 打开 action，新增
`flow_action_acknowledged` Event 关闭 action。Flow State/Event Schema 同步从 14 提升至 15，不提供迁移、
双读、fallback 或兼容层。Output Registry 仍为 4，Card Projection 仍为 7。`pending_actions` 不是存储字段。
Card 继续使用唯一 `cardContent`。快照文件名在原时间戳后携带 `Projection.SourceEventID` 的 SHA-256。
pending action 期间，`card render` 要求 Projection source 等于 trigger；`trace bind` 与 `trace sync`
改变 Card 可见状态后刷新 Projection，并继续使用原 trigger 作为 Projection source。

Loader 将 `panorama_presented` 与 `panorama_card_published` 作为 Runtime 保留字直接拒绝，公开错误码为
`CONDITION_RESERVED`（32），类型 workflow，退出码 5，不可重试。诊断句与 loop_cli 一致：
`Condition "panorama_presented" is reserved for the Loop Runtime Panorama action`。
叶子 `--help` 的 Errors 段列出该码。本仓库的 Example 包删除自声明的 Panorama Condition、Prompt、
SkillBinding、Route operand 与 `example-panorama` Skill。`workflow.yaml` 的 Step id、name、顺序与
executor 不变。

生产 Workflow 包（ai-daily-brief、code-change、daily-journal、general、pstack-native、
remnote-flashcards、second-brain-skills-maintainer、write-skill 等）只存在于 second-brain-skills 的
`fanloop-workflow/<id>/`，不在本仓库。本变更不修改那些包；它们在删除保留 Condition 之前不能被新 CLI 加载。
不移植 loop_cli 同期的 Workflow reconciliation（`replaces`、`WORKFLOW_RECONCILIATION_REQUIRED`、
Output retirement tombstone）。

## 契约审核

用户要求把 loop_cli head `e4d8f1a` 的 Panorama runtime action 按相同语义和配置格式移植到 fanloop，
并要求保留字拒绝使用可在叶子 `--help` Errors 段列出的明确错误码。主 Agent 按该授权审阅并接受下列
Thrift 变更；不改 `idl/yaml.thrift`。

- `idl/error.thrift`：新增 `CONDITION_RESERVED = 32`，以及 ERROR_SPECS 条目（workflow，exit 5，
  retryable false，hint 要求删除保留 Condition）。
- `idl/storage.thrift`：`FLOW_STATE_SCHEMA_VERSION` 与 `EVENT_SCHEMA_VERSION` 从 14 改为 15。
  `EventKind` 新增 `flow_action_acknowledged = 7`。新增 `ActionType.panorama_delivery_required = 1`
  与 `FlowActionAcknowledgedPayload`（action_id、action_type、trigger_event_id、panorama_snapshot_path），
  并作为 `EventPayload` union 成员 7。
- `idl/flow.thrift`：`FlowState` 增加字段 5 `pending_actions`。新增 `PendingAction`、
  `FlowActionAcknowledgeRequest` / `Response`，以及 service 方法
  `cli.id = "flow.action.acknowledge"`。`CONDITION_RESERVED` 列入 init、status、skills、progress、
  result 与 acknowledge 的错误目录。

生成物只通过 `scripts/generate-cli-idl.sh` 更新。
