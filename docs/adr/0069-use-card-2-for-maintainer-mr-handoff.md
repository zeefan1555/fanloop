---
status: superseded by ADR-0071
date: 2026-08-23
amends: ADR-0066
supersedes: ADR-0068
amended_by: ADR-0070
---

# 维护者 MR 交接使用简洁 Card 2.0

本 ADR 取代 ADR-0068，并只修订 ADR-0066 的最终飞书交接段落。原生 Post 决策保留在原编号中作为
历史记录；`fanloop-dev-mr-handoff` 仍在最终 reviewed HEAD 上幂等 push 并按分支创建或更新唯一 MR；
八步 Workflow、本地验证和代码审查语义保持不变。

MR 描述面向不了解需求和实现背景的评审者。背景、问题和解决方案使用通俗、直接、可独立理解的
语言，必要术语或缩写首次出现时给出简短解释；改动点/影响范围、本地测试、ADR impact 和风险/非目标
仍是必填事实，不能为了简洁而省略。

MR 存在后，handoff 使用当前 `fanloop-dev` 应用的 bot 身份调用 lark-cli，在固定群创建
`interactive` Card 2.0 首帖。蓝色 header 显示 `MR Review 请求 · !编号`；正文依次展示安全的 MR
标题、真实 `fanloop-cr` mention 与文字 MR 链接、`变更概要` 和 `验证`；footer 真实 mention
张菲帆，并 cc 苏文钦和吴瑜明。`变更概要` 用一段短文压缩背景、问题、方法和影响范围，`验证`
只给本地验证与 AI Code Review 的简短结论。不使用按钮、回调、图片或 HTML，也不重复详细测试、
ADR、风险、非目标或上线时间。footer 如实说明 `fanloop-cr` 审查、张菲帆终批、终批后进入
Codebase merge queue，且不自动发布。

Card 2.0 的 body 不支持直接使用 `plain_text` 元素，因此动态 MR 标题、变更概要和验证摘要只进入
独立 `div.text.plain_text`；header title 继续使用其原生 `plain_text`。MR 编号从 Codebase 回读 URL
的数字路径段得到，文字链接只由静态 label、该编号和真实回读 URL 组成。只有 `fanloop-cr` 与三位
关注人进入固定 Markdown `<at>`。Card 对象与外层请求都通过标准 JSON 序列化器生成，不手工拼接，
因此动态标题或摘要不能改写链接、增加 mention 或改变卡片结构。

首帖使用由 Requirement、MR URL、模板版本与用途派生的稳定 idempotency key。外部写之前先原子记录
`schemaVersion=2`、`providerProjectionVersion=1`、`reviewedHead`、`root_send_pending` 和精确 Card 2.0
`rootContent`。Card 首次发送和更新后的 `config` 均显式设置 `update_multi=true`。同一 Requirement + MR
必须遵守
**Same MR, Same Thread**：已有可访问 Card root 时精确回读当前 Card JSON；内容变化则先记录
`root_update_pending`，再调用卡片专用的 `PATCH /open-apis/im/v1/messages/:message_id`，请求体只包含
序列化后的 `content`；成功后确认 root/message/thread ID 不变。普通消息的 PUT 编辑接口只支持 text/post，
不得用于 Card。PATCH 前确认消息仍在飞书 14 天更新窗口内且 Card 可更新；超龄、未启用 update_multi、
明确拒绝更新、既有 root 不可访问或候选不唯一时 fail closed，不自动创建替代 root。迁移需要新的具体
人工批准，不保留通用兼容回退。

唯一例外是已回读确认 `msg_type` 不是 `interactive` 的 legacy root。飞书不支持把该消息原地改成
Card 2.0，因此允许一次受控替换：先记录 `root_replacement_pending`、旧 root/thread、
`replacementReason=legacy_non_card_root`、目标 Card 与稳定 key；恢复或发送唯一新 Card root 并精确回读；
再在旧话题发送一条指向新权威链接的幂等重定向。只有新 root 与 redirect 都回读成功才完成；重试只
恢复同一 pending replacement，不得创建第三个 root。该例外不形成通用旧模板兼容或迁移层。

