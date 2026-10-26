# Fanloop 变更门禁

## 本地材料不入库

1. `.scratch/`、`docs/research/` 和 `docs/specs/` 仅可作为本地临时材料，永远不得纳入版本控制，也不得使用 `git add -f` 绕过忽略规则。
2. 不得因历史引用或旧流程恢复这些目录中的已删除内容；确需临时查阅时从 Git 历史读取，且不重新提交。
3. 提交前检查这三个路径在暂存区中只能出现删除记录；一旦出现新增或修改，立即停止提交并移出暂存区。

## 已确认决策沉淀

触发条件：需求经过 `/grill-with-docs`、飞书评审或对话澄清。

1. 编码前，把已确认的决策整理为决策、约束和验收条件。代码行为契约落在现有 Thrift IDL 或契约测试；生产配置契约随仓库 `fanloop-workflow/<workflow-id>/` 完整包提交，并记录 Git 基线与文件摘要；长期且难以逆转的架构选择进入 `docs/adr/`。研究与任务拆分留在仓库外或上述本地临时目录中。
2. 只沉淀用户明确确认的结论，或用户授权范围内经主 Agent 完整审阅并记录明确决定的结论。开放问题继续保留为开放问题，未审阅的推荐项不自动成为决策。
3. 让代码契约真值与实现处于同一分支，并在实现提交之前或同一提交中纳入 Git；外部文档、聊天记录和未跟踪文件不能作为唯一事实源。

完成标准：逐项对照澄清来源与目标分支相对 merge base 的 diff；每个已确认的行为、边界和验收条件都能定位到已跟踪文件。

## 代码与配置分离

先按实际改动区分配置与代码：外部纯配置按目标选择 Runtime：普通 Skill 使用 second-brain-skills 的 `write-skill`，Workflow 包使用 `manage-fanloop-workflow`；代码和混合实现走 second-brain-skills `fanloop-workflow/code-change` Workflow（`fanloop-maintainer` 已退役，见 ADR-0135）。配置改变业务流程，不等于修改执行引擎。

1. 纯配置变更包括 `fanloop-workflow/<workflow-id>/` 的五份 Workflow YAML、包内非可执行 Skill 指令与参考材料、`<category>/<skill-id>/` 下的仓库级 Skill，以及相关说明文档。按现有 Schema 调整 Step、Condition、Route、Prompt 或 Skill 绑定，仍属于配置变更。
2. 配置按语义验证：错字/排版检查 diff 和引用并有据免测；普通模板、说明和 Prompt 由当前 Agent 实际自检，加载受影响完整包并检查代表性结果；Step、Condition、Route、绑定变化才跑受影响 Workflow 矩阵。路由、绑定、权限、安全、审批或外部写入规则变化仍需全新独立 Agent 实际试用并回读原始产物。解析器错误回归留在代码测试，不为文案重跑全产品或手造错误集。证据标明 self_check、independent 或 not_applicable，不要求新 binary 或代码 manifest。
3. 纯配置变更新开对应 Runtime Run：普通 Skill 使用 `write-skill`，Workflow 包使用 `manage-fanloop-workflow`；按最新 Status 的步骤完成修改、验证、独立审核、分支 PR 与实际配置源回读。旧维护仅继续旧 Run。普通 Skill 先安装已验证 head 的只读完整快照并确认，再交付 Skills PR；Workflow publish 完成父 PR、gitlink 与必需配置同步后才前进。用户指定停止位置时服从该边界。不要求 Go 全量测试、代码 Contract Golden 更新、构建、隔离 Release、独立代码黑盒验收、CLI 安装或代码验证 manifest。仅为配合固定生产配置的测试而修改代码或快照，会扩大原需求；应单独处理测试与配置的耦合。
4. Go/Node/Python 实现、可执行 Skill 脚本、IDL、Schema、加载器、通用 Runtime、构建或测试基础设施的修改属于代码变更，执行下方代码验证门禁。混合改动分别列出两类影响，代码部分仍完成代码验证。
5. 配置交付固定仓库绝对路径、Workflow ID、Git 基线/提交与完整资源摘要；验证并回读实际使用的本地源，不为配置更新重建 CLI。YAML 变更不设置独立人工审批；已有业务 Human Step 按用户保留的具体人工门禁执行；IDL 按下方主 Agent 契约审核执行。
6. 配置执行与文档要求以对应 Runtime 的最新绑定为准，保留原始工具结果及候选身份。普通 Skill 的自检、冻结安装和交付按 write-skill，Workflow 的分析、构建、审核和发布按 manage；不在辅助 Skill 中另建或绕过这些步骤。风险配置的独立试用仍按第 2 条执行，业务 Workflow 自身的 Panorama 和审批不变。只修改已确认范围，目标、权限、副作用或类别扩大时重新对齐。
7. CI 用完整 Git diff 的轻量 changes 任务分流，纯配置可跳过引擎任务；仅 `skills/`、`.agents/skills/<skill>/` 下普通非可执行 Skill Markdown、包根五 YAML、普通 docs Markdown、根 AGENTS.md/README.md 可豁免。脚本、验证资产、可执行位、符号链接、未知或混合改动走完整代码 CI；重命名检查前后路径和 mode，基线缺失不能判配置，分流失败阻止交付。本地材料禁入库检查始终保留。授权 PR 合并须核对同 HEAD 分流结果和适用 required checks，不能绕过保护；代码通道门禁不变。

