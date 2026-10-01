# Fanloop 变更门禁

## 本地材料不入库

1. `.scratch/`、`docs/research/` 和 `docs/specs/` 仅可作为本地临时材料，永远不得纳入版本控制，也不得使用 `git add -f` 绕过忽略规则。
2. 不得因历史引用或旧流程恢复这些目录中的已删除内容；确需临时查阅时从 Git 历史读取，且不重新提交。
3. 提交前检查这三个路径在暂存区中只能出现删除记录；一旦出现新增或修改，立即停止提交并移出暂存区。

## 已确认决策沉淀

触发条件：需求经过 `/grill-with-docs`、飞书评审或对话澄清。

1. 编码前，把已确认的决策整理为决策、约束和验收条件。代码行为契约落在现有 Thrift IDL 或契约测试；生产配置契约随仓库 skills/<workflow-id>/ 完整包提交，并记录 Git 基线与文件摘要；长期且难以逆转的架构选择进入 `docs/adr/`。研究与任务拆分留在仓库外或上述本地临时目录中。
2. 只沉淀用户明确确认的结论。开放问题继续保留为开放问题，不把推荐项自动写成决策。
3. 让代码契约真值与实现处于同一分支，并在实现提交之前或同一提交中纳入 Git；外部文档、聊天记录和未跟踪文件不能作为唯一事实源。

完成标准：逐项对照澄清来源与目标分支相对 merge base 的 diff；每个已确认的行为、边界和验收条件都能定位到已跟踪文件。

## 代码与配置分离

Define 按改动内容确定类别并记录；后续节点依类别绑定独立的配置或代码 Skill。公共入口只负责路由，不混写两类操作。配置改变业务流程，不等于修改执行引擎。

1. 纯配置变更包括五份 Workflow YAML、绑定 Skill 的指令与参考材料，以及相关说明文档。按现有 Schema 调整 Step、Condition、Route、Prompt 或 Skill 绑定，仍属于配置变更。
2. 配置只做受影响 Bundle 的加载与 Schema 校验、引用和必需 Skill 检查、路由完整性检查；需要运行验证时使用已有兼容 CLI 和隔离配置，按需检查受影响路径。执行行为变化在 Certify 由全新、不继承修改上下文的 Sub Agent 实际试用受影响场景，回读原始产物后判定；纯错别字/排版可记录理由免测。配置内容与试用证据不要求新 binary 或代码验证 manifest。
3. 纯配置变更走 `fanloop-maintainer` 的配置通道，保留 Define、Build、Certify、Deliver 四步及本地仓库提交、公开 CLI 回读闭环，不启动代码交付通道，不要求 Go 全量测试、代码 Contract Golden 更新、构建、隔离 Release、独立代码黑盒验收、CLI 安装或代码验证 manifest。仅为配合固定生产配置的测试而修改代码或快照，会扩大原需求；应单独处理测试与配置的耦合。
4. Go/Node/Python 实现、可执行 Skill 脚本、IDL、Schema、加载器、通用 Runtime、构建或测试基础设施的修改属于代码变更，执行下方代码验证门禁。混合改动分别列出两类影响，代码部分仍完成代码验证。
5. 配置交付固定仓库绝对路径、Workflow ID、Git 基线/提交与完整资源摘要；验证并回读实际使用的本地源，不为配置更新重建 CLI。YAML 变更不设置独立人工审批；已有业务 Human Step 与 IDL 门禁按各自适用范围保留。

完成标准：变更清单标明代码或配置；纯配置 diff 不包含为此次配置配套改写的引擎代码、代码测试或 Golden，并附配置校验/行为试用、本地配置提交及公开 CLI 回读。

## 主 Agent 监督自迭代

触发条件：使用 `fanloop-maintainer` 维护 Fanloop 代码；纯配置变更由当前 Agent直接推进，不派发实现子 Agent；Certify 的独立行为试用按上方规则派发。

1. 用户只与主 Agent 对齐仓库目录、目标、范围和验收标准。主 Agent 派生执行子 Agent、持续监督，并负责需求批准、最终候选验收和 main 集成确认；只有用户主动改变已对齐内容时才重新询问用户。
2. 当用户针对当前 Requirement 明确授权主 Agent 全程自行决定时，主 Agent 可在完整审阅对应材料后代行需求批准、最终候选验收和 main 集成确认，无需逐次向用户复询。该授权不转授给执行子 Agent，不适用于其他 Requirement，也不替代 IDL 人审、自动 YAML 校验、独立 Code Review、公开 CLI 黑盒验收、CI、合并或安装证据。
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
5. 认证通过后才进行一次性交付：生产配置只交付一个已验证的 Git 候选，代码只维护一个 PR；合并使用显式 squash subject/body，最终 trailer block 中 `Co-Authored-By: Aiden` 恰好一次。
6. `delivery-record.md` 记录 lead time、Build/Certify 次数、返工来源和对应改进。只把重复出现、可执行且能被验证的原因沉淀为新规则，避免为单次事件增加流程。

