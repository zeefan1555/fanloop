---
status: accepted
date: 2026-08-29
amends: ADR-0003, ADR-0035, ADR-0037, ADR-0039, ADR-0048, ADR-0066, ADR-0079
---

# 通过 Workflow Condition 发布 Human Step 全景材料

Panorama 是 Human Step 的审核前置事实，不再是 Flow Runtime 的自动远端副作用。Go Runtime 删除
`AttemptPanoramaDelivery` 及其 `botmux send` 子进程调用；`flow init`、`flow report progress` 和
`flow report result` 仍在本地提交成功后更新 Card Projection，并保留既有 Trace provision、自动
sync、显式 `trace` 命令、Trace State/Storage 与全部 Thrift 契约。显式 `card render`、不可变 Card
快照和 Card Binding 同样保留。

两套生产 Workflow 在 `condition.yaml` 共同声明原子 Condition
`panorama_card_published`，输出 `panorama_card_receipt_id:string`。所有 Human Step 的批准、拒绝和
反馈分类 Route 都把该 Condition 与人工结论放在同一 AND 组；`fanloop-maintainer` 的
`requirements_changed` 早期回流保持原组合。对应 `prompt.yaml` 与已绑定 Skill 要求 Agent 根据
最新 Status、有效 Outputs 和审核材料生成自包含全景，通过当前 Agent 渠道发送并回读成功，再把本轮
真实 messageId 或 Agent 交互事件 ID 与人工结论一起上报。前一 Step 或前一次进入同一 Step 的回执
不可复用；Loop 的现有 Output 失效规则会移除被回流 Human Step 及其下游产生的旧回执。

CLI 只验证 Condition Output 类型、互斥关系和所选 Route 的唯一匹配，不发送消息，也不证明回执或
人工结论真实。五份 YAML 因而完整表达“先发布审核全景，再接受人工决定”的推进语义；运行时不新增
Workflow、Step 或 Condition 专用分支。

本次不修改两套 `workflow.yaml` 的任何字节，Step `id`、`name`、顺序和 `executor` 均不变；不修改
YAML Schema、`idl/*.thrift`、生成物、State/Event Schema 或公开 CLI Request/Response。旧
Requirement 继续由其绑定 Release 解释，不增加兼容层或迁移器。
