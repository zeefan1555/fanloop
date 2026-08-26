---
name: fanloop-dev-workflow
description: 维护 zeefan1555/fanloop 自身的入口。用于自我迭代机器人接收缺陷、优化或代码变更请求，并沿 fanloop-maintainer Workflow 从需求澄清推进到最终 MR 交接。
---

# Fanloop Dev Workflow

维护 `zeefan1555/fanloop` 时，以 CLI 返回为唯一流程依据，持续执行：

**读取 Status -> 执行 Prompt/Skills -> 上报 Progress 或 Result -> 读取新 Status**

`.fanloop` 由 CLI 管理。构造命令输入前读取目标叶子命令的 `--help`。新 Requirement
使用 `~/fanloop/issues/<issue-slug>`；已有 Requirement 使用包含 `.fanloop` 的目录。

## 启动

每次启动或继续 Requirement，先将 Requirement Root 解析为绝对路径并运行 `flow status`：

## 按 Session 解析 Requirement Root

当调用上下文尚未给出已初始化的 Requirement Root 时，在生成 issue slug 或执行
`flow init` 前，先读取当前 `BOTMUX_SESSION_ID`，逐个解析
`~/fanloop/issues/*/.fanloop/card/config.json` 并精确比较 `session_id`：

1. 唯一命中时，将该配置所在 Requirement 作为唯一 Root，立即运行 `flow status`。
   运行中 State 直接继续；不得重新选择 Workflow 或初始化新目录。
2. 零命中时，才按下文的新 Requirement 协议生成 Root 并 init。
3. 多命中时，停止并报告所有冲突 Root；不猜测、不清理、不创建第三个 Root。
4. 唯一命中已完成 Requirement 时，停止并要求在新话题/Session 启动下一项需求；
   不创建新的 Requirement Root。

该查找只复用 Card Binding 已有事实，不修改配置、不建立全局 Registry，也不合并多个
Requirement。`BOTMUX_SESSION_ID` 不存在或为空时没有可精确匹配的 Session，按零命中处理。

收到 Fanloop CLI 的缺陷、优化或代码变更请求默认表示启动新维护流程；只有用户明确要求仅
回答且不进入研发流程时才跳过。

1. Status 返回已初始化 State 时，直接继续其固定的 `fanloop-maintainer` Workflow，不更新 Release
   或重新选择 Workflow。
2. 仅当 Status 返回 `NOT_INITIALIZED` 且用户要启动新流程时，完整读取并执行
   [`ref/role.md`](ref/role.md)，再使用当前配套 Release 的固定 Workflow ID 初始化：

   ```bash
   fanloop flow init --root <ABSOLUTE_ROOT> --workflow fanloop-maintainer ...
   ```

   同一次新流程启动只初始化一次；不在线解析或切换 npm `latest`。

Skill、CLI 或 init 任一不可用或失败时，原样报告阻塞并停止；不得降级为普通代码
分析、文档生成或研发交付。

## 执行当前 Step

1. 使用启动阶段或上一次响应后读取的最新 `flow status`；仅按启动协议初始化新流程。
2. 只使用最新 `data.state.current`：读取扁平 `context`、`prompt`、`conditions`、
   `available_routes` 和已有 `outputs`。
3. 完整执行 `current.prompt`。每个结构化 Skill 都以 `path` 给出与当前运行 Release 匹配的
   绝对 `SKILL.md`；使用前必须完整读取该文件，不得按 `id` 去全局 Skills Root 猜测或搜索。
   依次使用 `optional=false` 的 Skills；`optional=true` 只在对应 Prompt 的条件成立时使用。
   `path` 缺失或不可读时停止执行并运行 `doctor`，不得 fallback 到其他同名 Skill。
4. 工作尚未形成退出结论时上报：

   ```bash
   fanloop flow report progress --step-id <当前 Step ID> --status <in_progress|fixing|blocked> --summary <摘要> [--evidence '<JSON>']
   ```

5. 形成真实结论后，从 `current.conditions[]` 选择原子 Condition，并按其 `output.type` 与约束
   构造：

   ```bash
   fanloop flow report result --step-id <当前 Step ID> --condition-result '{"condition_id":"<ID>","output":{"type":"<TYPE>","value":<JSON>}}' [--condition-result ...] <--next-step-id ID|--back-step-id ID|--terminal> --summary <摘要> [--evidence '<JSON>']
   ```

   `when.any_of` 外层是 OR、内层是 AND。提交一个完整组合；同一 `exclusive_group` 只选一个
   Condition。Output key 由 Condition 定义，Agent 不提交 key 或 producer_step_id。Evidence
   只用于审计，不参与路由。
