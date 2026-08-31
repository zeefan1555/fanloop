---
name: fanloop-workflow
description: Fanloop 通用 Workflow/Loop 入口。适用于按显式场景启动或继续结构化流程、执行当前 Prompt、上报 Progress 或 Condition Result，以及选择前进、回流或终止 Route。
---

# Fanloop Workflow/Loop 推进

以 CLI 返回为唯一流程依据，持续执行：

**读取 Status → 执行 Prompt/Skills → 上报 Progress 或 Result → 读取新 Status**

`.fanloop` 由 CLI 管理。构造命令输入前读取目标叶子命令的 `--help`。新需求未指定目录时使用 `~/fanloop/worktrees/<requirement_slug>`；已有需求使用包含 `.fanloop` 的目录。

## 已初始化 Requirement 的固定控制器

解析出绝对 Requirement Root 后、第一次执行 `flow status` 前，依次检查：

1. `<ABSOLUTE_REQUIREMENT_ROOT>/bound-release-home/current/bin/fanloop`；
2. 已有维护需求的 `<ABSOLUTE_REQUIREMENT_ROOT>/issue-workspace/bound-release-home/current/bin/fanloop`。

首个存在的路径就是该 Requirement 的固定控制器。它后续的 `flow`、`trace`、`card` 和 `doctor`
命令都必须同时设置对应 `bound-release-home` 下的 `FANLOOP_DATA_HOME`，以及
`skill-roots/{codex,agent,trae,claude}` 四个 Skill Root；不得再调用全局 `fanloop`。先用该控制器执行
`flow status`，失败即原样阻塞，不得回退到全局 current、扫描其他 Release、改写 State 或猜测版本。

未找到固定控制器时才使用全局 `fanloop`。新 Requirement 的 `flow init` 始终使用全局 current；固定
控制器只保持已经初始化的 Requirement 继续运行，不影响新 Requirement 选择当前 Release。

## 启动

每次启动或继续 Requirement，先将 Requirement Root 解析为绝对路径并运行 `flow status`：

用户明确要求使用 Fanloop 或选择配置中的场景时表示要启动新流程；普通问答、分析或直接操作不自动进入 Workflow。

1. Status 返回已初始化 State 时，直接继续当前 Workflow，不执行 Release 更新。此前单独执行的 update 失败不阻止已有 Requirement 继续。
2. Status 返回 `NOT_INITIALIZED` 且用户要启动新流程时，完整读取并执行 [`ref/role.md`](ref/role.md)，再读取 [`routes.yaml`](routes.yaml)。只按用户显式选择的场景取得 Workflow ID，随后运行 `flow init`；用户尚未选择场景时展示配置中的可用场景并等待，不得初始化默认 Workflow。同一次新流程启动只执行一次选择。

Skill、CLI、场景配置或 init 任一不可用或失败时，原样报告阻塞并停止；不得降级为普通代码分析、文档生成或研发交付。

## 执行当前 Step

1. 使用启动阶段或上一次响应后读取的最新 `flow status`；仅按启动协议初始化新流程。
2. 只使用最新 `data.state.current`：读取扁平 `context`、`prompt`、`conditions`、`available_routes` 和已有 `outputs`。
3. 完整执行 `current.prompt`。每个结构化 Skill 都以 `path` 给出与当前运行 Release 匹配的绝对 `SKILL.md`；使用前必须完整读取该文件，不得按 `id` 去全局 Skills Root 猜测或搜索。依次使用 `optional=false` 的 Skills；`optional=true` 只在对应 Prompt 的条件成立时使用。`path` 缺失或不可读时停止执行并运行 `doctor`，不得 fallback 到其他同名 Skill。
4. 工作尚未形成退出结论时上报：

   `flow report progress --step-id <当前 Step ID> --status <in_progress|fixing|blocked> --summary <摘要> [--evidence '<JSON>']`

5. 形成真实结论后，从 `current.conditions[]` 选择原子 Condition，并按其 `output.type` 与约束构造：

   `flow report result --step-id <当前 Step ID> --condition-result '{"condition_id":"<ID>","output":{"type":"<TYPE>","value":<JSON>}}' [--condition-result ...] <--next-step-id ID|--back-step-id ID|--terminal> --summary <摘要> [--evidence '<JSON>']`

   `when.any_of` 外层是 OR、内层是 AND。提交一个完整组合；同一 `exclusive_group` 只选一个 Condition。Output key 由 Condition 定义，Agent 不提交 key 或 producer_step_id。Evidence 只用于审计，不参与路由。
6. 从最新 `available_routes` 选择一条满足该 Condition 组合的 Route，并把 `route` 原样表达为 `--next-step-id`、`--back-step-id` 或 `--terminal`。`direction=flow` 返回 `advanced|completed`；`direction=loop` 返回 `looped`，目标 Step 及其下游生产的 Output 失效。不要猜目标。
7. CLI 只在所选方向和目标内校验 `when.any_of` 唯一命中；未知 Route、事实与选择不一致、零命中或多命中都原子拒绝。
8. Human Step 使用同一个 Result 接缝。当前 Conditions 提供 `agent_approved` 且 Agent 已独立确认无阻塞项、无需人作出新决定时，可直接提交该 Condition 与 Route 要求的其他事实，不发布 Panorama；否则走人工路径。人工 Route 要求 `panorama_card_published` 时，先按其绑定 Skill 原样展示 renderer 生成的 Panorama；等到明确的人类决定后，把本次精确 `panorama_snapshot_path` 与 approved/rejected Condition、消息引用和 Evidence 一起上报。CLI 不自动发送，也不认证审批人或事实真伪。
9. 每次响应后重新读取 Status。命令错误不修改 State/Event；dry-run 返回计算结果但不写 Event、不触发远端投影。

## 人类提问顺序

需要人类澄清、确认或审批时：

1. 禁止调用 `botmux ask` 或其他结构化问答模式；问题只通过当前会话的普通回复发送。
2. 先完成承载最新状态的非 dry-run 写命令：新 Requirement 使用 `flow init`；Result 进入 Human Step 时复用该 `flow report result`；Agent Step 新增人工依赖时使用 `flow report progress --status blocked`。
3. Human Step 选择人工路径时，按当前 Prompt/Skill 展示审核材料，并按 Condition 绑定 Skill 原样展示 Panorama、保留本次 `panorama_snapshot_path` 后，才能请求人工决定；CLI 写命令不会代为发送。
4. 当前 State 已记录同一 blocked 事实时不重复上报；Human Step 重入仍必须生成新的 Panorama 快照。

State、Event、Bundle、Skill 或 Release 疑似不一致时运行 `doctor`。

## 最终普通回复

每次准备结束一轮普通回复时，先紧邻使用本轮已经解析的 Requirement 控制器执行以下两个参数序列；
`<REQUIREMENT_CONTROLLER>` 表示上文固定控制器及其五个环境变量，未固定时才表示全局 `fanloop`，不是
可省略的装饰占位符：

```bash
<REQUIREMENT_CONTROLLER> flow status --root <ABSOLUTE_REQUIREMENT_ROOT>
<REQUIREMENT_CONTROLLER> card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown --dry-run
```

成功后保留响应的 `data.content`；过程中的 commentary 和工具输出仅作中间反馈，本轮最终普通回复必须完整展示同一份 Panorama。不展示 JSON envelope，不自行拼装、压缩或重排内容。

任一命令失败即以真实错误阻塞并停止；不得手工 fallback、复用旧 render 或快照。

对用户只说明当前 Stage/Job/Step、已接受结果及下一项真实依赖。