完成标准：变更清单标明代码或配置；纯配置 diff 不包含为此次配置配套改写的引擎代码、代码测试或 Golden，并附配置校验/行为试用、本地配置提交及公开 CLI 回读。

## 主 Agent 监督自迭代

触发条件：使用 second-brain-skills `fanloop-workflow/code-change` Workflow 维护 Fanloop 代码；纯配置由当前 Agent直接推进，普通配置不派发实现或验收子 Agent；风险配置的独立行为试用按上方规则派发。

1. 用户只与主 Agent 对齐仓库目录、目标、范围和验收标准。主 Agent 派生执行子 Agent、持续监督，并负责需求批准、最终候选验收和 main 集成确认；询问用户的边界以第 2 条为准。
2. 在用户授权的目标、范围、副作用和停止位置内，主 Agent 默认自主决定需求批准、契约审核（包括 Thrift IDL）、最终候选验收和集成确认；先完整审阅具体材料，再记录明确决定，无需逐次向用户请示。执行子 Agent 仍须向主 Agent申请并等待决定。只有用户明确保留的具体人工门禁、超出授权范围的决策或仅人类能提供的信息才向用户确认；自主决定不扩大外部副作用或交付授权，也不替代自动 YAML 校验、独立 Code Review、公开 CLI 黑盒验收、CI、合并或安装证据。
3. 执行子 Agent 创建并持有 Requirement Root，持续读取最新 `flow status`、执行当前 Prompt/Skills、选择 Route、提交 Progress/Result，并直接修改受管代码直到 PR 合码和本地更新完成。主 Agent 不创建或驱动 Requirement，也不直接修改受管代码。
4. 执行子 Agent只向主 Agent回报，不直接联系用户。到达需求批准、最终候选验收或 main 集成确认时，必须向主 Agent申请明确决定并等待回复，不得自行批准。
5. 执行子 Agent每次读取最新 Status 和当前 Step 的全部必需 Skills，逐项核验真实产物后上报 Progress 或 Result。主 Agent发现偏差时通过 follow-up 要求其继续修复、补证据或重跑验证。
6. 独立 Code Review 和公开 CLI 黑盒验收继续使用不继承实现上下文的全新 Sub-agent，由执行子 Agent派发、聚合并推进；不得把实现上下文中的自证当作独立验收。

完成标准：主 Agent完成对齐、派发、监督和三类决定；执行子 Agent独立驱动 Workflow、完成受管实现、合并唯一 PR 并更新本地 Fanloop，所有决定都能回溯到主 Agent明确回复。

## 自迭代短反馈闭环

触发条件：任何 Fanloop 代码自迭代。

