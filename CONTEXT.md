# Fanloop 领域词典

## 流程配置

**Requirement**
一次需要持续推进和审计的任务。事实保存在 Requirement 自己的 `.fanloop/`。

**Live Workflow**
second-brain-skills main 的 `fanloop-workflow/<workflow-id>/` 完整包。默认读取 config/current 指向的提交快照，flow init 刷新；已有流程离线读取。显式 FANLOOP_CONFIG_SOURCE 为作者仓库根并跳过刷新。五 YAML、子 Skills 与资源独立于 CLI Release。

**Configuration Snapshot**
config/snapshots/<commit-sha>/ 下的不可变配置集合。configuration.json 固定仓库、ref、commit、Workflow IDs 与文件路径、内容、可执行位摘要。完整校验后才原子切换 current。

**Requirement Workflow Resolution**
Requirement 只持久化 Workflow ID 与创建它的 CLI Release version。每次 CLI 调用都按该 ID 从同一个
live 根重新加载五份 YAML 和绑定 Skills；历史 Event 不按当前 Route predicate 重判，但当前 Step、Output
所有权、因果链、失效集合、replay tail 和所有新 Result 都必须与最新 live Workflow 兼容。旧 Storage
Schema 不迁移，也不由新 CLI 读取。

**Stage / Job / Step**
显式执行层级。`step_id` 在 Workflow 内唯一；同一 Requirement 最多有一个活动 Step。

**Condition**
可被多个 Step Route 引用的原子事实定义。包含 ConditionID、PromptRef、一个 OutputSpec 和可选 exclusive_group，不绑定 Step 或跳转目标。

**OutputSpec / RegisteredOutput**
Condition 声明 Output 的 key、type 和约束。支持 `string`、`boolean`、`integer`、`path`、`url`、`url_list`、`enum_value`、`object`。Agent 上报 type/value；CLI 写入 State 时补充 key 与 `producer_step_id`。

**FlowRoute / LoopRoute**
两者都按 StepID 映射 Route 数组，共享 PromptRef 与 `when.any_of`。外层数组是 OR，内层 ConditionID 数组是 AND。FlowRoute 声明 `next_step_id` 或 `terminal`；LoopRoute 声明 `back_step_id`。

**Prompt**
给 Agent 的执行或路由指导。可绑定多个 Skill；每个 Skill 有独立 Prompt 和显式 optional。

**Evidence**
由 source、content 和可选 ref 组成的审计内容。进入 Event、Trace 和 Card，但不参与 Route 匹配，也不证明业务事实或人工身份。

## 调用与结果

**Flow Progress Report**
`flow report progress` 更新当前 Step 的 `in_progress|fixing|blocked`、Summary 和 Evidence，不移动 Step。

**Flow Result Report**
`flow report result` 上报 StepID、ConditionResults、必填 one-of RouteSelection、Evidence 和 Summary。RouteSelection 是 `next_step_id`、`back_step_id` 或 `terminal: true`；调用方不提交 Effect、Output key、producer 或内部 matched 信息。

**Route 匹配**
Agent 从最新 `available_routes` 选择 Route。CLI 只在该方向与目标内校验 ConditionResults 唯一命中 `when.any_of`；零命中、多命中、互斥冲突、非法目标和错误方向都是原子拒绝。

**Flow Effect / Transition**
Progress 返回 `status_updated`。Result 返回 `advanced|looped|completed`，以及 direction、from_step_id 和可选 to_step_id。非 dry-run Result 还返回 EventID。

**FlowState**
由 State 和当前 live Workflow 派生。运行中返回扁平 CurrentContext、execution、Prompt/Skills、当前相关 Conditions、统一 `available_routes` 和有效 Outputs；完成后不含 current。

## 当前事实与投影

