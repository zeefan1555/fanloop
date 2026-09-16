---
status: accepted
date: 2026-09-16
amends: ADR-0063, ADR-0099
supersedes: ADR-0094
---

# 对齐 Treeloop Maintainer 的人工验收与 PR 交接

`fanloop-maintainer` 保持 3 Stage / 3 Job / 9 Step，但从自动合并和本地安装改为技术方案自主评审、
human 端到端验收和 GitHub PR/CI 交接：

~~~text
TechDesign: bootstrap_techdesign -> clarify_requirements -> design_technical_solution -> confirm_technical_solution
Implement:  implement_code -> review_code
Test:       execute_agent_acceptance -> confirm_human_acceptance -> handoff_merge_request
~~~

## Step 契约变更

| 目标序号 | Step ID | ADR-0094 基线 | 目标 | 变化 |
| ---: | --- | --- | --- | --- |
| 1 | `bootstrap_techdesign` | 需求确认/需求闭环/1，工作区准备，agent | TechDesign/TechDesign/1，仓库范围确定，agent | 改名与归组 |
| 2 | `clarify_requirements` | 需求确认/需求闭环/2，需求澄清，agent | TechDesign/TechDesign/2，需求澄清，agent | 归组 |
| 3 | `design_technical_solution` | 研发实现/实现闭环/1，方案设计，agent | TechDesign/TechDesign/3，方案设计，agent | 重排与归组 |
| 4 | `confirm_technical_solution` | 不存在 | TechDesign/TechDesign/4，方案自主评审，agent | 新增 |
| 5 | `implement_code` | 研发实现/实现闭环/2，代码实现，agent | Implement/Implement/1，代码实现与过程 CR，agent | 改名与归组 |
| 6 | `review_code` | 研发实现/实现闭环/3，代码审查，agent | Implement/Implement/2，整体 Code Review，agent | 改名与归组 |
| 7 | `execute_agent_acceptance` | 验收交付/交付闭环/1，Agent 自动化验收，agent | Test/Test/1，Agent 端到端测试，agent | 改名与归组 |
| 8 | `confirm_human_acceptance` | 不存在 | Test/Test/2，人类端到端测试，human | 新增 |
| 9 | `handoff_merge_request` | 不存在 | Test/Test/3，MR 门禁与交接，agent | 新增 |

删除 `confirm_requirements`、`merge_code`、`update_local_cli`。审核结论：删除 3 个、新增 3 个、改名 4 个、
重排 1 个存续 Step、executor 变化 0 个存续 Step；新增的 `confirm_human_acceptance` 明确为 human。

## 推进与交付

需求澄清包含完整计划和真实 human 决定回执；技术方案由 Agent 独立自主评审。实现 Step 完成聚焦检查、
`./tests/run-unit`、`./tests/run-e2e` 和独立整体 CR；Review Step 只核验证据并冻结 `review_base` / `reviewed_head`。

Agent 验收复用 Fanloop 已有的隔离 `FANLOOP_DATA_HOME` 安装，由恰好一个无实现上下文的全新 Sub-agent 只用公开 CLI 验证一至三个场景。之后 Human Step 显示同一候选和验收材料，只接受真实 human 的通过、跳过或反馈；不使用不存在的 `agent2user` / `user2agent` API。

最终 Step 使用 GitHub/main：必要时合入最新 `origin/main` 并完成集成短门禁，幂等创建或更新唯一 PR，回读 final base/head/merge-base，等待精确 head 的 required checks，同步已有 Review 结论并向当前用户交接。不自动 approve、merge、publish 或更新全局 CLI。

human 可在任意运行中 Step 指定九个 Step 中的唯一目标。前向跳转和 self/back 回流都要求 Panorama、本地上下文和真实决定回执；跳转不生成被跨过 Step 的完成事实。为使调用方显式选择不同目标，通用 Flow 歧义校验只拒绝“事实重叠且目标相同”的重复 Route；不同 `next_step_id` 仍由必填 RouteSelection 区分。

最终五文件包含 51 Conditions、38 个 Flow Route objects、45 个 Loop Route objects 和 56 Prompts。不修改 Thrift IDL、Workflow Schema、State/Event/Output Storage 或公开 CLI 字段。

## Live Skill 边界

ADR-0099 确立 `skills/**` 与 `exemplars/**` 为 live 配置。在尚未初始化 Requirement 时，若预期 diff 全部位于这两个目录，且不影响 Workflow YAML、CLI/Runtime、IDL、测试基础设施、构建、安装或发布，则直接修改和聚焦验证，不启动 `fanloop-maintainer`。已初始化的 Requirement 仍按其绑定 Workflow 继续。

本变更修改 Step/Route/Condition 推进语义和通用 Flow 校验，选择 `e2e` 验证档：聚焦 Bundle、Route、Skill 与 Contract 测试，并从同一最终候选运行 `./tests/run-unit`、`./tests/run-e2e` 和无实现上下文的公开 CLI 黑盒验收。

人工审核记录：用户在收到上述九步 Step diff、GitHub/main 适配、不改 Thrift 与不引入不受支持 API 的说明后，于 2026-09-16 回复“确认”。
