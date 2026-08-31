---
name: fanloop-dev-workflow
description: 维护 zeefan1555/fanloop 自身的入口。沿 fanloop-maintainer 从本地验证、功能图谱、独立 Eval、CI 和机器人验收推进到自动合码。
---

# Fanloop Dev Workflow

维护 `zeefan1555/fanloop` 时，以 CLI 返回为唯一流程依据，持续执行：

**读取 Status -> 执行 Prompt/Skills -> 上报 Progress 或 Result -> 读取新 Status**

`.fanloop` 由 CLI 管理。构造命令输入前读取目标叶子命令的 `--help`。新 Requirement
使用 `~/fanloop/issues/<issue-slug>`；已有 Requirement 使用包含 `.fanloop` 的目录。

已有 Requirement 必须先执行通用 `fanloop-workflow` 的“固定控制器”解析，再读取 Status。存在
`bound-release-home` 后，本维护流程自身的 Status、Progress、Result、Doctor 与最终 Card 都使用该
控制器；全局 `$HOME/.fanloop/current` 只供候选安装、两个机器人和后续新 Requirement 使用。固定
控制器失败时原样阻塞，禁止以全局 current 重试或修改 State 绕过 `WORKFLOW_MISMATCH`。

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
8. `confirm_requirements` 是 Agent Step。Agent 已独立确认无阻塞项、无需人作出新决定时，提交
   `agent_approved` 与 Route 要求的其他事实，不发布 Panorama；否则保持 blocked 并请求真实产品
   决策。补充人工 Route 要求 `panorama_card_published` 时，先按其绑定 Skill
   原样展示 renderer 生成的 Panorama；等到明确的人类决定后，把本次精确
   `panorama_snapshot_path` 与 approved/rejected Condition、消息引用和 Evidence 一起上报。
   CLI 不自动发送，也不认证审批人或事实真伪。
9. 每次响应后重新读取 Status。命令错误不修改 State/Event；dry-run 返回计算结果但不写 Event、
   不触发远端投影。

## 需求确认

confirm_requirements 由 Agent 主动复核，同时保留需要真实产品决策时的人工补充路径。

Agent 路径先独立复核最新 `requirements.md`。只有所有决策明确、Open Questions 为空、完整改造
计划与验证边界自洽且不存在需要人决定的阻塞项时，才同时上报 `agent_approved` 与
`implementation_required` 或 `implementation_not_required`；Evidence 记录复核理由和
`requirements.md`。该路径不发送 Panorama，不构造人工消息或回执。

不能独立批准时进入人工路径：从最新 `requirements.md` 生成一张自包含 Markdown 审核卡并使用
`botmux send --mention-back` 发送。确认卡必须依次完整展示当前 Stage/Job/Step、目标、现状问题、逐项改造、影响文件/契约、保持不变与非目标、验证计划、交付边界和精确授权口令；不得只给摘要、只贴链接或要求人回翻上下文。
发送并回读成功后立即把 Botmux `messageId` 写入 `requirements.md`，并以该审核卡建立
turn boundary。另按 `panorama_card_published` 绑定 Skill 原样展示紧凑 Panorama，该
Condition 只输出本次 `panorama_snapshot_path`：
只接受此后到达的全新用户消息，不得复用进入 Human Step 前或计划卡发送前的消息。

选择人工补充路径前，在 `requirements.md` 记录 `expectedApprover`：当前 `fanloop-dev` 应用
`cli_aaf6cd8160b89bda` 作用域下的张菲帆 OpenID
`ou_3b0b9cf8364168c5eb999bd6c5a33b95`。收到消息后用 `botmux quoted <message_id> --raw` 或当前
话题的 `botmux history` 回读元数据；只有 `senderType=user`、`senderId` 精确等于该 OpenID，且消息在
turn boundary 之后到达，才继续检查正文。不能只校验显示名，也不能使用其他应用作用域的 OpenID。