**State / Output Registry**
恢复执行的当前事实分开落盘：`.fanloop/flow/state.json` 保存 Requirement、CLI Release/Workflow ID 绑定、活动 Step、Summary/Evidence、integration 和最后 Event 指针；`.fanloop/output/state.json` 保存当前有效 RegisteredOutputs。当前为 State/Event Schema 14、Output Registry Schema 4；字段、枚举和版本以 `idl/storage.thrift` 为唯一可编辑真值。

**Event**
不可变审计记录。Event Schema 14 使用 Thrift `EventPayload` union；落盘的 `kind` 与唯一 payload 成员一一对应，例如 `flow_progressed`、`flow_result`。Result Event 保存原始 ConditionResults、Evidence、Summary、Effect、Transition 和 accepted/invalidated OutputChanges。dry-run 和 rejected request 不产生 Event。

**CLI Execution Log**
Requirement 范围的完整诊断 transcript。每个具有有效绝对 `--root` 的真实叶子调用在结束后 best-effort 向 `.fanloop/log/cli.jsonl` 追加一条 schema 2 Thrift JSON，包括有序 arguments、实际消费的 stdin、完整 stdout/stderr、只读、失败和 `--dry-run`；`--input @file` 只记录参数中的路径，不把文件内容算作 stdin。Help 与无 Root 的 bootstrap/版本调用不记录。日志不脱敏、不截断，可能包含 URL、Evidence、密钥或 token；目录/文件权限保持 `0700/0600`。它不参与恢复、路由或 Doctor 健康判断，写入失败也不改变命令结果；当 `internal/traceconfig/registry.yaml` 选中的策略要求 CLI 日志文档时，`trace sync` 会把同步开始前的完整字节快照投影到该文档。

**Output 失效**
Loop 根据 State 中 `producer_step_id` 失效由 back Step 及其下游生产的有效 Output，保留更早 Step 的事实。

**Panorama 发布**
Workflow 通过 YAML 的 `panorama_card_published` 或 `panorama_presented` Condition 声明 renderer 生成的紧凑 Panorama 已由当前宿主成功展示或发送，Output 原样保存本次 non-dry-run render 返回的 Requirement Root 相对 `panorama_snapshot_path`。该 Condition 必须与对应 Route 事实在同一组合中上报；CLI 只校验类型与 Route，不自动发送。Panorama Skill 只做宿主分流和原样展示，不二次拼装审核材料。

**Trace / Card**
Trace 从已提交 State/Event 生成人类可读历史并可同步飞书。Registry endpoint、字段名、CLI 日志要求和 Workflow Output 字段映射由严格加载的 `internal/traceconfig/registry.yaml` 决定；仓库中的 Lark 坐标是占位符，provider 为 lark 时由 `FANLOOP_TRACE_REGISTRY_URL`、`FANLOOP_TRACE_REGISTRY_BASE_TOKEN`、`FANLOOP_TRACE_REGISTRY_TABLE_ID`、`FANLOOP_TRACE_REGISTRY_VIEW_ID` 覆盖。Runtime 不比较具体 Workflow 或 Output ID。Card 由被接受的 Flow 当前事实独立更新 `.fanloop/card/projection.json`，URL Output 使用 YAML 中的 `output.description` 或 key 展示，渲染时不读取或写入 Trace；显式 `card render` 生成快照，Flow Runtime 不负责远端发送。

**Agent 手册**
每次行动前读取 `flow status`，执行 Prompt/Skills，按 Conditions 判断事实，并把匹配 `available_routes[].route` 原样随 Result 上报。

**Workflow Dashboard**
`fanloop dashboard` 在随机 loopback 端口提供拓扑与完整源码导航。默认只读绑定源；显式绝对 FANLOOP_CONFIG_SOURCE 开启该仓库的编辑。HTTP 与 CLI 共享规范源目录锁，保存要求 allowlist、观察 SHA-256、无符号链接、完整包校验和原子替换。保存直接写回本地源，不进行远端发布。

**Source Anchor**
Dashboard 从当前 live 文件解析出的 allowlisted 相对路径与 1-based inclusive 行范围，只用于在完整文件中
导航和选择真实源码。它不是 Workflow 契约、持久化事实或可独立保存的 YAML 片段。

