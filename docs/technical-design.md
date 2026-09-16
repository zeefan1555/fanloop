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

当前携带两套 Bundle：

- `technical-solution-design`：十六步技术文档流程，第 0 至第 10 章各有独立 Step 和产物，按业务问题、技术判断和结果与规划三阶段推进。
- `fanloop-maintainer`：Fanloop 的 3 Stage / 3 Job / 9 Step 维护闭环；TechDesign、Implement 与 Test 依次完成方案自主评审、两级验收和 PR/CI 交接。

生产目录严格保持 `workflows/<workflow-id>/ ↔ skills/<workflow-id>/` 一一对应，不设公共
Skill 组例外。统一入口位于 `entrypoints/fanloop-workflow/`；Release 构建和 Doctor 拒绝 live
配置中缺失同名 Skill 组、未知 Skill 组和跨 Workflow SkillBinding。

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
`back_step_id`，并失效目标 Step 及其下游产生的 Outputs。写命令在同一 Requirement lock 下提交
State、Output Registry 与 Event；dry-run 只计算响应，不落盘。

声明 `step_start` 与 `jump` 的 Workflow 在每次进入 Step 时先进入 `awaiting_confirmation`。此时只展示
开工确认 Prompt、start Route 和面向全部真实 Step 的 jump Routes；人的明确确认形成独立 Event 后，
同一 Step 才进入 `in_progress` 并展示业务 Prompt。Jump 可在任意运行中 Step 使用，向前跨过的 Step
进入 `skipped_step_ids`，目标及下游 Outputs 失效；Card 与 Trace 将 skipped 显示为“已跳过”。完整
决策见 [ADR-0101](./adr/0101-add-common-step-controls.md)。

Human Step 的审核与 Panorama 同样由五份 YAML 驱动。`fanloop-maintainer.confirm_human_acceptance` 是最终
human 验收点；需求澄清中的批准同样只接受真实 human 决定，Developer 不得自批。`technical-solution-design` 的三个 Human Step 必须同时具备已回读飞书
文档 URL、`panorama_card_published` 与人的明确结论。审批 Skill 组织审核材料并按最早受影响层分类，
Panorama Skill 只按宿主原样展示 renderer 的紧凑投影并返回本次
`panorama_snapshot_path:path`；CLI 只校验 Output 与 Route。Runtime 不调用发送工具，但继续维护本地
Card Projection、显式 Card 渲染以及 Trace provision/sync。完整决策见
[ADR-0086](./adr/0086-align-panorama-with-treeloop.md) 与
[ADR-0087](./adr/0087-allow-agent-approval-at-human-steps.md)、
[ADR-0089](./adr/0089-split-technical-solution-into-reviewed-sections.md)、
[ADR-0098](./adr/0098-use-eleven-section-technical-document-workflow.md)、
[ADR-0100](./adr/0100-align-maintainer-with-treeloop-handoff.md)。

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

`fanloop-maintainer` 使用 3 Stage / 3 Job / 9 Step：TechDesign 包含仓库范围确定、需求澄清、方案设计和方案自主评审；
Implement 包含代码实现与过程 CR、整体 Code Review；Test 包含 Agent 端到端测试、人类端到端测试和 MR 门禁与交接。
Runtime 仍是单活动 Step，不增加并行状态、IDL 或通用执行层。

实现阶段运行聚焦测试、`./tests/run-unit`、`./tests/run-e2e` 和独立整体 CR；Review 阶段核验证据并冻结
`review_base` / `reviewed_head`。Agent 验收在一次性数据目录安装候选，由恰好一个无实现上下文的全新
Sub-agent 使用 1 至 3 个公开 CLI 场景做黑盒测试，全程不改全局 current。human 验收通过或明确跳过后，
`handoff_merge_request` 发布唯一 PR、校验精确 final head 的 Ruleset/required checks、同步 Review 并交接；不自动合并或更新本地 CLI。

## 当前持久化版本

- Workflow / Flow / Condition / Loop / Prompt：`7 / 5 / 3 / 4 / 2`
- Flow State / Event / Output Registry：`13 / 13 / 3`
- Card Projection / Card Binding / Trace Config / CLI Log：`6 / 2 / 2 / 2`

Requirement 文件集中在 `.fanloop/{flow,output,trace,card,log}`；公开命令、文件位置和恢复提示均不
提供旧产品身份的兼容入口。

## 本地构建与验证

`./scripts/build-local.sh [OUTPUT_DIR]` 只编译本机二进制，生成携带 `bin/fanloop`、统一入口和
Workflow 的本地目录；标准输出是构建目录的绝对路径。`skills/` 与 `exemplars/` 不进入 Release，
也不参与 CLI 的 dirty 版本摘要。显式输出目录必须尚不存在，默认在 `dist/` 下新建唯一
`local-*` 目录。

Release Manifest schema 3 以 `cli.binary_sha256` 校验本机二进制，同时固定版本、组件路径及
统一入口/Workflow SHA-256；没有平台归档。`./scripts/install-local.sh [OUTPUT_DIR]` 构建后复用
目录安装，验证和 Doctor 成功才切换 `~/.fanloop/current`，并原子维护
`~/.fanloop/config/current -> <源码仓库>`。Status 每次从该 live 配置根解析原子 Skill 的绝对路径，
因此 Skill 和范文随源码拉取即时更新，不改变 CLI 版本。Workflow、入口与二进制仍按 Release 固定；
已有 Requirement 的 Workflow digest 和固定控制器语义不变。完整决策见
[ADR-0099](./adr/0099-load-skills-from-live-configuration.md)。

GitHub 托管源码并保留现有代码检查，不承担 npm 或二进制发布；构建不依赖 Node.js、GoReleaser
或 tar/xz。完整决定见 [ADR-0095](./adr/0095-local-source-builds.md)。

新增 Workflow 的发布改动只包含五份 YAML、同名 Skill 组和场景路由。配置-only 契约测试会临时
构造第三套 Workflow，并经 Bundle loader、Skill discovery、目录/绑定校验和场景校验完整通过。

仓库级门禁只有两个：

```bash
./tests/run-unit
./tests/run-e2e
```

前者覆盖格式、IDL 新鲜度、静态检查、Go 测试与 Contract；后者从当前工作树构建一次 CLI，
执行技术方案完整生命周期，并为两套生产 Workflow 动态遍历全部 Flow/Loop Route。
