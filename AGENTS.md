# Fanloop 变更门禁

## 本地材料不入库

1. `.scratch/`、`docs/research/` 和 `docs/specs/` 仅可作为本地临时材料，永远不得纳入版本控制，也不得使用 `git add -f` 绕过忽略规则。
2. 不得因历史引用或旧流程恢复这些目录中的已删除内容；确需临时查阅时从 Git 历史读取，且不重新提交。
3. 提交前检查这三个路径在暂存区中只能出现删除记录；一旦出现新增或修改，立即停止提交并移出暂存区。

## 已确认决策沉淀

触发条件：需求经过 `/grill-with-docs`、飞书评审或对话澄清。

1. 编码前，把已确认的决策整理为决策、约束和验收条件。需要入库的行为契约直接落在对应的现有真值中，例如生产 YAML、Thrift IDL 或契约测试；长期且难以逆转的架构选择进入 `docs/adr/`。研究与任务拆分留在仓库外或上述本地临时目录中。
2. 只沉淀用户明确确认的结论。开放问题继续保留为开放问题，不把推荐项自动写成决策。
3. 让需要入库的契约真值与实现处于同一分支，并在实现提交之前或同一提交中纳入 Git；外部文档、聊天记录和未跟踪文件不能作为唯一事实源。

完成标准：逐项对照澄清来源与目标分支相对 merge base 的 diff；每个已确认的行为、边界和验收条件都能定位到已跟踪文件。

## YAML 架构真值与人工审核

触发条件：变更 Workflow YAML，或修改其模型、加载、校验、Schema 与执行语义。

1. 五份 Workflow YAML 是流程架构与推进语义的唯一真值；Stage、Job、Step、原子 Condition、正常/回流 Route 和 Prompt/Skills 必须能从 YAML 读出。Go 代码只实现通用加载、校验与执行，不承载某个 Workflow 的专用流程。
2. 涉及 YAML 文件、字段、类型、引用关系或推进语义的代码修改，编码前先提供真实生产 YAML 的变更前后示例和影响说明，经人明确审核通过后再实施或合并。
3. 生产 `workflow.yaml` 的 Step 集合与拓扑是人工维护的契约。每次生产 YAML 变更都必须逐项比较当前默认版本与目标版本的 Step `id`、`name`、顺序和 `executor`，明确记录是否存在新增、删除、改名、重排或 executor 变化。未经针对具体 Step diff 的人工批准，这四项必须完全不变；任何变化都必须作为独立契约变更高亮报告，不能夹在 Condition、Route、Prompt 或其他 YAML 审核中默认通过。
4. 审核通过后，同步更新 YAML、模型/Schema、加载校验、运行时和对应 Contract；涉及 `idl/yaml.thrift` 时同时遵循下方 Thrift IDL 门禁。公开 CLI 契约按 ADR-0040/0050 以 Thrift 与叶子 Help 为真值，生成的 Go `CommandSpec` 只读。

完成标准：人只阅读五份生产 YAML 就能理解 Workflow 做什么、哪些 Condition 组合会前进以及哪些组合可回到哪里；Step `id`、`name`、顺序和 `executor` 的基线对比结论明确；每个 YAML 结构变化都能定位到明确的人审记录。

## Thrift IDL 契约人工审核

触发条件：计划修改 `idl/` 下任意 `.thrift` 文件。

1. `idl/` 下的 Thrift 是人与 AI 共同维护的契约真值，默认只读。实现、测试和文档应先服从现有 IDL；不得为迁就实现、生成代码或测试而反向改契约，也不得直接编辑 `internal/idl/` 下的生成物来绕过 IDL。
2. 确需改变契约时，编辑前先向人报告并等待明确审核。报告至少包括具体文件与符号、变更前后 Thrift 片段、field ID/可选性/类型/枚举/Service/Annotation/Error 的变化、对生成代码、Runtime、持久化、Workflow YAML、调用方及 Contract 的影响，以及不修改 IDL 的替代方案为何不可行。
3. 人工确认只覆盖已报告的具体变更。范围扩大或新增契约变化时重新报告；开放问题、推荐项和笼统的实现授权不视为 IDL 修改许可。
4. 审核通过后，以 Thrift 为第一修改点，使用仓库现有生成链同步生成物、实现、测试和 Contract；所有相关内容进入同一分支，不手写生成文件。

完成标准：每个 `.thrift` diff 都能定位到修改前的人审记录；实际 diff 与审核过的前后示例一致，生成物新鲜，所有受影响消费者与 Contract 已同步验证。

## 按风险验证门禁

触发条件：修改 Go、Node、Python、Workflow、IDL、生成链、构建或发布脚本等会改变产品或测试行为的代码。

1. 实现期间必须运行测试计划列出的聚焦测试和必要格式检查；命令、结果和当前 HEAD 写入验证记录。
2. Fanloop maintainer Workflow 的本地验证与代码审查只使用当前工作树；审查通过后冻结 `candidate_head`，再发布唯一 PR。后续 CI 硬门禁必须回读该精确 SHA 的 Ruleset 与 required checks，不能替代或倒推本地验证。需要在真实 Agent 会话中复现 Skill、Prompt 与 Workflow 时，用 `npm run install:local` 从同一候选安装配套 Release，不拿已发布版本或旧安装当证据。源码、测试或验证资产更新后，旧本地验证、Review、Eval、CI 与机器人验收全部失效，必须形成新 HEAD 并从本地验证重跑。
3. 测试计划必须选择 `targeted` 或 `e2e`。文档、自迭代 Skill、未改变 Step/Route/Condition/executor 的 Prompt/SkillBinding，以及具备完整聚焦 seam 的局部叶子行为可以选择 `targeted`；该档只执行计划命令，不在本地重复 `run-unit` 或 `run-e2e`。
4. 修改 Step/Route/Condition 推进语义、Thrift IDL、durable state/storage/output/trace/card、发布/安装/更新/打包、测试入口，或者缺少可靠聚焦 seam、影响面不确定时必须选择 `e2e`。该档在计划命令之外，从同一工作树运行 `./tests/run-unit` 与 `./tests/run-e2e` 并记录报告路径。
5. 任一必需验证失败都阻止合并；不得用旧脚本、旧二进制或代码审阅代替。无法确定风险时选择 `e2e`，不得伪造 `targeted` 结论。

完成标准：聚焦命令通过，本地验证与 AI Code Review 覆盖冻结的最终候选 HEAD；`e2e` profile 还要求本地 `run-unit` 与 `run-e2e` 零退出，报告对应当前工作树和二进制，且测试执行未改变源码状态。发布 PR 后，远端 Ruleset 与 required checks 还必须在同一 `candidate_head` 上通过，才可进入机器人验收和自动合码。

## ADR 一致性

触发条件：每一次仓库变更。

1. 修改前，按受影响的领域、公开字段、存储路径和组件检索 `docs/adr/`，完整读取相关 ADR 及其 supersede/amend 链。较新的 ADR 可能只取代旧 ADR 的一部分，不能只看 `status` 字段。
2. 在 MR 的 `ADR impact` 中记录本次保留、修订、取代的 ADR；确实无影响时记录 `none` 及理由。
3. 计划行为与仍有效的 ADR 冲突时，先新增或修订 ADR，明确冲突范围和取代关系，再同步技术设计、代码、测试与 Contract。已有决策不得被实现静默改写。
4. 提交前，用目标分支的最新 merge base 复核完整 diff，并重新对照相关 ADR；测试通过不能替代这一步。

完成标准：变更与所有相关有效 ADR 一致；有意改变的决策具备已提交的 ADR 记录、明确的取代关系和对应回归证据。
