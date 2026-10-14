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

second-brain-skills main 的 fanloop-workflow/ 托管生产配置，主要包括：

- `playbook-flow`：通用 8 节点入口；代码变更走 `code-change` playbook（取代已退役的 `fanloop-maintainer` 与 `remnote-direct-maintainer`，见 ADR-0135）。
- `pstack-native`：六步剧本交付，保留完整脚本与资源。
- `remnote-flashcards`：四步 HOME 分类、RemNote 闪卡写入、Wiki 更新与引用、最终回读闭环；查重后允许零新增卡片。

完整包目录为 `fanloop-workflow/<workflow-id>/{SKILL.md,五份 YAML,skills/<skill-id>/**}`。本地 Release 暴露
`fanloop-workflow` 与 `create-fanloop-workflow` 两个公共 Skill；安装、Runtime 和 Doctor 从同一 live Workflow 根加载，并拒绝缺失或
额外文件、符号链接、空 Skill、未知 Skill 与跨 Workflow SkillBinding。

入口的 `routes.yaml` 使用 `schema_version: 2`，把用户显式选择的场景映射到 Workflow：
`fanloop-maintenance` 对应
`playbook-flow`，`remnote-flashcards` 对应 `remnote-flashcards`。没有默认值；用户未选择或场景未知时不得执行 `flow init`。当前生产来源与
清单见 [ADR-0132](./adr/0132-git-workflow-configuration-center.md)；`technical-solution-design` 的历史退役决策见
[ADR-0120](./adr/0120-host-production-workflows-in-fornax.md)。

Trace Registry 是独立的部署配置：`internal/traceconfig/registry.yaml` 按 profile 提供默认策略和
可选 Workflow 覆盖。Runtime 只通过 `Resolve(profile, workflowID)` 取得 provider、endpoint、远端字段名、
CLI 日志要求和 Output 字段映射。完整通用化决策见
[ADR-0080](./adr/0080-make-runtime-workflow-agnostic.md)。`provider: none` 时 `flow init` 不建 Trace 文档，
Registry 投影跳过，见 [ADR-0133](./adr/0133-trace-registry-provider-switch.md)。production profile 现已全部为
`provider: none`（飞书投影关闭，运行记录改由 Notion run-ledger 承担），test profile 保留飞书测试表，见
[ADR-0134](./adr/0134-turn-off-lark-trace-projection.md)。

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

Human Step 的审核与 Panorama 由五份 YAML 驱动。代码通道（`code-change` playbook）由主 Agent批准需求与最终候选，配置通道经独立行为试用后提交本地完整包并回读。Panorama Skill 按宿主原样展示 renderer 输出，并返回本次快照；CLI只校验类型与Route，不代替审批或证明业务事实。

## 当前配置实例

已退役的 `fanloop-maintainer`（ADR-0135）曾使用 4 Stage / 4 Job / 4 Step：Define 冻结目标与验收契约，Build 持续实现和自主验证，
Certify 在 Pre-Certify 后并行执行独立 Review 与黑盒验收，等待两者结束再记录主 Agent决定及回执；Deliver 合并唯一 PR、校验 tree 并更新本地
CLI。Runtime 仍是单活动 Step，不增加并行状态、IDL 或通用执行层。

用户先与主 Agent 对齐目录和目标；主 Agent派生并持续监督只向其回报的执行子 Agent，执行子 Agent创建、
持有并驱动 Requirement，且直接修改受管代码。Build 运行聚焦测试、`./tests/run-unit` 和真实公开 CLI
自验证；提交并确认 clean 后，Pre-Certify 一次性复核契约、完整 diff、验证资产、外部依赖与 candidate identity；
涉及生产配置时身份同时包含 `configuration_manifest_sha256`。
Certify 冻结 `certification_base` / `certification_head` 后，并行启动独立整体 Review 和恰好一个无实现上下文的全新
Sub-agent 黑盒验证；二者读取同一候选、使用隔离写状态且全部结束后再聚合。黑盒使用 1 至 3 个公开 CLI 场景，
全程不改全局 current。执行子 Agent向主 Agent申请候选
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

