---
name: fanloop-dev-code-review
description: 对冻结 certification_head 执行不继承实现上下文的独立整体 Review，并返回原始结论。
---

# Independent Code Review

只读 requirements.md、相关 ADR、verification-report.md 和完整 diff。核验 Acceptance Set、Feature Impact
Set、聚焦测试、`./tests/run-unit`、验证资产维护、候选身份与 clean 状态全部覆盖同一候选。

Review 结果只能是 passed 或 failed。任何 P0/P1、未关闭 finding、验证缺口、契约不一致或身份漂移都返回
failed，并附文件与行号；不得在本任务中修改代码、测试、报告或验证资产。通过时返回原始 Review receipt，
供执行子 Agent写入 certification-report.md。