1. Define 在编码前记录 `low|medium|high` 风险、Acceptance Set、Feature Impact Set、依赖/权限、允许副作用和停止位置；不确定时按 `high`。契约未获批准不得修改受管代码。
2. Build 按可独立验证的小批纵向切片推进。每批只关闭一个风险或验收缺口并立即跑最小聚焦检查；失败归类为 `product`、`test/harness`、`environment` 或 `contract drift`，没有新证据时不得重复同一尝试。
3. 最后一次完整 `./tests/run-unit` 与公开 CLI 验证通过、提交并确认工作树 clean 后执行 **Pre-Certify**：复核完整 diff、ADR/契约、生成物、Feature Map、manifest、外部依赖和唯一 candidate identity；涉及生产配置时 identity 必须包含 `configuration_manifest_sha256`。任一项失败留在 Build。
4. 只有 Pre-Certify 通过，才可启动外层 Agent。独立 Code Review 与黑盒验证可并行认证，但必须读取同一冻结 candidate identity，使用互不共享的可写目录，并等待两者全部结束；黑盒使用的验证 Skill/Feature Map 必须从该提交复制到隔离只读目录并核对摘要。任一候选文件变化使两者全部失效。
5. 认证通过后才进行一次性交付：生产配置只交付一个已验证的 Git 候选，代码只维护一个 PR；合并使用显式 squash subject/body。
6. `delivery-record.md` 记录 lead time、Build/Certify 次数、返工来源和对应改进。只把重复出现、可执行且能被验证的原因沉淀为新规则，避免为单次事件增加流程。

完成标准：一个冻结候选对应一组本地证据、两份独立认证回执和一次交付；任何返工都回到最早责任阶段，而不是在 Review、CI 或发布时继续补丁。

## Workflow 配置中心

1. 生产真值是 `zeefan1555/second-brain-skills` 的 `main`，完整包位于 `fanloop-workflow/<workflow-id>/`。Fornax 只在 `FANLOOP_USE_FORNAX` 为 `1`、`true` 或 `yes` 且 `fornax.json` 是普通文件时作为显式选项。设置了 `FANLOOP_CONFIG_SOURCE` 时读取作者仓库。本代码仓库只保留通用 fixture 和作者工具。
2. 新的 `flow init` 默认用 HTTPS 拉取 `https://github.com/zeefan1555/second-brain-skills.git` 的 `main`，即使存在 `fornax.json`。远端探测每次最多 15 秒；每次 fetch 与 checkout 使用独立两分钟预算且不提示输入凭据。失败时同一次操作回退一次 `git@github.com:zeefan1555/second-brain-skills.git`，父 context 已取消时不重试。每次 checkout 尝试使用独立私有暂存仓库，成功后才发布固定 checkout；仅清理本次私有目录，不删除用户锁。失败按传输报告安全分类并保留 context 原因，不输出原始 Git stderr。`configuration.json` 记录实际成功的地址。已有快照按 commit 复用，不因 SSH 地址重新克隆。成功才原子激活 `$FANLOOP_DATA_HOME/config/snapshots/<commit-sha>/`。同一 commit 复用已安装快照。`FANLOOP_NO_AUTO_CONFIG=1` 跳过拉取。Git 或网络失败不替换 `config/current`；已有快照时告警并继续使用它，没有快照时返回 `CONFIGURATION_FETCH_FAILED` 且不创建 Requirement。`FANLOOP_USE_FORNAX=1` 且存在 `fornax.json` 时才改用已安装的 Fornax 快照。`fanloop update` 与 `fanloop update config` 使用同一条 Git 默认路径和同一个 Fornax 显式选项。`config/current` 指向不可变快照；校验或激活失败不修改 current、不创建 Requirement。dry-run 只校验临时内容。新 Requirement 在首次提交前把快照目录写到 `.fanloop/configuration-snapshot`，之后固定读取该目录。
3. 已有 Requirement 离线读取它固定的快照；没有固定文件的旧 Requirement 仍跟随 `current`。显式绝对 `FANLOOP_CONFIG_SOURCE` 指向配置仓库根，跳过远端刷新，用于作者和测试。Dashboard 默认只读快照，仅显式作者源可编辑；共享源锁、摘要检查和完整包校验不变。
4. 本地安装器离线；显式 `--config-source` 导入干净 Git 提交，否则保留 current。首次安装无配置时由后续 init 拉取。configuration.json 记录提交、Workflow IDs 和路径/内容/executable bit 的完整摘要。
5. 配置在 second-brain-skills 仓库交付，使用现有兼容 CLI 和本仓库 `scripts/validate-configuration.py` 校验。Agent 每次读取 Status 的精确 Skill 路径。配置不进入 State 或 CLI Release；Thrift 保留主 Agent 契约审核。
6. `fanloop update` 在切换 CLI 之后，与 `fanloop update config` 一样默认同步 `second-brain-skills` 的 `main`。它执行 `git ls-remote`，commit 未变则跳过 checkout，否则校验后发布 `config/snapshots/<sha>` 并在配置锁内激活。成功打印完整 40 位 commit 和 workflow id。Git 失败退出 1，`config/current` 不变。缺少 `fornax.json` 不是错误。`FANLOOP_CONFIG_SOURCE` 是作者源，拒绝同步。只有 `FANLOOP_USE_FORNAX` 为 `1`、`true` 或 `yes` 且 `fornax.json` 是普通文件时，才通过用户已登录的 `fornax-cli` 拉取 Online/prod 发布并打印 `id@version`。程序不内置 AK/SK。Fornax 快照目录名仍是排序后 `key@version` 行的 sha256。失败保留原来的 `current`。镜像 ZIP 根只含包装 `SKILL.md` 与唯一 `<workflow-id>/`；内部含五 YAML、可选 `trace.yaml` 和完整 `skills/**`。`loop-workflow-<id>` 与 `fanloop-workflow-<id>` 都可以装入。
7. `${FANLOOP_DATA_HOME:-$HOME/.fanloop}/fornax.json` 只存 source 列表。每一项是 `workspace_id`、`key_prefix` 和 `access`（`read_write` 或 `read_only`）。同一个 workflow id 同时出现在 `read_write` 和 `read_only` 源时，`read_write` 源胜出。同一源内重复，或两个源打成平局，整组拒绝。该文件不存 AK/SK、JWT、下载 token、Skill ID 或版本。Runtime 和安装器不创建或修改它。
8. 日常编辑是向 `second-brain-skills` 的 `main` 提 PR，合并后运行 `fanloop update config`。`.agents/skills/update-fanloop-workflow` 已退役，不再作为 Fornax 发布入口。`FANLOOP_USE_FORNAX=1` 且存在 `fornax.json` 时，`update config` 仍可同步 Fornax。把已发布快照写回 Git checkout 用 `scripts/mirror-fornax-to-git <snapshot-dir> <checkout-dir>`。从 Git 发布到 Fornax 的 `tools/workflow-package build` 只作为恢复手段。Fornax `skill install` 不保留 Unix 文件模式；以 `#!` 开头的脚本安装为 `0555`。写结果不明先回读，不盲重试。未获具体发布授权时只准备候选，不执行 Fornax 写入。

