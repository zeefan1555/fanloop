---
status: superseded by ADR-0090
date: 2026-08-24
amends: ADR-0066
supersedes: ADR-0069, ADR-0070
---

# 维护者 MR 交接使用 Botmux 人工审核话题

`fanloop-maintainer` 保留 ADR-0066 的三阶段八步拓扑、本地验证、AI Code Review 和以 MR 交接为
terminal 的边界。最终交接不再触发独立审查机器人或自动 merge queue；MR 由审批人人工审核，
不自动合并或发布。

MR 创建或更新后，`fanloop-dev-mr-handoff` 把 UTF-8 Markdown 正文写到 Issue Workspace，并执行：

```bash
botmux send --top-level --chat-id oc_example_review_previous \
  --mention ou_example_approver:审批人 \
  --mention ou_example_cc_one:关注人甲 \
  --mention ou_example_cc_two:关注人乙 \
  --content-file <issue-workspace>/handoff-topic.md
```

审批人是唯一审核人；关注人甲、关注人乙只作为 cc 关注人。三人都在同一卡片中真实 mention，cc 位于
影响范围后的末行，固定写为 `cc：@关注人甲 @关注人乙`。全角冒号确保首个 `@` 符合 Botmux 的内联
mention 边界，三个显式 mention 均由正文消费，不产生 `发送给` 尾栏，也不另发消息。正文包含 MR 标题、编号和真实链接，
并按固定顺序提供四个业务区块：
背景、问题、解法、影响范围。详细测试、ADR impact、风险和非目标继续保留在 MR 描述，不复制到话题。

成功回执以 `botmux send` 的零退出码及 JSON 中的 `success=true`、非空 `messageId`、非空 `sessionId`
和审批人、关注人甲、关注人乙三个准确 mention 为准。Issue Workspace 的 `handoff.json` 只保存 `phase=complete`、Requirement、
MR URL、reviewed HEAD、messageId 和 sessionId。相同 MR URL 与 reviewed HEAD 已完成时直接复用记录，
不重复发送。旧 Card 或 merge queue 记录不迁移、不兼容；遇到其他格式时停止并报告。

本决策删除 `fanloop-cr-review` Skill、verdict 模板、Card provider fixture，以及 Card 2.0、lark-cli、
provider projection、legacy replacement 和自动 merge queue 的维护契约。ADR-0069 与 ADR-0070 因此被
完整取代；ADR-0068 已由 ADR-0069 取代，继续作为历史记录保留。

五份生产 Workflow 中只修改 `prompt.yaml` 的 MR 交接 Prompt 与 handoff record 条件文本。
Step `id`、`name`、顺序、`executor`、Condition、Route 和 SkillBinding 均无新增、删除或变化。
不修改 Thrift IDL、生成代码、Go Runtime、State/Storage Schema、安装更新流程或公开 CLI。

生产 YAML 前后示例、完整边界和验收条件于 2026-08-24 由审批人确认；最终卡片结构见
`om_example_07`，包含需求确认完整计划的合并方案见
`om_example_03`，进入实现授权为
`om_example_04`。

2026-08-24 修订：固定人工审核群由 `oc_example_review_previous` 直接替换为
`oc_example_review`。不增加配置层或多群路由；审批人仍是唯一审核人，
关注人甲与关注人乙仍只作为同卡 cc，正文、mention、幂等和 `handoff.json` 契约不变。本修订由
审批人通过 `om_example_06` 批准进入实现。
