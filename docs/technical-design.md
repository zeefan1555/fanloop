# Fanloop 当前技术设计

## 真值与边界

- `idl/*.thrift` 定义公开 CLI、durable storage 与 Workflow YAML 结构；`internal/idl/` 只保存生成物。
- 每套 Workflow 由 `workflow.yaml`、`condition.yaml`、`flow.yaml`、`loop.yaml`、`prompt.yaml`
  五个文件共同定义，Go Runtime 只负责通用加载、校验与执行。
- Agent 提交事实和 RouteSelection；CLI 校验结构、输出类型、互斥条件与唯一 Route，不证明业务事实。
- Flow State 与 Output Registry 是本地事实，Trace 与 Card 是相互隔离的投影。

当前只发布两套 Bundle：

- `promotion-design`：默认入口，依次执行需求澄清、需求人工确认、晋升方案写作、陌生评委审校、方案人工确认。
- `fanloop-maintainer`：Fanloop 自迭代的八步维护流程。

Selector 只执行“显式选择优先，否则 `promotion-design`”，不按仓库或部门分流。完整切换决策见
[ADR-0077](./adr/0077-bootstrap-fanloop-with-promotion-design.md)。

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

## Promotion 领域边界

`promotion-design` 的 `require_points.md`、`方案.md`、`.promotion/brainstorm.md`、
`evidence.md` 与 `.promotion/review.md` 是领域产物。它们不决定当前 Step，也不保存流程
状态；唯一推进状态位于 `.fanloop/flow/state.json`，不会创建 `.promotion/state.json`。

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
执行 Promotion 完整生命周期，并为两套生产 Workflow 动态遍历全部 Flow/Loop Route。