完成标准：一个冻结候选对应一组本地证据、两份独立认证回执和一次交付；任何返工都回到最早责任阶段，而不是在 Review、CI 或发布时继续补丁。

## 本地 Workflow Skill 包真值

触发条件：读取、修改、安装或执行 Workflow/Skill，或查看 Dashboard。

1. 仓库 skills/<workflow-id>/ 是 Workflow 配置唯一真值；包根包含 SKILL.md 与五份 YAML，子 Skill 和完整资源位于同包 skills/ 下。不存在 workflows/<id>/ 包装层。
2. 本地安装通过 $FANLOOP_DATA_HOME/config/current 绑定绝对源仓库，默认绑定安装脚本所在仓库；显式绝对 FANLOOP_CONFIG_SOURCE 优先。每次调用读取源包，源失效立即失败，不搜索 cwd、旧副本或其他仓库。没有远端配置同步命令。
3. Dashboard 默认只读绑定源；只有显式绝对 FANLOOP_CONFIG_SOURCE 时可编辑该仓库，包括当前绑定仓库。必须拒绝符号链接、陈旧摘要和不完整包，所有数据目录共享同一规范源目录锁。配置经自动校验、必要独立行为试用、本地提交与公开 CLI 回读交付。
4. scripts/validate-configuration.py 对固定本地源执行全部 Workflow Route Matrix 并生成完整文件摘要。代码 CI 使用 internal/workflow/testdata/{linear-runtime,runtime-controls} 独立 fixture，在临时源根 skills/ 下组装，不绑定个人凭据或生产正文。
5. Agent每次读取最新Status返回的精确Skill路径。配置内容不写入 Requirement State 或 CLI Release；五份 YAML 在加载时自动校验，Thrift继续保留人审门禁。

## YAML 架构真值与自动校验

触发条件：变更 Workflow YAML，或修改其模型、加载、校验、Schema 与执行语义。

1. 五份 Workflow YAML 是流程架构与推进语义的唯一真值；Stage、Job、Step、原子 Condition、正常/回流 Route 和 Prompt/Skills 必须能从 YAML 读出。Go 代码只实现通用加载、校验与执行，不承载某个 Workflow 的专用流程。
2. YAML 变更不设置独立人工审批。修改必须在隔离完整 Bundle 中完成，并通过严格解码、生成类型校验、跨文件引用/图校验、必需 Skill 校验和适用的行为试用后，才允许发布。
3. 每次生产 `workflow.yaml` 变更都自动比较当前发布版本与候选版本的 Step `id`、`name`、顺序和 `executor`，记录新增、删除、改名、重排或 executor 变化；差异属于普通配置契约事实，不再形成单独人工门禁。
4. 纯配置变更只更新 YAML、绑定 Skill 与必要说明并校验 Bundle；确实改变模型/Schema 或通用执行语义时，才同步加载校验、运行时和对应代码 Contract。涉及 `idl/yaml.thrift` 时同时遵循下方 Thrift IDL 门禁。公开 CLI 契约按 ADR-0040/0050 以 Thrift 与叶子 Help 为真值，生成的 Go `CommandSpec` 只读。

完成标准：五份生产 YAML 能完整表达 Workflow；加载时稳定拒绝语法、结构、引用、Condition、Route、Step 与 Skill 绑定错误；验证记录包含公开 CLI 错误码、退出码和无业务副作用证据。

## Workflow 全景图 Condition

触发条件：新增或修改生产 Workflow。

1. 每个 Workflow 必须在 `condition.yaml` 声明全景图 Condition，输出 `panorama_snapshot_path:path`，并通过 `prompt.yaml` 绑定同一 Workflow live 根中的必需展示 Skill。
2. 全景图 Condition 必须接入实际 Route；需要展示的 Step，其正常、终止及回流 Route 的每个 `when.any_of` AND 组都必须包含该 Condition，不能只声明而允许路由绕过。
3. 每次进入这些 Step（包括回流重入）先展示本次 renderer 的完整内容；宿主展示成功后才上报本次精确快照路径。失败或结果不明时报告 blocked；同一次 Step 内复用已展示回执，进入新 Step 后重新展示。

