# Fanloop 当前技术设计

Fanloop 的产品定位是通用 Loop 引擎：执行图来自配置，Go 代码只提供稳定的解释器与持久化能力。

## 真值与边界

- `idl/*.thrift` 定义公开 CLI、durable storage 与 Workflow YAML 结构；`internal/idl/` 只保存生成物。
- 每套 Workflow 由 `workflow.yaml`、`condition.yaml`、`flow.yaml`、`loop.yaml`、`prompt.yaml`
  五个文件共同定义，Go Runtime 只负责通用加载、校验与执行。
- Agent 提交事实和 RouteSelection；CLI 校验结构、输出类型、互斥条件与唯一 Route，不证明业务事实。
- Flow State 与 Output Registry 是本地事实，Trace 与 Card 是相互隔离的投影。
- `cmd/`、`internal/` 与 `scripts/` 的生产代码不得出现生产 Workflow、Step、Condition、Output 或
  原子 Skill ID；测试会从当前 Bundle 动态提取并扫描这些事实。

源码提供三套首次安装配置：

- `technical-solution-design`：十六步技术文档流程，第 0 至第 10 章各有独立 Step 和产物，按业务问题、技术判断和结果与规划三阶段推进。
- `fanloop-maintainer`：Fanloop 的 4 Stage / 4 Job / 4 Step 维护闭环；Define、Build、Certify 与 Deliver 依次完成验收契约、自主验证、独立认证和 PR/本地交付。
- `material-flashcards`：九步材料理解、知识选择、卡片审核与持久化闭环。

生产目录固定为 `workflows/<workflow-id>/{五份 YAML,skills/<skill-id>/SKILL.md}`。统一入口位于
`entrypoints/fanloop-workflow/`；安装、Runtime 和 Doctor 从同一 live Workflow 根加载，并拒绝缺失或
额外文件、符号链接、空 Skill、未知 Skill 与跨 Workflow SkillBinding。

入口的 `routes.yaml` 使用 `schema_version: 2`，把用户显式选择的场景映射到 Workflow：
`technical-solution` 对应 `technical-solution-design`，`fanloop-maintenance` 对应
`fanloop-maintainer`。没有默认值；用户未选择或场景未知时不得执行 `flow init`。完整决策见
[ADR-0079](./adr/0079-add-technical-solution-design-workflow.md)。

Trace Registry 是独立的部署配置：`internal/traceconfig/registry.yaml` 按 profile 提供默认策略和
可选 Workflow 覆盖。Runtime 只通过 `Resolve(profile, workflowID)` 取得 endpoint、远端字段名、
CLI 日志要求和 Output 字段映射。完整通用化决策见
[ADR-0080](./adr/0080-make-runtime-workflow-agnostic.md)。

## 推进模型

`flow status` 返回当前 Step 的 context、execution、Prompt/Skills、Conditions、available routes、
Workflow 级 common Skills/Conditions、skipped Steps 和有效 Outputs。Agent 只从该响应选择一组
Condition 与一条 Route，再调用：

```text
fanloop flow report progress
fanloop flow report result
```

`when.any_of` 外层为 OR、内层为 AND。Flow 前进到 `next_step_id` 或 terminal；Loop 回到
`back_step_id`。Loop 只失效本次 Result 提交前已经存在、且由目标 Step 或其下游产生的 Outputs；本次
Result 新接受的反馈或交接 Outputs 在失效完成后写入并保留。写命令在同一 Requirement lock 下提交
State、Output Registry 与 Event；dry-run 只计算响应，不落盘。

声明 `step_start` 与 `jump` 的 Workflow 在每次进入 Step 时先进入 `awaiting_confirmation`。此时只展示
开工确认 Prompt、start Route 和面向全部真实 Step 的 jump Routes；人的明确确认形成独立 Event 后，
同一 Step 才进入 `in_progress` 并展示业务 Prompt。Jump 可在任意运行中 Step 使用，向前跨过的 Step
进入 `skipped_step_ids`，目标及下游 Outputs 失效；Card 与 Trace 将 skipped 显示为“已跳过”。完整
决策见 [ADR-0102](./adr/0102-add-common-step-controls.md)。

Human Step 的审核与 Panorama 同样由五份 YAML 驱动。`technical-solution-design` 的三个 Human Step
必须同时具备已回读飞书文档 URL、`panorama_card_published` 与人的明确结论。`fanloop-maintainer` 不包含
Human Step：Define 的需求批准和 Certify 的最终候选验收均由执行子 Agent整理真实产物并向主 Agent申请决定。审批 Skill 组织审核材料并按最早受影响层分类，
Panorama Skill 只按宿主原样展示 renderer 的紧凑投影并返回本次
`panorama_snapshot_path:path`；CLI 只校验 Output 与 Route。Runtime 不调用发送工具，但继续维护本地
Card Projection、显式 Card 渲染以及 Trace provision/sync。完整决策见
[ADR-0086](./adr/0086-align-panorama-with-treeloop.md) 与
[ADR-0087](./adr/0087-allow-agent-approval-at-human-steps.md)、
[ADR-0089](./adr/0089-split-technical-solution-into-reviewed-sections.md)、
[ADR-0098](./adr/0098-use-eleven-section-technical-document-workflow.md)、
[ADR-0100](./adr/0100-align-maintainer-with-treeloop-handoff.md) 与
[ADR-0106](./adr/0106-drive-maintainer-with-main-agent-and-subagents.md) 与
[ADR-0109](./adr/0109-collapse-maintainer-into-four-step-verification-loop.md)。

## 当前配置实例

