# Flow 创建及 Step 迁移后 best-effort 发送 Panorama Card

`flow init` 在本地 `flow.initialized` Event 成功提交后发送首张 Panorama Card。之后，被接受的 `flow report --type decision` 在本地 `flow.reported` Event 成功提交后，如果 effect 是 `advanced`、`looped` 或 `completed`，每次自动渲染现有 `panorama + lark_json` Card 快照，并向 Requirement 绑定的 Botmux 目标同步尝试发送一次。`progress`、`rejected` 和 dry-run 不触发。

发送绑定保存在 `.fanloop/card/config.json`，只包含 `schema_version`、`chat_id` 和 `session_id`。配置不存在时，运行时优先在 `flow init` 中从同时存在的 `BOTMUX_CHAT_ID` 与 `BOTMUX_SESSION_ID` 捕获一次；已有配置不会因后续命令来自另一个 Session 而隐式改绑。发送时只用绑定的 `session_id` 恢复原话题，不传 `--top-level` 或 `--chat-id` 改写路由。配置不是 Flow State 或投递状态，不改变公开 IDL、State/Event Schema。

本地 Flow commit 是成功边界。绑定、Card 渲染或 Botmux 发送失败只写 warning，不回滚已经成立的 Flow/Step，不改变 `FlowInitResponse`、`FlowReportResponse` 或成功退出码，也不自动重试。sender 不回显 Botmux stdout/stderr 或绑定内容；timeout warning 明确远端结果未知。Card 仍由 renderer 负责确定性内容和不可变快照，sender 只消费本次返回的精确 `snapshot_path`。Doctor 将精确的 `config.json` 与 Card 快照分开严格校验，其他未知 Card 文件继续视为损坏。

该方案只保证 Flow 创建及每次 Step 迁移在进程存活时 best-effort 尝试一次，不保证最终必达、at-least-once 或 exactly-once。只有明确提出崩溃补发、无人值守重试或投递审计后，才引入 transactional outbox 与接收方幂等键。本决策取代 ADR-0003、ADR-0033 和 ADR-0034 中“Flow 初始化/决策不会编排 Card 渲染/发送”的部分；保留 Card renderer 与 sender 分离、不可变快照以及本地事实先于远端副作用的边界。

2026-08-24 修订：Release-bound Maintainer Skill 可以在创建新 Requirement 前只读
`.fanloop/card/config.json` 中已有的 `session_id`，用于定位并复用同一 Botmux Session
唯一的 Requirement Root。该用途不改变 Card Binding Schema、写入时机、不可隐式改绑语义或
CLI Runtime；精确选择与冲突边界见 ADR-0072。
