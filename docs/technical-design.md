# Fanloop 当前技术设计

## 真值与边界

- `idl/*.thrift` 定义公开 CLI、durable storage 与 Workflow YAML 结构；`internal/idl/` 只保存生成物。
- 每套 Workflow 由 `workflow.yaml`、`condition.yaml`、`flow.yaml`、`loop.yaml`、`prompt.yaml`
  五个文件共同定义，Go Runtime 只负责通用加载、校验与执行。
- Agent 提交事实和 RouteSelection；CLI 校验结构、输出类型、互斥条件与唯一 Route，不证明业务事实。
- Flow State 与 Output Registry 是本地事实，Trace 与 Card 是相互隔离的投影。

当前发布两套 Bundle：

- `technical-solution-design`：独立七步技术方案流程，每个 Step 绑定一个专用 Skill，按问题定义、方案推导和方案成文三阶段推进。
- `fanloop-maintainer`：Fanloop 自迭代的八步维护流程。

生产目录严格保持 `workflows/<workflow-id>/ ↔ skills/<workflow-id>/` 一一对应；
`skills/common/` 仅承载统一入口与 Selector。Release 构建拒绝缺失同名 Skill 组、未知业务
Skill 组和跨 Workflow SkillBinding。

Selector 的内联 `schema_version: 2` 配置把用户显式选择的场景映射到 Workflow：
`technical-solution` 对应 `technical-solution-design`，`fanloop-maintenance` 对应
`fanloop-maintainer`。没有默认值；用户未选择或场景未知时不得执行 `flow init`。完整决策见
[ADR-0079](./adr/0079-add-technical-solution-design-workflow.md)。

## 推进模型

`flow status` 返回当前 Step 的 context、execution、Prompt/Skills、Conditions、available routes 和
有效 Outputs。Agent 只从该响应选择一组 Condition 与一条 Route，再调用：

```text
fanloop flow report progress
fanloop flow report result
```

`when.any_of` 外层为 OR、内层为 AND。Flow 前进到 `next_step_id` 或 terminal；Loop 回到
`back_step_id`，并失效目标 Step 及其下游产生的 Outputs。写命令在同一 Requirement lock 下提交
State、Output Registry 与 Event；dry-run 只计算响应，不落盘。

## Technical Solution 领域边界

`technical-solution-design` 先冻结 `.technical-solution/problem.md`，再通过独立人工门禁确认
`.technical-solution/proposal.md` 的选型与总体架构方向，最后生成 `technical-solution.md`、
`.technical-solution/architecture.mmd` 和 `.technical-solution/review.md`。七个 Step 分别绑定
`technical-problem-framing`、`technical-problem-approval`、`technical-solution-derivation`、
`technical-direction-approval`、`technical-solution-writing`、`technical-solution-review` 和
`technical-solution-approval`。

问题定义变化回到第一步并失效全部下游 Output；方向变化回到方案推导；写作或审校问题只回到
方案写作。Human Step 的结论保存在 Flow Event/Evidence，不创建第二套流程状态或审批状态。

## 当前持久化版本

- Workflow / Flow / Condition / Loop / Prompt：`7 / 4 / 2 / 4 / 1`
- Flow State / Event / Output Registry：`12 / 12 / 3`
- Card Projection / Card Binding / Trace Config / CLI Log：`5 / 2 / 2 / 2`

Requirement 文件集中在 `.fanloop/{flow,output,trace,card,log}`；公开命令、文件位置和恢复提示均不
提供旧产品身份的兼容入口。

## Release 与验证

一个 Release 原子携带 `bin/fanloop`、两套 Workflow 和它们引用的 Skills。Release Manifest 固定
版本、组件路径与 SHA-256；安装验证成功后才切换 `~/.fanloop/current`。

仓库级门禁只有两个：

```bash
./tests/run-unit
./tests/run-e2e
```

前者覆盖格式、IDL 新鲜度、静态检查、Go/npm 测试与 Contract；后者从当前工作树构建一次 CLI，
执行技术方案完整生命周期，并为两套生产 Workflow 动态遍历全部 Flow/Loop Route。
