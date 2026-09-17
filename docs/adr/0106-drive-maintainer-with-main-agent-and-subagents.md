---
status: accepted
date: 2026-09-17
amends: ADR-0099, ADR-0100
---

# 由主 Agent 监督、执行子 Agent 驱动 Maintainer 自迭代

本 ADR 只部分修订 ADR-0100 中的角色分工和三类决定责任：需求批准、最终候选验收、main 集成确认。
ADR-0100 其余决策仍然有效，包括 GitHub PR/CI 交接、精确候选身份、不自动 approve/merge、
不发布且不更新全局 CLI。

`fanloop-maintainer` 保持 TechDesign / Implement / Test 三阶段、三 Job、九 Step 单线拓扑。用户只与主 Agent
对齐仓库目录、目标、范围和验收标准；主 Agent派生执行子 Agent并持续监督。执行子 Agent创建并持有
Requirement Root，读取 Status、执行 Skills、选择 Route、提交 Result，并直接修改受管代码直至 PR 交接。
主 Agent不创建或驱动 Requirement，也不直接实现。

需求批准、最终候选验收和 main 前进后的集成确认改由执行子 Agent整理证据并向主 Agent申请决定；执行子
Agent不得自批。用户主动改变已对齐目标时才由主 Agent重新对齐。独立整体 Code Review 和候选公开 CLI
黑盒验收仍由不继承实现上下文的全新 Sub-agent执行，执行子 Agent派发、核验产物并提交 Workflow Result。

## Step 契约变更

| 序号 | ADR-0100 | 当前契约 | 变化 |
| ---: | --- | --- | --- |
| 1-7 | 原 Step、名称、顺序与 executor | 不变 | 无 |
| 8 | `confirm_human_acceptance`，人类端到端测试，human | `confirm_main_agent_acceptance`，主 Agent 验收决策，agent | ID、名称、executor |
| 9 | `handoff_merge_request`，MR 门禁与交接，agent | 不变 | 无 |

总数和顺序不变；删除一个旧 Step、增加一个新 Step，executor 变化一次。当前 Bundle 不保留
`confirm_human_acceptance`、`human_acceptance_*`、`human-review.md` 或
`handoff_integration_human_verified` 的兼容名称；旧 Requirement 继续由其不可变旧 Release 解释。

需求决定继续使用通用 `requirements_*` Condition，但决定 actor 是主 Agent、记录和上报者是执行子 Agent。第八步使用
`main_agent_acceptance_passed` 和 `main_agent_acceptance_recorded=main-agent-review.md`；main 集成确认使用
`handoff_integration_main_agent_verified`。最终五文件包含 49 Conditions、38 个 Flow Route objects、45 个
Loop Route objects 和 55 Prompts。

ADR-0100 的 GitHub PR/CI 交接、精确候选身份、不自动合并、不发布和不更新全局 CLI 边界继续有效。本次部分修订
不修改 Thrift IDL、Workflow Schema、Runtime、State/Event/Storage、公开 CLI 字段或其他 Workflow。

本地验证只采用聚焦 Bundle、Route、Step、Skill 和入口契约测试及 `./tests/run-unit`；已退役的
`./tests/run-e2e` 不再是生产 Workflow 门禁。最终候选另从隔离 Release
启动全新 Requirement，由无实现上下文的 Sub-agent只用公开 CLI 验证第八步 ID、名称、executor 和
Panorama。

人工审核记录：用户先回复“确认按上述 YAML / Step diff 实施”，随后在职责边界修订为“主 Agent只监督、
执行子 Agent驱动 Flow 并实现、三类决定回请主 Agent”后回复“继续”。最新 Receipt：
`decision-ac7e4176763535a4b0822f43ca00a8db6163c181379375511b546bd334e68858`。
