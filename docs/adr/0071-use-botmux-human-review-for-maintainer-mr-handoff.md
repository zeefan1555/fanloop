---
status: superseded by ADR-0090
date: 2026-08-24
amends: ADR-0066
supersedes: ADR-0069, ADR-0070
---

# 维护者 MR 交接使用 Botmux 人工审核话题

`fanloop-maintainer` 保留 ADR-0066 的三阶段八步拓扑、本地验证、AI Code Review 和以 MR 交接为
terminal 的边界。最终交接不再触发独立审查机器人或自动 merge queue；MR 由张菲帆人工审核，
不自动合并或发布。

MR 创建或更新后，`fanloop-dev-mr-handoff` 把 UTF-8 Markdown 正文写到 Issue Workspace，并执行：

```bash
botmux send --top-level --chat-id oc_0a667d0df3a67ec0d969f12524c94f13 \
  --mention ou_3b0b9cf8364168c5eb999bd6c5a33b95:张菲帆 \
  --mention ou_fdc66f7d48be8c75fa926e0ec27ee809:苏文钦 \
  --mention ou_0cc015d1f9aadf74f752989a0992b869:吴瑜明 \
  --content-file <issue-workspace>/handoff-topic.md
```

张菲帆是唯一审核人；苏文钦、吴瑜明只作为 cc 关注人。三人都在同一卡片中真实 mention，cc 位于
影响范围后的末行，固定写为 `cc：@苏文钦 @吴瑜明`。全角冒号确保首个 `@` 符合 Botmux 的内联
mention 边界，三个显式 mention 均由正文消费，不产生 `发送给` 尾栏，也不另发消息。正文包含 MR 标题、编号和真实链接，
并按固定顺序提供四个业务区块：
背景、问题、解法、影响范围。详细测试、ADR impact、风险和非目标继续保留在 MR 描述，不复制到话题。

成功回执以 `botmux send` 的零退出码及 JSON 中的 `success=true`、非空 `messageId`、非空 `sessionId`
和张菲帆、苏文钦、吴瑜明三个准确 mention 为准。Issue Workspace 的 `handoff.json` 只保存 `phase=complete`、Requirement、
MR URL、reviewed HEAD、messageId 和 sessionId。相同 MR URL 与 reviewed HEAD 已完成时直接复用记录，
不重复发送。旧 Card 或 merge queue 记录不迁移、不兼容；遇到其他格式时停止并报告。

本决策删除 `fanloop-cr-review` Skill、verdict 模板、Card provider fixture，以及 Card 2.0、lark-cli、
provider projection、legacy replacement 和自动 merge queue 的维护契约。ADR-0069 与 ADR-0070 因此被
完整取代；ADR-0068 已由 ADR-0069 取代，继续作为历史记录保留。

五份生产 Workflow 中只修改 `prompt.yaml` 的 MR 交接 Prompt 与 handoff record 条件文本。
Step `id`、`name`、顺序、`executor`、Condition、Route 和 SkillBinding 均无新增、删除或变化。
不修改 Thrift IDL、生成代码、Go Runtime、State/Storage Schema、安装更新流程或公开 CLI。

生产 YAML 前后示例、完整边界和验收条件于 2026-08-24 由张菲帆确认；最终卡片结构见
`om_x100b67894593cca8b3c04d2eb273c11`，包含需求确认完整计划的合并方案见
`om_x100b678964a1a4bcb25236829136cf2`，进入实现授权为
`om_x100b678961cefcbcb306ab63ae0d4a3`。

2026-08-24 修订：固定人工审核群由 `oc_0a667d0df3a67ec0d969f12524c94f13` 直接替换为
`oc_9f25fc928e2e5a6a602e58fa80b4750a`。不增加配置层或多群路由；张菲帆仍是唯一审核人，
苏文钦与吴瑜明仍只作为同卡 cc，正文、mention、幂等和 `handoff.json` 契约不变。本修订由
张菲帆通过 `om_x100b678abaa41ca0b3018e94e33bb91` 批准进入实现。
