---
status: superseded by ADR-0071
date: 2026-08-23
amends: ADR-0066, ADR-0069
---

# 维护者 MR 使用人工终批的 Codebase 合并队列

`fanloop-maintainer` 的八步 Workflow 与 MR 交接 terminal 保持不变。交接后由独立的
`fanloop-cr` 完成审查；只有 clean `Approve` verdict 才能请求张菲帆终批。Review 通过本身不构成
合并授权，`fanloop-dev` 也不得代替 Review 角色入队。

权威 handoff 由当前飞书 root/thread、Requirement、MR URL 与 reviewed HEAD 联合定位，不再要求 MR URL
全局唯一。记录必须是 ADR-0069 的 `schemaVersion=2` complete checkpoint；同一 MR 的其他记录必须已标记
superseded 并指向当前 root。这样旧 Requirement 可保留历史，但不能重新成为 queue 权威来源。

终批必须是 verdict 卡回读成功后、同一权威 MR 话题中由张菲帆发送的新消息，去除首尾空白后精确
等于 `批准进入合并队列`。Review 应用 `cli_aaf634adc0789cc9` 只使用其自身作用域下已验证的张菲帆
OpenID `ou_5bfe2b1bfa7625ac055073ee38c8fd10`；请求 mention、回复 sender 校验和状态通知均使用这一身份，
不得复用 `fanloop-dev` 应用的 OpenID。旧消息、错误 sender、笼统同意、附加解释或其他话题消息都
无效。

批准后重新回读 MR，要求它仍 open、target 为 `master`、source HEAD 等于 reviewed HEAD，且不存在
Block/Recommend 或未处置 finding。当前版本使用现有 Codebase 用户凭证；`bytedcli --json auth userinfo`
必须精确返回 `zhangfeifan.15`，所以 Codebase 审计记录张菲帆为 queue actor。独立服务身份不属于本版。

`handoff.json.mergeQueue` 使用独立 `schemaVersion=1` 状态机：
`approval_request_pending -> approval_pending -> enqueue_pending -> queued -> merged|blocked`。每次入口先按
已有 phase 恢复，不得重置或重新发送批准请求。批准请求与状态通知使用 outbox checkpoint：外部写之前
在嵌套 `mergeQueue.notification` 中保存 `phase=send_pending`、精确正文、thread、CR app、目标 OpenID 和
尝试时间，不能覆盖主生命周期 phase；返回后保存 message ID。若发送后 checkpoint 前退出，只接受同
thread、同发送 app、同正文和同 mention 的唯一消息；零个或多个候选都 fail closed，不冒险重发。状态
通知恢复完成后，主 phase 仍保持原 queued、blocked 或 merged。

入队复用现有 `bytedcli codebase mr queue`。批准前存在的 entry 一律不收编。空队列回读后先写
`enqueue_pending`、唯一 attempt ID、reviewed HEAD、actor、merge method 与时间，再以 `merge_commit`
enqueue。结果不明确或跨会话恢复时，只接受 API 原始字段能证明 MR、target、source HEAD、merge method、
creator 与创建时间均关联本次 attempt 的唯一 entry；缺字段、旧 entry、其他 actor 或多个 entry 都
fail closed。零 entry 时可依赖 Codebase 的单 MR 队列唯一性重放一次相同 enqueue；第二次仍不明确则
blocked。queued phase 只跟踪已记录 entry ID，不再次 enqueue。

入队后每五分钟回读 queue entry 与 MR 状态，最多十二次。状态未变化不发送重复消息；只在 queued、
checks pending、blocked、merged 转换时通过同一 outbox 回复原话题，并以 CR 应用作用域 OpenID 真实
mention 张菲帆（即真实 mention 张菲帆）。超过一小时仍等待时停止本地轮询并如实报告，Codebase 队列继续处理。只有 MR 真实回读
为 merged 才能宣称已合并。

`fanloop-cr` 继续禁止 Codebase `--approve`、`--request-changes`、direct merge、dequeue 和 release。
任一身份、时序、HEAD、finding、queue entry 或回读异常都 fail closed，不降级为直接合并，也不触发
自动发布。ADR-0066 的本地验证、AI Code Review、八步拓扑和 handoff terminal 不变；ADR-0069 的
Card 与权威话题继续提供本协议所需的 review/approval 上下文。