## 维护者验收

**Maintainer Lifecycle**
`fanloop-maintainer` 已退役（ADR-0135），Fanloop 代码维护改走 `code-change` Workflow；下文记录原四阶段闭环的验收原则，仍适用于代码变更。配置由当前 Agent 连续执行，普通配置自检，风险配置独立试用；代码或混合变更采用以下监督方式。用户先与主 Agent 对齐目录、目标、
范围和验收标准；执行子 Agent持有 Requirement Root、最新 Status、Route 与受管实现。Define 冻结 Acceptance
Set 与 Feature Impact Set；Build 持续实现、运行、诊断和修复；Certify 冻结 HEAD/tree/binary 并完成独立
Review、黑盒验证和主 Agent最终决定。

**Agent Acceptance / Delivery**
冻结候选安装到一次性 `FANLOOP_DATA_HOME`，不切换全局 current。全新 Sub-agent 只使用隔离 CLI、叶子 Help、
Feature Map 和 Acceptance Set 做真实黑盒验收，不读源码、不使用机器人、Botmux 或用户凭据。Feature Map 位于 `.cursor/skills/verify-fanloop/features/`。Deliver 发现
main 前进时只记录新 base 并回 Build，不能修改候选；重新认证后发布唯一 PR、等待 required checks、自动
squash merge，校验 merge tree 等于认证 tree，再把本 Requirement 的源码 worktree 更新到 merge commit，并从该提交本地安装和回读。

**Distribution**
用户通过 GitHub Release 安装 CLI：形如 `vX.Y.Z` 的 tag 在 ubuntu 上交叉编译 darwin/linux amd64/arm64，上传平台 tar.gz、`checksums.txt`、`release.json` 和薄 npm 包装。安装命令是 `npm i -g --foreground-scripts --allow-scripts=fanloop https://github.com/zeefan1555/fanloop/releases/download/vX.Y.Z/fanloop-X.Y.Z.tgz`。`fanloop update` 切换 CLI，且不会把较新的本机版本降级到 `latest`。`fanloop update config` 只同步 Workflow 配置。不使用 npm registry。`go.mod` 要求 Go 1.23.0 或更高版本。`npm run build` 和 `npm run install:local` 仍是本地开发脚本的快捷方式；开发安装离线导入显式干净提交或保留当前快照。

**Maintainer Reports**
代码通道固定产物为 `requirements.md`、`verification-report.md`、`certification-report.md` 和
`delivery-record.md`，按代码 Route 发布阶段飞书报告。配置通道只使用本地 `delivery-record.md` 阅读文档及原始工具证据，不发阶段飞书报告；Panorama 在每个 Step 进入时展示一次。源码、测试或
验证资产变化都会使下游认证与交付事实失效。

## 公开契约

**CommandSpec**
公开叶子命令的生成目录。公开方法、Request/Response、枚举和错误目录以根级模块化 `idl/*.thrift` 为真值；结构化落盘文件以 `idl/storage.thrift` 为真值；`release.json` 以 `idl/release.thrift` 为字段真值；Workflow 推进语义以 live 五份 YAML 为真值。本地源码更新执行 `./scripts/install-local.sh`；配置从绑定或显式指定的本地仓库读取。

**统一结果信封**
成功写 stdout，使用 `ok/data/meta/_notice`；错误写 stderr，包含稳定 type、code、message、hint 和 retryable。

**Contract Golden**
真实 CLI 输出、退出码与文件副作用的受审快照。随机 ID、时间和临时路径仅在比较时规范化。

**Release Manifest**
一次发布中 CLI、统一 Workflow 入口、Storage Schema 支持与文件摘要的唯一声明；字段和局部约束由
`idl/release.thrift` 定义。Workflow YAML 与绑定 Skills 均位于 live 配置根，不进入 Release Manifest。