完成标准：全部生产 Workflow 均有已绑定并实际使用的全景图 Condition；配置检查覆盖所需正常、终止和回流 Route 的全部 AND 组。必要时用已有 CLI 检查回执缺失的拒绝行为，不为此触发代码交付门禁。

## Thrift IDL 契约人工审核

触发条件：计划修改 `idl/` 下任意 `.thrift` 文件。

1. `idl/` 下的 Thrift 是人与 AI 共同维护的契约真值，默认只读。实现、测试和文档应先服从现有 IDL；不得为迁就实现、生成代码或测试而反向改契约，也不得直接编辑 `internal/idl/` 下的生成物来绕过 IDL。
2. 确需改变契约时，编辑前先向人报告并等待明确审核。报告至少包括具体文件与符号、变更前后 Thrift 片段、field ID/可选性/类型/枚举/Service/Annotation/Error 的变化、对生成代码、Runtime、持久化、Workflow YAML、调用方及 Contract 的影响，以及不修改 IDL 的替代方案为何不可行。
3. 人工确认只覆盖已报告的具体变更。范围扩大或新增契约变化时重新报告；开放问题、推荐项和笼统的实现授权不视为 IDL 修改许可。
4. 审核通过后，以 Thrift 为第一修改点，使用仓库现有生成链同步生成物、实现、测试和 Contract；所有相关内容进入同一分支，不手写生成文件。

完成标准：每个 `.thrift` diff 都能定位到修改前的人审记录；实际 diff 与审核过的前后示例一致，生成物新鲜，所有受影响消费者与 Contract 已同步验证。

## 代码验证门禁

触发条件：修改 Go、Node、Python 实现、可执行 Skill 脚本、IDL、Schema、加载器、Runtime、生成链、构建、发布或测试基础设施。纯 Workflow YAML、Skill 指令与参考材料变更走上方配置路径。

1. 实现期间必须运行受影响的聚焦测试、必要格式检查和一次 `./tests/run-unit`；命令、结果和当前 HEAD 写入验证记录。
2. 代码变更影响用户可见 CLI、Workflow 执行机制、安装、状态、投影或 Skill 脚本行为时，从同一候选安装隔离 Release。安装、`fanloop verify doctor`、Feature Map 和 `verify.evidence.outcome` 只按 `.cursor/skills/verify-fanloop/SKILL.md` 执行。生产 Workflow 目录写在 `docs/ownership.md`。Pre-Certify 通过后冻结 `candidate_head`，同时启动独立 Code Review 与一个无实现上下文的全新 Sub-agent 黑盒验证；后者只使用公开 CLI 验证需求中批准的场景。不得拿已发布版本、旧安装、机器人或源码阅读当证据。
3. `.github/scripts/run-requirement-validation` 只属于 CI，实现完整 Requirement lifecycle 与通用运行时 fixture Route Matrix；它不是本地开发入口，也不得被 Maintainer Skill 要求手工执行。
4. 发布唯一 PR 后，CI 硬门禁必须回读同一 `candidate_head` 的 Ruleset 与 required checks。源码、测试或验证资产更新后，旧本地验证、Review、Sub-agent 验收与 CI 全部失效，必须形成新 HEAD 并从本地验证重跑。
5. 任一必需验证失败都阻止合并；不得用旧脚本、旧二进制、代码审阅或远端 CI 倒推本地验证结论。

完成标准：聚焦命令与 `./tests/run-unit` 通过，本地验证和 AI Code Review 覆盖冻结的最终候选 HEAD；用户表面变化还要求隔离候选的真实公开 CLI 证据与 Sub-agent 黑盒验收。远端 Ruleset 与 required checks 在同一 `candidate_head` 上通过后自动 squash 合并唯一 PR；随后只更新本 Requirement 的源码 worktree，并从精确 merge commit 本地构建、安装和回读版本、Doctor 与 smoke。

## 本地源码分发

1. GitHub 只托管源码和运行 CI；仓库不保留 npmjs.org、GitHub Packages、bnpm、Luban 或手工 `npm publish` 入口。
2. `npm run build` 与 `npm run install:local` 只是本地 Shell 入口的快捷方式，不生成或发布 npm 包。
3. 合并后从精确 merge commit 执行 `./scripts/install-local.sh`，并回读 `fanloop version`、`fanloop doctor` 与必要 smoke；安装离线绑定源仓库，不创建 Workflow 运行副本。
4. 生产配置直接编辑仓库完整包，下一次 CLI 调用生效；不需要网络、CLI 重建或重新安装。

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