`technical-solution-design` 用十六个 Step 依次产出业务背景、目标与问题定义、业务特点与技术约束、
方案调研、总体方案、关键模块、技术决策、落地与风险、结果收益、复盘规划和摘要十一个独立片段，
再组装 `technical-solution.md`。摘要最后生成但在最终文档置顶；总体架构图和独立审校报告分别写入
`.technical-solution/architecture.mmd` 与 `.technical-solution/review.md`。

这十六个 Step 共享 required `grill-with-docs`：每一步都先结合已有文档澄清范围并取得人的明确确认；
optional `human-step-jump` 允许人在确认影响后跳到任意 Step。Step 的 ID、名称、顺序和 executor 不变。

业务问题、技术判断和完整文档分别经过强制人工审核，并输出稳定飞书文档 URL。反馈按十一章或
最终呈现中最早受影响的一层回流，目标 Step 及其下游
Output 全部失效；不存在技术方案 Agent 代批路径。

`fanloop-maintainer` 使用 4 Stage / 4 Job / 4 Step：Define 冻结目标与验收契约，Build 持续实现和自主验证，
Certify 先执行独立 Review，确认身份未变后再执行黑盒验收，并记录主 Agent决定及回执；Deliver 合并唯一 PR、校验 tree 并更新本地
CLI。Runtime 仍是单活动 Step，不增加并行状态、IDL 或通用执行层。

用户先与主 Agent 对齐目录和目标；主 Agent派生并持续监督只向其回报的执行子 Agent，执行子 Agent创建、
持有并驱动 Requirement，且直接修改受管代码。Build 运行聚焦测试、`./tests/run-unit` 和真实公开 CLI
自验证；Certify 冻结 `certification_base` / `certification_head`，先完成独立整体 Review，再在身份未变时
由恰好一个无实现上下文的全新
Sub-agent 使用 1 至 3 个公开 CLI 场景做黑盒测试，全程不改全局 current。执行子 Agent向主 Agent申请候选
验收决定；main 前进时 Deliver 记录 `delivery_main_head` 回 Build，由 Build 集成该提交并产生新候选。`merge_and_update_local` 由执行子 Agent发布唯一 PR、从 active main Ruleset 动态读取并校验精确 final head 的
required checks、同步 Review、自动 squash merge，把本 Requirement 的源码 worktree 更新到精确 merge commit，并从该提交更新本地 CLI。已合并重入先固定原 merge commit，合码后的本地失败只恢复该提交。
黑盒公开 CLI 证据只承担可直接观测的 candidate/Doctor、四步 Panorama、Step ID/名称和旧 Step 缺失；
非当前 Step 的 executor 与 Route/Condition 语义由五份生产 YAML 和聚焦 contract test 证明，验收不伪造批准跨过治理边界。

## 当前持久化版本

- Workflow / Flow / Condition / Loop / Prompt：`7 / 5 / 3 / 4 / 2`
- Flow State / Event / Output Registry：`14 / 14 / 4`
- Card Projection / Card Binding / Trace Config / CLI Log：`7 / 2 / 2 / 2`

Requirement 文件集中在 `.fanloop/{flow,output,trace,card,log}`；公开命令、文件位置和恢复提示均不
提供旧产品身份的兼容入口。

## 本地构建与验证

`./scripts/build-local.sh [OUTPUT_DIR]` 只编译本机二进制，生成携带 `bin/fanloop`、统一入口和
Manifest 的本地 Release；标准输出是构建目录的绝对路径。`workflows/` 与 `exemplars/` 不进入 Release，
也不参与 CLI 的 dirty 版本摘要。显式输出目录必须尚不存在，默认在 `dist/` 下新建唯一
`local-*` 目录。

Release Manifest schema 4 以 `cli.binary_sha256` 校验本机二进制，同时固定版本、Storage Schema 支持与
统一入口 SHA-256；没有 Workflow artifact 或平台归档。`./scripts/install-local.sh [OUTPUT_DIR]` 构建后
复用目录安装：首次安装从已校验的配置源播种 `~/.fanloop/current/workflows`，后续升级完整保留其字节；
只有 `~/.fanloop/current/release` 原子切换不可变 CLI Release。稳定的 `current/bin`、`entrypoints` 与
`release.json` 通过该 selector 一起切换。每次 CLI 调用按 State 中的 Workflow ID 重新加载 live YAML 与
Skill，不持久化或比较 Workflow digest。完整决策见
[ADR-0108](./adr/0108-use-live-workflows-and-local-dashboard.md)。

`fanloop dashboard` 复用同一个 live 根，以 Go 标准库提供 loopback-only 的嵌入式编辑器。API 只允许
已有五份 YAML 与 `SKILL.md`，读取返回 SHA-256，保存执行乐观并发、候选 Bundle 校验和原子替换；页面每
两秒轮询，干净编辑器自动刷新，脏编辑器遇到外部变化时标记 stale。它不包含账号、远程监听、Daemon、
创建、删除、重命名或可视化拓扑编辑。

GitHub 托管源码并保留现有代码检查，不承担 npm 或二进制发布；构建不依赖 Node.js、GoReleaser
或 tar/xz。完整决定见 [ADR-0095](./adr/0095-local-source-builds.md)。

新增 Workflow 的默认配置改动只包含一个完整 Workflow 目录和场景路由。配置-only 契约测试会临时
构造第三套 Workflow，并经 Bundle loader、Skill discovery、目录/绑定校验和场景校验完整通过。

仓库级本地门禁只有一个：

```bash
./tests/run-unit
```

它覆盖格式、IDL 新鲜度、静态检查、Go 测试与 Contract。完整 Requirement lifecycle 与生产 Workflow
Route Matrix 由 CI 的 `requirement-e2e` 执行；本地用户表面由隔离候选的 Verification Skill 验证。