## YAML 架构真值与自动校验

触发条件：变更 Workflow YAML，或修改其模型、加载、校验、Schema 与执行语义。

1. 五份 Workflow YAML 是流程架构与推进语义的唯一真值；Stage、Job、Step、原子 Condition、正常/回流 Route 和 Prompt/Skills 必须能从 YAML 读出。Go 代码只实现通用加载、校验与执行，不承载某个 Workflow 的专用流程。
2. YAML 变更不设置独立人工审批。修改必须在隔离完整 Bundle 中完成，并通过严格解码、生成类型校验、跨文件引用/图校验、必需 Skill 校验和适用的行为试用后，才允许发布。
3. 每次生产 `workflow.yaml` 变更都自动比较当前发布版本与候选版本的 Step `id`、`name`、顺序和 `executor`，记录新增、删除、改名、重排或 executor 变化；差异属于普通配置契约事实，不再形成单独人工门禁。
4. 纯配置变更只更新 YAML、绑定 Skill 与必要说明并校验 Bundle；确实改变模型/Schema 或通用执行语义时，才同步加载校验、运行时和对应代码 Contract。涉及 `idl/yaml.thrift` 时同时遵循下方 Thrift IDL 门禁。公开 CLI 契约按 ADR-0040/0050 以 Thrift 与叶子 Help 为真值，生成的 Go `CommandSpec` 只读。

完成标准：五份生产 YAML 能完整表达 Workflow；加载时稳定拒绝语法、结构、引用、Condition、Route、Step 与 Skill 绑定错误；验证记录包含公开 CLI 错误码、退出码和无业务副作用证据。

## Workflow 全景图 Condition

触发条件：新增或修改生产 Workflow。