审核卡与 Panorama 展示成功后，需要实现时，正文去除首尾空白后必须精确等于 `批准进入 需求实现`。
`同意`、`按这个改`、方案反馈、
字段调整，以及同时包含批准与需求修改的消息都不构成开发授权；错误 sender 即使发送精确短语也无效。
这些消息必须连同真实 message ID、senderType 与 senderId 记录后回到需求澄清。形成有效批准时，先把
审核卡 messageId、Panorama 快照路径、批准消息 messageId、senderType、senderId、正文结论和实现必要性写入
`requirements.md`，再将 `requirements_evidence_written=requirements.md` 与其他成功 Conditions 一起上报。
有效 Agent 或人工批准 Result 被 CLI 接受且最新 Status 已进入 `design_technical_solution` 前，不得修改源码、提交、推送、创建或更新 PR。
该批准不替代后续 Agent 验收或合码事实。

## 维护者协作边界

需求确认后自动推进。Review 冻结 `candidate_head` 后源码只读：协调者冻结 Case，多个隔离候选并行
执行，不同模型裁判必须 10/10；随后发布唯一 PR 并校验 Ruleset 与该 SHA 的 required checks。

execute_agent_acceptance 只允许“使用 Fanloop 机器人”在固定群驱动“FanLoop 机器人”，校验内容寻址
Playbook 与两个 brief/Rubric 摘要后，把两个冻结原始 brief 直接派发到全新话题、目录和 Requirement；
禁止生成、复制、选择或改题。派发前必须从 `candidate_head` 的干净工作树执行真实
`npm run install:local`，清除 `FANLOOP_DATA_HOME` 与 Skill Root 覆盖，并以默认
`$HOME/.fanloop/current/bin/fanloop` 的 commit 和 Doctor 回读为准；临时候选 bin 或隔离安装不能代替。
外层 Botmux 只通信；内层 Fanloop CLI 清除 Botmux 环境，不使用用户身份，
不生成 Card Binding、Trace Integration、远端 Trace Event 或用户文档。Candidate 到达 merge_code 前停止。

只有 merge_code 可对唯一 PR 使用 `--auto --squash --match-head-commit`；不发送 MR 交接、不等待人工
端到端验收、不使用 `--admin` 或直接 push main。main push 的 Release 由 GitHub Actions 独立处理。

## 人类提问顺序

需要人类澄清、确认或审批时：

1. 禁止调用 `botmux ask` 或其他结构化问答模式；问题只通过当前会话的普通回复发送。
2. 先完成承载最新状态的非 dry-run 写命令：新 Requirement 使用 `flow init`；Result 进入
   Human Step 时复用该 `flow report result`；Agent Step 新增人工依赖时使用
   `flow report progress --status blocked`。
3. 需求确认选择人工补充路径时，按当前 Prompt/Skill 展示审核材料，并按 Condition 绑定 Skill
   原样展示 Panorama、保留本次 `panorama_snapshot_path` 后，才能请求人工决定；CLI 写命令不会代为发送。
4. 当前 State 已记录同一 blocked 事实时不重复上报；Human Step 重入仍必须生成新的 Panorama
   快照。

State、Event、Bundle、Skill 或 Release 疑似不一致时运行 `doctor`。

## 最终普通回复

专用维护入口不得绕过通用 `fanloop-workflow` 的 renderer-owned 最终回复契约。每次准备结束一轮
普通回复时，先紧邻执行：

```bash
fanloop flow status --root <ABSOLUTE_REQUIREMENT_ROOT>
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown --dry-run
```

成功后，本轮最终普通回复必须完整原样展示 render 响应的 `data.content`；不展示 JSON envelope，
不自行拼装、压缩或重排内容。任一命令失败即以真实错误阻塞并停止；不得手工 fallback、复用旧 render 或快照。

Panorama 之外只说明当前 Stage/Job/Step、已接受结果及下一项真实依赖。
