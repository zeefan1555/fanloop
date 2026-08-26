---
status: superseded by ADR-0069
date: 2026-08-22
amends: ADR-0066
---

# 维护者 MR 交接使用原生 Post 与权威话题链接

本决策的历史身份与内容保留；Card 2.0 交接由 ADR-0069 取代。

本 ADR 只修订 ADR-0066 的最终飞书交接段落。`fanloop-dev-mr-handoff` 仍在最终 reviewed HEAD 上
幂等 push 并按分支创建或更新唯一 MR；MR 描述、八步 Workflow、本地验证和代码审查语义均保持不变。

MR 存在后，handoff 使用当前 `fanloop-dev` 应用的 bot 身份调用 lark-cli，在固定群创建原生
`post` 首帖。首帖只包含七个固定顺序字段：需求背景、问题、解决手段、改动影响范围、MR链接、
关注人、上线时间。关注人真实 mention 张菲帆，并 cc 苏文钦和吴瑜明；未排定上线时间时固定写
“待终批，合并后随下一版本发布”。`fanloop-cr` 不进入首帖，在首帖成功后通过同一话题的独立
回复被真实 mention。

发送和回复使用由 Requirement、MR URL 与用途派生的稳定 idempotency key。同一 Requirement + MR
优先复用 `handoff.json` 中已验证且可访问的 root，并只补做缺失动作；仅 MR URL 变化、旧 root 不可访问
或证据丢失时新建话题并记录原因。交接后回读首帖和 CR 回复，验证 `msg_type=post`、目标 mentions 和
同话题关系，并从飞书响应取得权威 `message_app_link`，不得根据 ID 手工拼接。记录保存 Requirement、
MR URL、root/message/review message ID、权威链接与可选替换原因；发起会话最终收到该可点击链接。
每个验证通过的外部写立即原子 checkpoint：首帖 checkpoint 先持久化 root/message ID 和权威链接，
评审回复 checkpoint 再补入 review message ID。重试回读并复用已完成阶段，只补做后续缺失动作，不把
lark-cli 一小时幂等窗口当作跨进程持久化。

为覆盖发送成功但 checkpoint 前进程退出的窗口，每次外部写之前先原子记录 pending phase、精确正文、
目标、幂等键和尝试时间。重试 pending phase 时先完整分页读取目标群或话题，只按当前 bot、消息类型、
MR URL 与规范化正文的精确匹配恢复：唯一匹配则采用，多条匹配则 fail closed。没有匹配时，仅可在原
幂等键的一小时窗口内使用同一键重试；窗口已过则停止并要求人工核对，不冒险创建重复首帖或评审回复。

lark-cli 进程只使用当前 bot 的短期 tenant access token，通过环境变量启用 bot-only strict mode。宿主
受控凭据通道负责提供 token；Skill 不依赖 Botmux 私有模块、不修改全局 lark-cli profile，也不把 app
secret 或 token 写入 argv、日志、Issue Workspace 或 handoff 记录。身份不匹配，以及发送、消息类型、
mention、同话题或链接回读验证失败时全部 fail closed，不上报交接成功。

本 ADR 不修改五份生产 Workflow YAML、Step 的 id/name/顺序/executor、Condition、Route、Prompt
binding、Thrift IDL、Runtime、durable storage、发布/安装/更新流程或测试入口。ADR-0066 的其余决策及
ADR-0044、ADR-0047、ADR-0058、ADR-0059、ADR-0063 保持不变。
