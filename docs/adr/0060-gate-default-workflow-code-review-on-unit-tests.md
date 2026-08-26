---
status: accepted
date: 2026-08-19
amends: ADR-0043
---

# 默认 Workflow 只在单测通过后进入代码评审

默认 `fanloop` Workflow 从 `14.0.0` 起把真实单测结果设为代码评审前的强门禁：
`unit_tests_passed` 从 `check_merge_request_gate` 前进到 `review_code`，
`unit_tests_failed` 回流到 `implement_code`。MR 的其他 Check 仍只记录为 Evidence，不成为
新的 Route Condition。该变化修订 ADR-0043 中“单测 passed/failed 两种真实终态都可继续”
的默认流程语义。

`fanloop@13.0.0` 已发布，按 ADR-0009 保持五份文件和 digest 不变。新增不可变
`fanloop@14.0.0`，并把 `workflows/defaults.json` 的 `fanloop` 默认版本从 13 切到 14；
运行中的 Requirement 继续由 State 固定的历史版本解释，不迁移、不 fallback。

14.0.0 从 13.0.0 复制完整五文件 Bundle，只允许以下差异：

- `workflow.yaml` 只把顶层 `version` 改为 `14.0.0`；Stage、Job、Step 的 ID、名称、
  顺序和 executor 全部不变。
- `flow.yaml` 的 `check_merge_request_gate` 只保留 `[unit_tests_passed] -> review_code`。
- `loop.yaml` 在该 Step 的回实现 Route 中增加
  `[unit_tests_failed] -> implement_code`。
- `prompt.yaml` 只修改 `check_merge_request_gate_flow` 与 `unit_tests_condition` 两段文字，
  使 Agent 指令与新 Route 一致。
- `condition.yaml` 与 13.0.0 逐字节一致；不新增 Condition、Output、Schema 或 Runtime
  判断。

14.0.0 的 12 Step 契约逐项保持不变：

| # | Stage | Step ID | name | executor | 相对 13.0.0 |
|---:|---|---|---|---|---|
| 1 | TechDesign | `bootstrap_techdesign` | 仓库范围确定 | agent | 无变化 |
| 2 | TechDesign | `clarify_requirements` | 需求澄清 | agent | 无变化 |
| 3 | TechDesign | `design_technical_solution` | 方案设计 | agent | 无变化 |
| 4 | TechDesign | `confirm_technical_solution` | 方案审批 | human | 无变化 |
| 5 | Implement | `implement_code` | 编码实现 | agent | 无变化 |
| 6 | Implement | `check_merge_request_gate` | 检查与门禁 | agent | 无变化 |
| 7 | Implement | `review_code` | CodeReview | agent | 无变化 |
| 8 | Implement | `confirm_ready_for_test` | 人工审批 | human | 无变化 |
| 9 | Test | `clarify_test_cases` | 测试用例澄清 | agent | 无变化 |
| 10 | Test | `check_test_environment` | 环境依赖自检 | agent | 无变化 |
| 11 | Test | `execute_test_cases` | 用例执行 | agent | 无变化 |
| 12 | Test | `confirm_test_results` | 测试结果确认 | human | 无变化 |

`douyin-game@1.0.0` 是明确的业务例外，仍记录真实 passed/failed，并允许两种结果进入
代码评审；其五份 YAML 不随本决策修改。MR 内尚未发布的 `social-game@1.0.0` 与新默认
语义重复，直接删除且不进入 Release。`fanloop-maintainer@1.0.0` 的自维护 Route 也不变。

本决策保持 ADR-0038 的五文件 YAML 真值、原子 Condition、OR-of-AND、显式
RouteSelection 与通用 Runtime，保持 ADR-0043 的 12 Step 拓扑，保持 ADR-0056 的需求
文档人工审核门禁。不修改任意 Thrift IDL、生成物、State/Event、Storage、Workflow Schema
或公开 CLI 契约。