1. Panorama 是普通、可选的 Workflow Condition。只有需要展示全景图的 Workflow 才在 `condition.yaml` 声明对应 Condition，输出 `panorama_snapshot_path:path`，并通过 `prompt.yaml` 绑定同一 Workflow live 根中的展示 Skill；Runtime 不注入全景图节点或回执门禁。
2. Panorama 出现在哪些 Step、哪些正常/终止/回流 Route、何时展示以及是否需要完成态卡片，全部由该 Workflow 的 Condition、Route、Prompt 与绑定 Skill 明确声明。通用校验只验证这些声明自身闭合，不推断或补齐未声明的 Panorama 规则。
3. `awaiting_confirmation` 只暴露 `common_conditions` 与 `common_skills`。Workflow 若要在开工确认前展示 Panorama，须把 render/deliver 指令写进 step-start Prompt 或 common Skill；不能假设普通 Step Condition 会在确认前出现。
4. Workflow 的 Panorama Skill 要求真实投递时，`local_agent` 以及任何没有用户可见消息通道的 headless worker，不得把写入 `artifacts/panorama-*` 或任何仅保存文件当作成功。非 dry-run `fanloop card render --view panorama --format markdown` 返回的 `data.content` 必须按该 Skill 指定的时机原样投递，再上报本次 `data.snapshot_path`。
5. 能向用户聊天发消息的父 Agent 把 Panorama 任务交给 headless 子 Agent 时，子 Agent 不得提交投递 Condition。子 Agent render 后停住并交还 `data.content` 与 `data.snapshot_path`；父 Agent 原样发送，保留同一路径与合法的 `panorama_delivery_receipt` Evidence，让 Step 业务完成后再把 `panorama_presented` 与该 Route 的业务 Conditions 合并成一次 `flow report result`。
6. Runtime 在 terminal 后不创建隐藏节点。需要 `Done` / `100%` Panorama 的 Workflow 必须在 terminal 前已读取的绑定 Skill 中明确声明 post-terminal 动作；不需要的 Workflow 不受此规则约束。

| 规则 | 什么强制执行 |
| --- | --- |
| 无用户可见通道时 blocked；不得把写入 `artifacts/panorama-*` 当成已展示 | Skill 必须包含 `report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.`。`scripts/validate-configuration.py --check-panorama-skills` 拒绝缺少该句或把写文件当成投递的 Skill，并指出路径和应补的句子。`--reject-file-save-evidence` 拒绝 `content saved artifacts/panorama-` 这类证据。 |
| 父 Agent 能发消息时，headless 子 Agent 不得上报 `panorama_presented`；子 Agent 交还 `data.content` 与 `snapshot_path`，父 Agent 原样发送后与业务结果合并上报 | 提到 `local_agent` 的全景图 Skill 必须包含 `The child must not submit panorama_presented.`、`After non-dry-run fanloop card render --view panorama --format markdown, the child stops and returns data.content and data.snapshot_path to the parent; report blocked if render fails.`、`The parent sends the exact data.content as the first user-visible message for that Step, retains that same snapshot_path and a panorama_delivery_receipt Evidence item, lets the Step business work finish, then submits panorama_presented together with the business Conditions in one flow report result.`、合法 Evidence 示例 `{"source":"system","ref":"panorama_delivery_receipt","content":"{\"delivered\":true,\"snapshot_path\":\"<data.snapshot_path>\",\"host\":\"<host>\"}"}`，以及 `Never run a panorama-gated Workflow fully headless in a child that cannot post user-visible messages.`。`--check-panorama-skills` 拒绝缺少这些内容的 Skill。CLI 不证明用户看见了消息。 |

完成标准：选择全景图能力的 Workflow 用自身 YAML 和 Skill 完整表达所需时机与 Route；未选择的 Workflow 不受额外门禁。全景图 Skill 通过适用的投递完整性检查。通用 Runtime、入口与校验器不把某个 Panorama ID、Step 时机或完成态卡片变成隐藏规则。

## Thrift IDL 契约主 Agent 审核

触发条件：计划修改 `idl/` 下任意 `.thrift` 文件。