`./scripts/build-local.sh [OUTPUT_DIR]` 只编译本机二进制，生成携带 `bin/fanloop`、两个公共 Skill 和
Manifest 的本地 Release；标准输出是构建目录的绝对路径。`skills/` 与 `exemplars/` 不进入 Release，
也不参与 CLI 的 dirty 版本摘要。显式输出目录必须尚不存在，默认在 `dist/` 下新建唯一
`local-*` 目录。

Release Manifest schema 4 以 `cli.binary_sha256` 校验本机二进制，同时固定版本、Storage Schema 支持与
两个 Skill 的 SHA-256；没有 Workflow artifact 或平台归档。`./scripts/install-local.sh [OUTPUT_DIR]` 构建后
复用目录安装，离线切换 CLI Release；显式配置源导入干净提交快照，否则保留已有配置。

`$FANLOOP_DATA_HOME/config/current` 指向不可变 Git 配置快照。flow init 在稳定锁内拉取 main、校验整集、记录配置身份并原子激活；dry-run 只校验。已有 flow status/report/skills 离线读取。显式 FANLOOP_CONFIG_SOURCE 指向作者仓库根并跳过刷新。完整包目录是 fanloop-workflow/<id>。配置失败不创建 Requirement，也不替换 current。

`fanloop dashboard` 提供 loopback-only 文件与拓扑，保持源码锚点、缩放/聚焦/选择与轮询。默认绑定源只读；显式绝对 `FANLOOP_CONFIG_SOURCE` 才开启该仓库保存。写入只接受 allowlist 内完整文件，校验观察 SHA-256、无符号链接和整个候选 Bundle 后原子替换；可编辑当前绑定仓库，不调用远端配置服务。
五YAML仍是通用加载器的唯一执行图，配置内容不进入Requirement或CLI Release。
本地源绑定、完整包与 Dashboard 边界见 [ADR-0132](./adr/0132-git-workflow-configuration-center.md)。

GitHub 只托管源码、PR 和 CI；不构建或发布 npm、跨平台归档或其他分发制品。合并后从精确 merge commit 本地构建、安装并回读版本、Doctor 与 smoke。生产 Workflow 在 second-brain-skills 独立维护；只有新的 `flow init` 才会拉取、校验并激活远端更新，已有 `flow status/report/skills` 继续离线读取当前不可变快照。完整决定见
[ADR-0095](./adr/0095-local-source-builds.md) 与 [ADR-0132](./adr/0132-git-workflow-configuration-center.md)。

新增生产 Workflow 直接增加 fanloop-workflow/<workflow-id>/ 完整包。统一入口的场景映射仍在代码routes.yaml中；新增入口场景需要代码发布，本次不提供动态场景发现。通用代码测试用独立fixture验证额外Workflow无需运行时注册。

仓库级本地门禁只有一个：

```bash
./tests/run-unit
```

它覆盖格式、IDL 新鲜度、静态检查、Go 测试与 Contract。完整 Requirement lifecycle 与独立fixture的
Route Matrix 由 CI 的 `requirement-e2e` 执行；固定生产发布的矩阵由 `scripts/validate-configuration.py` 执行。本地用户表面按 `.cursor/skills/verify-fanloop/SKILL.md` 用隔离候选验证。

`run-unit` 先完成格式与生成物检查，再并发执行互不共享可写状态的 vet、Go test 和本地 Route Matrix；
它等待所有已启动检查并聚合失败。脚本显式使用 Go 自动工具链选择，避免宿主 `GOTOOLCHAIN=local` 让仓库门禁在运行前失败。

`flow skills` exposes the complete current Workflow Skill catalog for dynamic reading; membership does not make a Skill mandatory. The catalog decision is recorded in [ADR-0128](./adr/0128-discover-workflow-skills-and-retire-pstack-native.md); the current four-package inventory and the `pstack-native` restoration override are recorded in [ADR-0132](./adr/0132-git-workflow-configuration-center.md).
