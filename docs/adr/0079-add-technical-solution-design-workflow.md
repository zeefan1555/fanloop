---
status: accepted
date: 2026-08-29
supersedes_in_part: ADR-0057, ADR-0058, ADR-0059, ADR-0061, ADR-0066, ADR-0077
---

# 以场景选择的七步技术方案 Workflow 取代 Promotion

当前 Release 只发布 `technical-solution-design` 与 `fanloop-maintainer` 两套五文件 Bundle。
删除 `promotion-design` 的生产 YAML、Skill 与当前文档入口；历史 ADR 只保留为审计记录，
不构成可加载的兼容路径、别名、迁移器或 fallback。

`technical-solution-design` 固定为三阶段七 Step：

1. Problem：Agent `frame_technical_problem`、Human `confirm_technical_problem`；
2. Solution：Agent `derive_technical_solution`、Human `confirm_solution_direction`；
3. Document：Agent `write_technical_solution`、Agent `review_technical_solution`、Human
   `confirm_technical_solution`。

每个 Step 只绑定一个全新 Skill，依次为 `technical-problem-framing`、
`technical-problem-approval`、`technical-solution-derivation`、
`technical-direction-approval`、`technical-solution-writing`、
`technical-solution-review` 和 `technical-solution-approval`。Human Step 的 Skill 只组织
自包含审核材料、约束判定口径和回流分类；批准必须来自人的明确回复，完整回复进入 Flow Event
Evidence。

领域产物为 `.technical-solution/problem.md`、`.technical-solution/proposal.md`、
`technical-solution.md`、`.technical-solution/architecture.mmd` 和
`.technical-solution/review.md`。问题定义变化回到 `frame_technical_problem`；核心模型、选型或
总体架构方向变化回到 `derive_technical_solution`；仅写作、图或审校问题回到
`write_technical_solution`。Runtime 继续按目标 Step 失效下游 Output，不新增领域 State 或
审批文件。

生产目录采用唯一结构：

```text
workflows/<workflow-id>/
skills/<workflow-id>/<skill-id>/
entrypoints/fanloop-workflow/{SKILL.md,routes.yaml}
```

`workflows/` 与 `skills/` 的一级业务目录集合必须完全相同；每个 Workflow 只能引用同名
`skills/<workflow-id>/` 组。未知 Skill 组、缺失同名组和跨 Workflow 绑定阻断 Release 构建。
统一入口不属于任何 Workflow，独立位于 `entrypoints/`。这取代
`skills/fanloop-workflow/{common,<workflow-id>}`、`skills/self-iteration` 和 `skills/common`
三条旧路径，但不改变 Skill ID、Manifest 字段或安装器只全局暴露 `fanloop-workflow` 的行为。

`entrypoints/fanloop-workflow/routes.yaml` 使用 `schema_version: 2` 的场景配置。用户必须显式选择
`technical-solution` 或 `fanloop-maintenance`，再分别映射到
`technical-solution-design` 或 `fanloop-maintainer`。Selector 没有默认 Workflow；未选择、
未知场景、缺失映射或映射目标不在配套 Release 时停止，不执行 `flow init`，不按仓库、部门或
其他上下文猜测。

本决策保留 ADR-0006 的 Skill/Workflow/Runtime 职责分离、ADR-0038 的原子 Condition 与显式
Route、ADR-0052 的 Thrift-first YAML authoring、ADR-0054 的 Manifest 契约以及 ADR-0062 的
每 ID 一套当前 Bundle。不修改 `idl/*.thrift`、Workflow YAML Schema、Runtime、State/Event
或公开 CLI。