1. `idl/` 下的 Thrift 是人与 AI 共同维护的契约真值，默认只读。实现、测试和文档应先服从现有 IDL；不得为迁就实现、生成代码或测试而反向改契约，也不得直接编辑 `internal/idl/` 下的生成物来绕过 IDL。
2. 确需改变契约时，编辑前向主 Agent 提供具体文件与符号、变更前后 Thrift 片段、field ID/可选性/类型/枚举/Service/Annotation/Error 的变化、对生成代码、Runtime、持久化、Workflow YAML、调用方及 Contract 的影响，以及不修改 IDL 的替代方案为何不可行。主 Agent 按上方自主决策边界完整审阅并记录明确审核决定；执行子 Agent 必须等待该决定。用户保留的人工契约门禁仍由用户审核。
3. 契约审核只覆盖已审阅的具体变更。范围扩大或新增契约变化时重新提交主 Agent 审阅；开放问题、推荐项和笼统的实现授权不代替具体契约决定。超出用户授权范围时由主 Agent先向用户确认。
4. 审核通过后，以 Thrift 为第一修改点，使用仓库现有生成链同步生成物、实现、测试和 Contract；所有相关内容进入同一分支，不手写生成文件。

完成标准：每个 `.thrift` diff 都能定位到修改前的明确主 Agent 契约审核记录（用户保留门禁时为用户审核记录）；实际 diff 与审核过的前后示例一致，生成物新鲜，所有受影响消费者与 Contract 已同步验证。

## 代码验证门禁

触发条件：修改 Go、Node、Python 实现、可执行 Skill 脚本、IDL、Schema、加载器、Runtime、生成链、构建、发布或测试基础设施。纯 Workflow YAML、Skill 指令与参考材料变更走上方配置路径。

1. 实现期间必须运行受影响的聚焦测试、必要格式检查和一次 `./tests/run-unit`；命令、结果和当前 HEAD 写入验证记录。
2. 代码变更影响用户可见 CLI、Workflow 执行机制、安装、状态、投影或 Skill 脚本行为时，从同一候选安装隔离 Release。安装、`fanloop verify doctor`、Feature Map 和 `verify.evidence.outcome` 只按 `.cursor/skills/verify-fanloop/SKILL.md` 执行。生产 Workflow 目录写在 `docs/ownership.md`。Pre-Certify 通过后冻结 `candidate_head`，同时启动独立 Code Review 与一个无实现上下文的全新 Sub-agent 黑盒验证；后者只使用公开 CLI 验证需求中批准的场景。不得拿已发布版本、旧安装、机器人或源码阅读当证据。
3. `.github/scripts/run-requirement-validation` 只属于 CI，实现完整 Requirement lifecycle 与通用运行时 fixture Route Matrix；它不是本地开发入口，也不得被 Maintainer Skill 要求手工执行。
4. 发布唯一 PR 后，CI 硬门禁必须回读同一 `candidate_head` 的 Ruleset 与 required checks。源码、测试或验证资产更新后，旧本地验证、Review、Sub-agent 验收与 CI 全部失效，必须形成新 HEAD 并从本地验证重跑。
5. 任一必需验证失败都阻止合并；不得用旧脚本、旧二进制、代码审阅或远端 CI 倒推本地验证结论。

完成标准：聚焦命令与 `./tests/run-unit` 通过，本地验证和 AI Code Review 覆盖冻结的最终候选 HEAD；用户表面变化还要求隔离候选的真实公开 CLI 证据与 Sub-agent 黑盒验收。远端 Ruleset 与 required checks 在同一 `candidate_head` 上通过后自动 squash 合并唯一 PR；随后只更新本 Requirement 的源码 worktree，并从精确 merge commit 本地构建、安装和回读版本、Doctor 与 smoke。

## 安装与分发

