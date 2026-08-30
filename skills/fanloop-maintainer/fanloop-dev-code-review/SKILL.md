---
name: fanloop-dev-code-review
description: 独立审查已完成本地验证的 Fanloop CLI Diff，写 review-report.md。不要求已存在 PR，也不提前执行 Agent 验收。
---

# Code Review

固定比较点为 bootstrap 记录的 main Commit。记录当前 HEAD，读取完整三点 diff、commit 列表、仓库规则、
Issue Workspace 的 requirements.md、Spec/Tickets、local-test-report.md 和相关 ADR；不读取或要求 PR。

逐项检查：

- Standards：仓库规则、根因修复、安全、重复/散弹修改和投机抽象。
- Design/Spec：批准决策、五份生产 YAML 的 Stage/Job/Step/Route 前后对照、验收条件与实现逐项对应；无遗漏、超范围或语义偏差。
- Test Seam / Tracer Bullet：测试只落在批准 Seam；阻断私有实现耦合、内部调用次数断言、同算法重算
  预期、无 Seam 造测试、伪造 red 和水平切片。
- Architecture：复核 CONTEXT、ADR、五份 Workflow YAML、Release 与 Runtime 边界；任何 Thrift diff 都
  必须有编码前精确人工批准。
- Verification：baseline/candidate 必须是同一 Case；local-test-report.md 的 profile、candidate HEAD、
  命令、退出码和测试前后源码状态必须覆盖当前 reviewed HEAD。`.agents/skills/verify-fanloop/` 或测试资产变化
  会使旧报告失效。

真实 Release 安装、独立 Eval、Ruleset/CI 和机器人验收属于 Review 之后，不以其缺失阻断本 Step。

通过或回流都先写 review-report.md，包含 reviewed HEAD、main...HEAD、findings、结论和 ADR impact：

- 无阻断项且工作树干净：review_passed + review_report_written + candidate_head_frozen。
- 需求冲突：requirements_changed + review_report_written。
- 需求不变但 Test Seam、执行路径或技术方案不可靠：
  technical_solution_changes_requested + review_report_written。
- 普通实现阻断：review_failed + review_report_written。

修复或任何候选变化后，旧本地验证与 Review 都失效，必须重新验证和审查。不要因“可以更漂亮”扩大
范围。
