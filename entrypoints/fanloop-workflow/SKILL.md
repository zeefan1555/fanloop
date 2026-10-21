---
name: fanloop-workflow
description: Fanloop 通用 Workflow/Loop 入口。适用于按显式场景启动或继续结构化流程、执行当前 Prompt、上报 Progress 或 Condition Result，以及选择前进、回流或终止 Route。
---

# Fanloop Workflow/Loop 推进

以 CLI 返回为唯一流程依据，持续执行：

**读取 Status → 执行 Prompt/Skills → 上报 Progress 或 Result → 读取新 Status**

`.fanloop` 由 CLI 管理。构造命令输入前读取目标叶子命令的 `--help`。新需求未指定目录时使用 `~/fanloop/workspace/<workflow-id>/<requirement-root>`，选定 Workflow 后解析为绝对路径；已有需求使用包含 `.fanloop` 的原绝对目录继续。

## 已初始化 Requirement 的固定控制器

解析出绝对 Requirement Root 后、第一次执行 `flow status` 前，检查
`<ABSOLUTE_REQUIREMENT_ROOT>/bound-release-home/current/bin/fanloop`。该路径存在时就是该 Requirement
的固定控制器。它后续的 `flow`、`trace`、`card` 和 `doctor`
命令都必须设置对应 `bound-release-home` 下的 `FANLOOP_DATA_HOME` 和
`skill-roots/{codex,agent,trae,claude}` 四个 Skill Root，并用 `env -u FANLOOP_CONFIG_SOURCE`
清除外部来源覆盖，由固定控制器按其自身绑定解析配置；不得硬编码新旧配置布局，也不得再调用全局 `fanloop`。先用该控制器执行
`flow status`，失败即原样阻塞，不得回退到全局 current、扫描其他 Release、改写 State 或猜测版本。

未找到固定控制器时才使用全局 `fanloop`。新 Requirement 的 `flow init` 始终使用全局 current；固定
控制器只保持已经初始化的 Requirement 继续运行，不影响新 Requirement 选择当前 Release。

## 启动

每次启动或继续 Requirement，先将 Requirement Root 解析为绝对路径；新 Root 先用 `mkdir -p` 创建目录，再运行 `flow status`：

用户明确要求使用 Fanloop 或选择配置中的场景时表示要启动新流程；普通问答、分析或直接操作不自动进入 Workflow。

新普通 Skill 变更使用 `write-skill`；新 Workflow 包变更使用 Runtime `manage-fanloop-workflow`。只有独立绝对 Root 的 Status 返回 `NOT_INITIALIZED` 时才执行 `fanloop flow init --root <ABSOLUTE_REQUIREMENT_ROOT> --workflow <对应 ID> --title <非空标题>` 并回读 Status，按其当前绑定步骤执行验证、审核和发布。已有 Run 直接继续，其他错误先解决。旧维护只继续已绑定的旧 Run。维护 Fanloop 代码或混合实现时使用 `code-change` Workflow，Define按实际范围选择通道。
生产配置真值是 second-brain-skills main 的 `fanloop-workflow/<workflow-id>/` 完整包。新 flow init 自动拉取并原子激活不可变提交快照，即使存在 `fornax.json`；拉取失败且已有快照时继续使用该快照。`fanloop update config` 也从这条 GitHub main 同步；已有快照 commit 相同则跳过 checkout。已有 flow status/report/skills 离线读取固定快照。`FANLOOP_USE_FORNAX=1` 且 `fornax.json` 是普通文件时，init、update config 和 config drift 才改用 Fornax。显式绝对 FANLOOP_CONFIG_SOURCE 指向作者仓库根并跳过刷新，update config 与 config drift 会拒绝它。新的非 dry-run `flow init` 还会尝试把 CLI 更新到最新 GitHub Release，失败只告警并继续使用已安装版本。Dashboard 默认只读快照，显式作者源可编辑。每次读取最新 Status 的精确 Skill 路径。

1. Status 返回已初始化 State 时，直接继续当前 Workflow，不执行 Release 更新。
2. Status 返回 `NOT_INITIALIZED` 且用户要启动新流程时，完整读取并执行 [`ref/role.md`](ref/role.md)。已确认为普通 Skill 或 Workflow 包维护的需求使用上面的对应 ID；其他流程读取 [`routes.yaml`](routes.yaml)，按用户显式选择的场景取得 ID 再运行 `flow init`。用户尚未选择目标时展示可用场景并等待，不得初始化默认 Workflow。同一次新流程启动只执行一次选择。

Skill、CLI、场景配置或 init 任一不可用或失败时，原样报告阻塞并停止；不得降级为普通代码分析、文档生成或研发交付。

## 执行当前 Step