6. 从最新 `available_routes` 选择一条满足该 Condition 组合的 Route，并把 `route` 原样表达为
   `--next-step-id`、`--back-step-id` 或 `--terminal`。`direction=flow` 返回
   `advanced|completed`；`direction=loop` 返回 `looped`，目标 Step 及其下游生产的 Output
   失效。不要猜目标。
7. CLI 只在所选方向和目标内校验 `when.any_of` 唯一命中；未知 Route、事实与选择不一致、零命中
   或多命中都原子拒绝。
8. Human Step 使用同一个 Result 接缝。等到明确的人类决定后，上报 approved/rejected
   Condition、消息引用与 Evidence；CLI 校验格式和 Route，不认证审批人或事实真伪。
9. 每次响应后重新读取 Status。命令错误不修改 State/Event；dry-run 返回计算结果但不写 Event、
   不触发远端投影。

## 开发授权门禁

`confirm_requirements` 是进入需求实现前的唯一 Human Step。Human Step 的 Panorama Card 投递完成后，
从最新 `requirements.md` 生成一张自包含 Markdown 确认卡并使用
`botmux send --mention-back` 发送。确认卡必须依次完整展示当前 Stage/Job/Step、目标、现状问题、逐项改造、影响文件/契约、保持不变与非目标、验证计划、交付边界和精确授权口令；不得只给摘要、只贴链接或要求人回翻上下文。
发送成功后立即把 Botmux `messageId` 写入 `requirements.md`，并以该完整改造计划卡建立 turn boundary：
只接受此后到达的全新用户消息，不得复用进入 Human Step 前或计划卡发送前的消息。

进入 Human Step 前，在 `requirements.md` 记录 `expectedApprover`：当前 `fanloop-dev` 应用
`cli_aaf6cd8160b89bda` 作用域下的张菲帆 OpenID
`ou_3b0b9cf8364168c5eb999bd6c5a33b95`。收到消息后用 `botmux quoted <message_id> --raw` 或当前
话题的 `botmux history` 回读元数据；只有 `senderType=user`、`senderId` 精确等于该 OpenID，且消息在
turn boundary 之后到达，才继续检查正文。不能只校验显示名，也不能使用其他应用作用域的 OpenID。

完整改造计划卡发送成功后，需要实现时，正文去除首尾空白后必须精确等于 `批准进入 需求实现`。
`同意`、`按这个改`、方案反馈、
字段调整，以及同时包含批准与需求修改的消息都不构成开发授权；错误 sender 即使发送精确短语也无效。
这些消息必须连同真实 message ID、senderType 与 senderId 记录后回到需求澄清。形成有效批准时，先把
确认卡 messageId、批准消息 messageId、senderType、senderId、正文结论和实现必要性写入
`requirements.md`，再将 `requirements_evidence_written=requirements.md` 与其他成功 Conditions 一起上报。
有效授权 Result 被 CLI 接受且最新 Status 已进入需求实现前，不得修改源码、提交、推送、创建或更新 MR。

## 维护者协作边界

需求确认后自动推进。最终 MR 交接前只与张菲帆交互，不通知其他机器人。最终 MR 交接 Step 按当前
Release-bound Skill 在固定审核群创建话题，真实 @ 张菲帆进行人工审核，并在同一卡片末行 cc
苏文钦、吴瑜明。

不自动合并或发布；最终决定由张菲帆完成。

## 人类提问顺序

需要人类澄清、确认或审批时：

1. 禁止调用 `botmux ask` 或其他结构化问答模式；问题只通过当前会话的普通回复发送。
2. 先完成承载最新状态的非 dry-run 写命令：新 Requirement 使用 `flow init`；Result 进入
   Human Step 时复用该 `flow report result`；Agent Step 新增人工依赖时使用
   `flow report progress --status blocked`。
3. 只有对应命令的 Panorama Card 投递尝试完成并返回后，才能在本轮最终普通回复中提出问题；
   投递前的进度消息不得包含问题。
4. 当前 State 已记录同一 blocked 事实时不重复上报，直接在已有 Panorama Card 之后继续普通
   回复。

State、Event、Bundle、Skill 或 Release 疑似不一致时运行 `doctor`。

对用户只说明当前 Stage/Job/Step、已接受结果及下一项真实依赖。
