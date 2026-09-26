---
status: accepted
date: 2026-09-26
amends: ADR-0063, ADR-0109, ADR-0116
---

# 自迭代采用一次冻结的短反馈闭环

Fanloop 代码自迭代在现有 Define / Build / Certify / Deliver 四步内采用统一短反馈闭环，不增加 Step、
Workflow Runtime、状态、测试入口或调度系统。

Define 记录 `low|medium|high` 风险，并冻结 Acceptance Set、Feature Impact Set、依赖、权限、副作用和停止位置；
风险不确定时升级为 `high`。Build 按可独立验证的小批纵向切片推进，每批运行最小聚焦检查。只有准备冻结的
稳定候选执行完整 `./tests/run-unit` 与公开 CLI 验证。

本地 manifest 为 `passed` 后执行一次 Pre-Certify，复核完整 diff、ADR/契约、生成物、Feature Map、外部依赖、
clean 状态与 candidate identity。通过后，独立 Code Review 与无实现上下文的黑盒验证可以并行，但二者必须
读取同一冻结身份、使用隔离写状态并全部结束后聚合。任一候选内容变化使本地证据和两项认证全部失效。

认证通过后才发布生产配置、创建或更新唯一 PR、等待同一 head 的 required checks 并合并。交付记录追加
lead time、Build/Certify 次数、返工来源和对应改进；只沉淀重复、可执行且可验证的规则。

唯一 `./tests/run-unit` 入口先完成格式与生成物检查，再并发执行互不共享可写状态的 vet、Go test 和本地
Route Matrix，等待全部检查并聚合失败。它显式使用 Go 自动工具链选择，避免宿主环境固定旧工具链导致门禁
无法启动。ADR-0078 的隔离原则、ADR-0103/0114 的单入口原则、ADR-0120 的 Fornax 配置真值和 ADR-0122
的本地源码分发保持不变。
