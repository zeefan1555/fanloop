---
status: accepted
date: 2026-08-17
amends: ADR-0043
amended_by: ADR-0047
---

# 用 Agent Code Review 取代 Aime 结果采集

Workflow 9.0.0 保留 Implement 的独立代码评审 Step，但将 `check_aime_code_review` 直接替换为 `review_code`。该 Step 不再读取 Codebase 上已有的 Aime CheckRun，而是使用随 Fanloop Release 发布的 `fanloop-code-review` Skill，对每个 Merge Request 的 diff 与已确认技术设计执行真实审查。

代码评审使用 `Approve`、`Recommend`、`Block` 三态 Verdict。多 MR 以 `Block > Recommend > Approve` 聚合：`Block` 回到 `implement_code`；`Approve` 和 `Recommend` 前进到 `confirm_ready_for_test`，其中 `Recommend` 必须由 Human Gate 明确接受报告中的风险或后续项。MR、设计来源或读取工具不可用时保持当前 Step blocked，不形成路由事实。报告、发现与 Verdict 文件作为 Evidence，不成为新的必填 Output。

正式 Skill 使用 `fanloop-code-review` 名称，避免覆盖用户拥有的通用 `code-review` Skill。它保留确认上游 `webcast/social_skills_repo/tree_model_sdd/code-review@065e6ee3077f5c1e9fa282e5d31c6fe34674562d` 中 MR 审查、设计追溯、风险检查、报告和 Verdict 契约，删除本 Workflow 不使用的 local 模式以及退役 Driver 编排。不保留 `codebase-aime-cr-collector`、Aime alias、fallback、双读或兼容 Route。

本决策只取代 ADR-0043 中 Aime 专用 Step 名称、采集 Skill 和 `passed|failed` 二态结论。ADR-0043 的独立单测门禁、独立 CR Step、进入 Test 前 Human Gate 和 12 Step 主链继续有效；ADR-0038 的五文件 Bundle、原子 Condition、OR-of-AND、显式 RouteSelection、Output 失效和通用 Runtime 继续有效。Workflow 9.0.0 直接替换新源码与新 Release 中的 8.0.0 当前 Bundle；运行中的旧 Requirement 继续由其已安装 Release 解释。

真实生产 YAML 的变更前后示例、影响说明与人工确认记录见 `docs/specs/code-review-skill-cutover.md`。
