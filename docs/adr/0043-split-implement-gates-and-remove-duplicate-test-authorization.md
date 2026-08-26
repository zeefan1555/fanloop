---
status: accepted
date: 2026-08-17
amends: ADR-0038
amended_by: ADR-0044, ADR-0060
---

# 拆分 Implement 门禁并移除重复 Test 授权

Workflow 8.0.0 将 Implement 的单测采集与 Aime Code Review 拆为 `check_merge_request_gate` 和 `check_aime_code_review` 两个连续 Agent Step。前者复用 `unit_tests_passed`/`unit_tests_failed`，两种真实终态都可继续；后者复用 `code_review_passed`/`code_review_failed`，通过时前进到 `confirm_ready_for_test`，失败时回到 `implement_code`。MR 的其他 Check、Aime 任务链接、Verdict 和评论详情只记录为 Evidence，不成为新 Condition 或必填 Output。

`confirm_ready_for_test` 继续作为进入 Test 前的 Human Gate，因此 Test 阶段删除重复的 `authorize_test_execution` Step，以及 `test_execution_authorized`、`test_execution_rejected`、`test_execution_authorization_recorded` 三个未再引用的 Condition。环境就绪后直接进入接口用例执行；测试结果仍由 `confirm_test_results` 人工确认。

完整主链固定为 12 个 Step：TechDesign 的仓库确定、需求澄清、方案设计、方案确认；Implement 的编码实现、MR 远端门禁、Aime Code Review、进入 Test 确认；Test 的接口用例澄清、环境与依赖检查、接口用例执行、接口用例确认。Card 与 Status 继续从五份 Workflow YAML 投影这一顺序，不承载专用业务流程。

本决策仅修订 ADR-0038 中“删除独立 CR Step、单测与 Aime 合并上报”的生产拓扑，以及测试授权相关生产配置。ADR-0038 的原子 Condition、OR-of-AND、Agent 显式 RouteSelection、CLI 结构校验、Output 失效和本地事实边界继续有效；ADR-0039 的 Card/Output/Trace 存储隔离不变。Workflow 8.0.0 直接替换源码和新发布包中的 7.0.0 当前 Bundle，不提供兼容 Route、alias 或迁移工具；运行中旧 Requirement 继续由其已安装 Release 解释。