1. 用户安装来自 GitHub Release，不使用 npm registry，也不发布到 GitHub Packages 或私有 package registry。推送形如 `vX.Y.Z` 的 tag 后，Actions 只在 ubuntu 上交叉编译 darwin/linux 的 amd64 与 arm64，上传每个平台一个 tar.gz、`checksums.txt`、分发清单 `release.json`，以及 `npm pack` 生成的薄包装 tgz。
2. 安装命令是 `npm i -g --foreground-scripts --allow-scripts=fanloop https://github.com/zeefan1555/fanloop/releases/download/vX.Y.Z/fanloop-X.Y.Z.tgz`。`--allow-scripts=fanloop` 允许较新的 npm 运行 postinstall；`--foreground-scripts` 让安装程序打印的 `export PATH=...` 出现在终端里。安装程序校验 sha256 和完整的内部 release 清单，确认文件摘要后解压到 `~/.fanloop/releases/<version>`，创建指向 `current` 的公共 Skill 链接，最后原子切换 `current`。
3. `fanloop update` 下载最新 GitHub Release 并切换 CLI 的 `current`，然后默认从 `second-brain-skills` `main` 同步 Workflow 配置。已安装版本比 `latest` 新时 CLI 保持不变。同一版本会按已校验归档修复被改过的发布内容。`fanloop update config` 只做这次 Git 同步：commit 未变则跳过 checkout，并发运行共用配置锁，并打印完整 commit 与 workflow id。Git 失败保留 `current`。只有 `FANLOOP_USE_FORNAX` 打开且 `fornax.json` 存在时才同步 Fornax，并在已发布集合和已安装树都没变时跳过下载、打印 `id@version`。新的非 dry-run `flow init` 在 `current` 是版本化 Release 符号链接时用同一套校验检查最新 Release；已是最新或更新则跳过下载。失败打印 `automatic update failed; checking for an installed fallback` 并继续。更新成功后重新执行新二进制。`FANLOOP_NO_AUTO_UPDATE=1` 关闭。开发目录形式的 `current` 不自动更新。
4. 开发仍用 `./scripts/build-local.sh` 与 `./scripts/install-local.sh`。`npm run build` 与 `npm run install:local` 只是这两个脚本的快捷方式，不向 registry 发布。`go.mod` 要求 Go 1.23.0 或更高版本。
5. 合并后的代码验证仍可从精确 merge commit 执行 `./scripts/install-local.sh`，并回读 `fanloop version`、`fanloop doctor` 与必要 smoke。生产配置在 second-brain-skills main 维护；`fanloop update config` 或下次 flow init 从 GitHub 拉取。Fornax 仅在 `FANLOOP_USE_FORNAX` 打开时使用。显式作者源编辑在下一次调用生效。分发安装的 `current` 是符号链接时，若 `fanloop` 不在 PATH 上，`fanloop doctor` 给出带 `export PATH=...` 的 warning。

## ADR 一致性

触发条件：每一次仓库变更。

1. 修改前，按受影响的领域、公开字段、存储路径和组件检索 `docs/adr/`，完整读取相关 ADR 及其 supersede/amend 链。较新的 ADR 可能只取代旧 ADR 的一部分，不能只看 `status` 字段。
2. 在 MR 的 `ADR impact` 中记录本次保留、修订、取代的 ADR；确实无影响时记录 `none` 及理由。
3. 计划行为与仍有效的 ADR 冲突时，先新增或修订 ADR，明确冲突范围和取代关系，再同步技术设计、代码、测试与 Contract。已有决策不得被实现静默改写。
4. 提交前，用目标分支的最新 merge base 复核完整 diff，并重新对照相关 ADR；测试通过不能替代这一步。
5. ADR 文件名的四位数字前缀只用于排序，不是唯一标识。相同数字、不同完整文件名的 ADR 必须原样共存；不得仅因数字重复而重命名、顺延编号、阻塞、回流或让候选重新验证。只处理真实的同路径 Git 内容冲突和失效引用。

完成标准：变更与所有相关有效 ADR 一致；有意改变的决策具备已提交的 ADR 记录、明确的取代关系和对应回归证据。

## 外层循环

代码变更在 `.cursor/skills/verify-fanloop/SKILL.md` 留下 `outcome` 为 `passed` 的 schema-1 manifest 之前，不要为该变更启动云端 Agent 或并行 Agent。步骤见 `docs/outer-loop.md`。纯配置变更不要求代码验证 manifest。

## 与 second-brain-agent 的关系

本仓库是 second-brain-agent 的 `toolchain/fanloop` 子仓库。上文门禁照旧有效；在此之上，所有 bot 的通用交付步骤（读对应 skills 目录 → 分支 → 验证 → 自动 PR 附证据与中文描述 → CI 通过后 squash 合并 → 到主仓库自动 PR bump 指针）见
[「Bot 工作流程（所有仓库通用）」](https://github.com/zeefan1555/second-brain-agent/blob/main/AGENTS.md#bot-工作流程所有仓库通用)。
本仓库的验证 skill 是 [`.cursor/skills/verify-fanloop`](.cursor/skills/verify-fanloop/SKILL.md)（证据为 `fanloop verify smoke` 的 `manifest.json`）。