`root_send_pending` 恢复先完整分页查找发送时间之后的候选，只接受发送 bot、`msg_type=interactive`、
MR URL 与规范化 Card JSON 全部精确匹配的唯一消息。没有匹配时，仅可在原 key 的一小时窗口内用同一
key 和原始 Card 2.0 JSON 重试；多条匹配、窗口过期或仍无法唯一恢复时停止并要求人工核对。

恢复时把 pending 记录的不可变目标记为 A，把当前 MR 重新计算出的最新目标记为 B。三类 pending 都先
按原 key/body/root 恢复 A；A 与 B 不同则不得完成，而要在同一权威 Card root 原子进入新的
`root_update_pending(B)`，只有 B 回读通过才能写 complete。`root_replacement_pending` 先恢复 A 的新 root
与旧话题 redirect，再在该新 root 收敛到 B。该顺序覆盖 send/update/replacement 的外部写已生效和未生效
两种情况，并保证进程在两个 checkpoint 间再次退出时仍能从旧 pending 安全重放。

发送、复用或更新后，以 lark-cli 回读 `msg_type`、sender、mentions、thread ID 与权威链接，并通过
`botmux quoted <messageId> --raw` 的公开 CLI 输出取得原始 `cardJson`。提交值和回读值应用同一
provider-aware 安全投影：精确保留 schema、header、文本、mention、URL、列结构与元素顺序，只忽略飞书
生成的 element ID、未回显的 update_multi 和明确列举的默认布局字段；非默认值与所有未知字段都保留
比较，未知结构变化 fail closed。真实 provider 回读 fixture 进入聚焦契约测试。只有投影等于当前目标 B
才完成；pending A 只证明旧动作已恢复。

唯一的在途恢复例外是本次升级前已经落盘、尚无 `schemaVersion` / `providerProjectionVersion` 的
`root_replacement_pending`：若其 footer 明确写入 `text_size=notation|notation_small_v2`，而飞书回读只省略
该已知不受支持字段，可记录 `legacyProviderNormalization=footer_text_size_omitted` 并仅用该投影证明
pending A。该例外不得用于当前目标 B、新发送、新更新或其他字段；B 必须使用当前模板且不含
`text_size`，因此完成 checkpoint 后不保留通用兼容路径。

通过后先把同一 MR 的其他 complete handoff 记录逐个原子改为 `phase=superseded` 并指向当前
Requirement/root，再把当前记录原子写为 `schemaVersion=2`、`providerProjectionVersion=1`、
`phase=complete`，保存 reviewed HEAD、root/message/thread ID、当前目标 `rootContent` 与权威链接。常规
路径不保存 `replacementReason`；legacy replacement 额外保存 previous root/thread、固定 reason 和 redirect
message ID。该顺序允许崩溃时暂时没有 complete，但不允许留下两个 complete；重试从原 pending 记录补完。

lark-cli 进程只使用当前 bot 的短期 tenant access token，通过环境变量启用 bot-only strict mode。宿主
受控凭据通道负责提供 token；Skill 不依赖 Botmux 私有模块、不修改全局 lark-cli profile，也不把 app
secret 或 token 写入 argv、日志、Issue Workspace 或 handoff 记录。Card 原始结构只复用 Botmux 已有的
`quoted --raw` 公开读能力。身份、发送、消息类型、mention、链接或记录验证失败时全部 fail closed，
不上报交接成功。

本 ADR 修改 Issue Workspace 中 `handoff.json` 的 `rootContent` 消息契约，并同步收紧
`fanloop-maintainer/prompt.yaml` 的 `handoff_record_written` 成功条件。五份生产 Workflow 中只有 Prompt
文本变化；Step 的 id/name/顺序/executor、Condition ID/type、Route 与 SkillBinding 均不变。不修改
Thrift IDL、Runtime 核心 State/Storage Schema、发布/安装/更新流程或测试入口。ADR-0066 的其余决策及
ADR-0037、ADR-0044、ADR-0047、ADR-0058、ADR-0059、ADR-0063 保持不变。
