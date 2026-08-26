---
status: accepted
date: 2026-08-18
amends: ADR-0022
amended_by: ADR-0063, ADR-0078
---

# 使用本地测试与 Requirement E2E 两个仓库入口

Fanloop 仓库只暴露两个开发测试入口：`./tests/run-unit` 与
`./tests/run-e2e`。测试函数继续由所属 Go、Node 或 Python 模块维护，不为统一执行
搬入中央测试框架。

`run-unit` 是本地确定性门禁：复用现有格式、IDL 生成物新鲜度和 `go vet` 检查，
通过 `go test -short ./...` 运行全部非 Requirement E2E Go 测试，并运行 npm 与
Route Matrix runner 的函数测试。Codebase MR、Publish 与 GitHub CI 统一调用这一
入口，不再维护平行命令集合。

`run-e2e` 是 Requirement 行为验收：从当前工作树构建一次真实 CLI，只 mock
`lark-cli` 等外部系统，依次完成默认业务 Workflow 的 12 Step 生命周期，并对
`workflows/defaults.json` 声明的每个生产 Workflow 执行从 `flow status` 动态发现的
Flow/Loop Route Matrix。它允许测试未提交改动，但必须记录 commit、dirty 状态和
二进制摘要，并验证执行前后源码状态不变。任一推进、回流、拒绝、dry-run、持久化、
Trace、Card 或 Doctor 断言失败都返回非零并保留审计现场。

任何改变产品或测试行为的代码变更在提交前都必须运行两个入口并记录结果。默认远端
MR 门禁只自动运行 `run-unit`；Requirement E2E 作为开发者必须提供的显式验收证据，
后续若接入远端任务也只能调用同一入口。

删除 `scripts/test.sh`、`scripts/run-workflow-demo-e2e` 与
`scripts/run-route-matrix-e2e`，不保留 alias。发布后的匿名 npm 安装验证属于发布
smoke，不是第三个开发测试入口。

本决策修订 ADR-0022 中“每个 MR 运行全部确定性门禁”的执行边界：默认 MR 门禁由
`run-unit` 承担，完整 Requirement E2E 由 `run-e2e` 显式执行。ADR-0023 的最小
测试 seam、ADR-0043 的 12 Step 主链、ADR-0045 的生产源码验证边界保持不变。