1. 使用启动阶段或上一次响应后读取的最新 `flow status`；仅按启动协议初始化新流程。
2. 只使用最新 `data.state.current`：读取扁平 `context`、`execution.status`、`prompt`、`conditions`、`common_skills`、`common_conditions`、`available_routes` 和已有 `outputs`。
3. 每个结构化 Skill 都以 `path` 给出当前 live Workflow 目录中的绝对 `SKILL.md`；每次 Status 后都重新读取，不缓存旧内容，不得按 `id` 去全局 Skills Root 猜测或搜索。依次使用 `optional=false` 的 Skills；`optional=true` 只在对应 Prompt 的条件成立时使用。Panorama 与其他能力一样，只在当前 Prompt、Condition 或 Route 的绑定要求它时执行，不按 Condition ID 注入额外时机。`path` 缺失或不可读时停止执行并运行 `doctor`，不得 fallback 到 Release 副本或其他同名 Skill。
4. `execution.status=awaiting_confirmation` 时，先完整执行 `current.common_skills` 中必需的 Skill，但不执行业务 `current.prompt`；只能从 `current.common_conditions` 取得唯一 Condition，并选择 `start_current_step` 或 `jump_step_id` Route。必须等待针对当前 Step 和本次材料的全新、明确人类确认；早先的笼统授权不得代替逐 Step 确认。确认开工时使用 `--start-current-step`；确认跳转时使用 `--jump-step-id <ID>`。
5. `execution.status=in_progress|ready|fixing|blocked` 时，完整执行 `current.prompt`；若人类明确要求跳转，仍执行 `current.common_skills` 对应的跳转 Skill，从 `current.common_conditions` 上报唯一 Condition 并使用 `--jump-step-id <ID>`。
6. 工作尚未形成退出结论时上报：

   `flow report progress --step-id <当前 Step ID> --status <in_progress|fixing|blocked> --summary <摘要> [--evidence '<JSON>']`

7. 形成真实结论后，从 `current.conditions[]` 选择原子 Condition，并按其 `output.type` 与约束构造：

   `flow report result --step-id <当前 Step ID> --condition-result '{"condition_id":"<ID>","output":{"type":"<TYPE>","value":<JSON>}}' [--condition-result ...] <--next-step-id ID|--back-step-id ID|--terminal> --summary <摘要> [--evidence '<JSON>']`

   `when.any_of` 外层是 OR、内层是 AND。提交一个完整组合；同一 `exclusive_group` 只选一个 Condition。Output key 由 Condition 定义，Agent 不提交 key 或 producer_step_id。Evidence 只用于审计，不参与路由。
8. 从最新 `available_routes` 选择一条满足该 Condition 组合的 Route，并把 `route` 原样表达为 `--next-step-id`、`--back-step-id`、`--terminal`、`--start-current-step` 或 `--jump-step-id`。`direction=flow` 返回 `advanced|completed|started`；`direction=loop` 返回 `looped`；Jump 返回 `jumped`，按 Status 报告的目标和跳过项继续。不要猜目标。
9. CLI 只在所选方向和目标内校验 `when.any_of` 唯一命中；未知 Route、事实与选择不一致、零命中或多命中都原子拒绝。
10. Human Step 使用同一个 Result 接缝。按当前 Prompt/Skills 准备材料并等待全新、明确的人类决定；沉默、含糊回复、机器人结论或 Agent 自批均无效。CLI 不自动发送，也不认证审批人或事实真伪。
11. 每次响应后重新读取 Status。terminal Result 被接受后没有 `current`；只执行 terminal 前已读取的 Workflow Skill 明确声明的 post-terminal 动作，不从 Runtime 或 Condition ID 推断额外动作。命令错误不修改 State/Event；dry-run 返回计算结果但不写 Event、不触发远端投影。

## 按需发现整套 Skill

当前 Step 的必需 Skill 以最新 Status 的绑定为准。需要读取其他通用 Skill，或已读 Skill 引用了本步骤未绑定的 Skill 时，先读 `fanloop flow skills --help`，再使用同一 Requirement 控制器执行 `fanloop flow skills --root <ABSOLUTE_REQUIREMENT_ROOT>`。目录返回整包全部 Skill 的 ID、可为空的 description、绝对 SKILL.md 路径以及 package_root / skill_root；按任务和引用选择并读取，资源相对所选 Skill 目录定位。目录不是必读清单，也不要求读取凭证。每次需要刷新目录时重新查询当前绑定源；不要猜全局同名路径。

## 人类提问顺序

需要人类澄清、确认或审批时：

1. 禁止调用 `botmux ask` 或其他结构化问答模式；问题只通过当前会话的普通回复发送。
2. 先完成承载最新状态的非 dry-run 写命令：新 Requirement 使用 `flow init`；Result 进入 Human Step 时复用该 `flow report result`；Agent Step 新增人工依赖时使用 `flow report progress --status blocked`。
3. Human Step 选择人工路径时，按当前 Prompt/Skill 准备审核材料；若 Workflow 自己要求 Panorama，则按其绑定 Skill 指定的时机投递。CLI 写命令不会代为发送。
4. 当前 State 已记录同一 blocked 事实时不重复上报。

State、Event、Bundle、Skill 或 Release 疑似不一致时运行 `doctor`。

## 最终普通回复

最终回复执行当前 Workflow 已读取 Prompt/Skill 明确声明的收尾动作，然后说明已接受结果和下一项真实依赖。入口不按 Condition ID 自动 render、重复或禁止 Panorama。
