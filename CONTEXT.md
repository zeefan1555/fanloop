# Fanloop 领域词典

## 流程配置

**Requirement**
一次需要持续推进和审计的研发需求。事实保存在 Requirement 自己的 `.fanloop/`。

**Workflow Bundle**
不可拆分、不可变、可版本化的五文件流程包：`workflow.yaml` 注册 Stage/Job/Step，`flow.yaml` 声明正常 Route，`condition.yaml` 声明原子 Condition 与 OutputSpec，`loop.yaml` 声明回流 Route，`prompt.yaml` 保存 Prompt 与 SkillBinding。Release 和 State 绑定五文件规范化语义的同一个 digest。

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
由 State 和绑定 Bundle 派生。运行中返回扁平 CurrentContext、execution、Prompt/Skills、当前相关 Conditions、统一 `available_routes` 和有效 Outputs；完成后不含 current。

## 当前事实与投影

**State / Output Registry**
恢复执行的当前事实分开落盘：`.fanloop/flow/state.json` 保存 Requirement、Release/Workflow 绑定、活动 Step、Summary/Evidence、integration 和最后 Event 指针；`.fanloop/output/state.json` 保存当前有效 RegisteredOutputs。当前为 State/Event Schema 12、Output Registry Schema 3；字段、枚举和版本以 `idl/storage.thrift` 为唯一可编辑真值。

**Event**
不可变审计记录。Event Schema 12 使用 Thrift `EventPayload` union；落盘的 `kind` 与唯一 payload 成员一一对应，例如 `flow_progressed`、`flow_result`。Result Event 保存原始 ConditionResults、Evidence、Summary、Effect、Transition 和 accepted/invalidated OutputChanges。dry-run 和 rejected request 不产生 Event。

**CLI Execution Log**
Requirement 范围的完整诊断 transcript。每个具有有效绝对 `--root` 的真实叶子调用在结束后 best-effort 向 `.fanloop/log/cli.jsonl` 追加一条 schema 2 Thrift JSON，包括有序 arguments、实际消费的 stdin、完整 stdout/stderr、只读、失败和 `--dry-run`；`--input @file` 只记录参数中的路径，不把文件内容算作 stdin。Help 与无 Root 的 bootstrap/版本调用不记录。日志不脱敏、不截断，可能包含 URL、Evidence、密钥或 token；目录/文件权限保持 `0700/0600`。它不参与恢复、路由或 Doctor 健康判断，写入失败也不改变命令结果；`fanloop-maintainer` production 会由 `trace sync` 把同步开始前的完整字节快照投影到独立飞书文档。

**Output 失效**
Loop 根据 State 中 `producer_step_id` 失效由 back Step 及其下游生产的有效 Output，保留更早 Step 的事实。

**Trace / Card**
Trace 从已提交 State/Event 生成人类可读历史并可同步飞书。`fanloop-maintainer` production 的同一 Integration 绑定 Trace 与 CLI 日志两份文档，并把需求澄清、技术方案、MR、CLI 日志投影到专用 Registry；普通 Workflow 与 test Registry 保持原契约。Card 由被接受的 Flow 当前事实独立更新 `.fanloop/card/projection.json`，在全局证据区展示两份文档链接，渲染时不读取或写入 Trace；远端失败不回滚本地事实。

**Agent 手册**
每次行动前读取 `flow status`，执行 Prompt/Skills，按 Conditions 判断事实，并把匹配 `available_routes[].route` 原样随 Result 上报。

## 公开契约

**CommandSpec**
11 个公开命令的生成目录。公开方法、Request/Response、枚举和错误目录以根级模块化 `idl/*.thrift` 为真值；结构化落盘文件以 `idl/storage.thrift` 为真值；`release.json` 以 `idl/release.thrift` 为字段真值；Workflow 推进语义以五份 YAML 为真值。`fanloop update` 是 npm launcher 安装控制，不属于该 Thrift 目录。

**统一结果信封**
成功写 stdout，使用 `ok/data/meta/_notice`；错误写 stderr，包含稳定 type、code、message、hint 和 retryable。

**Contract Golden**
真实 CLI 输出、退出码与文件副作用的受审快照。随机 ID、时间和临时路径仅在比较时规范化。

**Release Manifest**
一次发布中 CLI、Skills、完整 Bundle、Schema 版本与文件摘要的唯一声明；字段和局部约束由 `idl/release.thrift` 定义，安装和更新只切换完整配套版本。
