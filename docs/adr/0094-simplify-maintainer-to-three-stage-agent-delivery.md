---
status: accepted
date: 2026-09-01
amends: ADR-0063, ADR-0066, ADR-0087
supersedes: ADR-0091, ADR-0092
---

# 将维护流程收敛为三阶段 Agent 交付闭环

`fanloop-maintainer` 从 5 Stage / 12 Job / 16 Step 收敛为 3 Stage / 3 Job / 9 Step：

~~~text
需求确认：工作区准备 → 需求澄清 → 需求确认
研发实现：方案设计 → 代码实现 → 代码审查
验收交付：Agent 自动化验收 → 合并 MR → 更新本地 CLI
~~~

Runtime 继续保持单活动 Step；Stage 与 Job 只表达职责和 Panorama 层级，不引入 DAG、并行 durable state
或新的通用执行抽象。

## 生产 Step 契约变更

以下表格完整比较 ADR-0091 对应生产基线与目标。所有目标 Step 的 executor 均为 `agent`：

| 目标序号 | Step ID | 变更前位置与名称 | 目标位置与名称 | 变化 |
| ---: | --- | --- | --- | --- |
| 1 | `bootstrap_techdesign` | `local_verification/requirement_design/1` 工作区准备 | `requirements/requirements/1` 工作区准备 | 归组变化 |
| 2 | `clarify_requirements` | `local_verification/requirement_design/2` 需求澄清 | `requirements/requirements/2` 需求澄清 | 归组变化 |
| 3 | `confirm_requirements` | `local_verification/requirement_design/3` 需求确认 | `requirements/requirements/3` 需求确认 | 归组变化 |
| 4 | `design_technical_solution` | `local_verification/requirement_design/4` 方案设计 | `implementation/implementation/1` 方案设计 | 归组变化 |
| 5 | `implement_code` | `local_verification/implementation/1` 代码实现 | `implementation/implementation/2` 代码实现 | 归组变化 |
| 6 | `review_code` | `feature_intelligence/local_quality/2` 代码审查 | `implementation/implementation/3` 代码审查 | 重排与归组变化 |
| 7 | `execute_agent_acceptance` | `cloud_delivery/robot_acceptance/1` 机器人端到端验收 | `delivery/delivery/1` Agent 自动化验收 | 改名、重排与归组变化 |
| 8 | `merge_code` | `cloud_delivery/automatic_merge/1` 自动合码 | `delivery/delivery/2` 合并 MR | 改名与归组变化 |
| 9 | `update_local_cli` | 不存在 | `delivery/delivery/3` 更新本地 CLI | 新增 |

删除八个 Step：`maintain_verification_skill`、`maintain_feature_map`、`execute_test_cases`、
`coordinate_eval`、`execute_eval_candidates`、`judge_eval`、`publish_candidate`、`verify_ci_gates`。
对应删除项目验证地图以及 create/maintain/verify、三角色 Eval、独立候选发布和独立 CI 门禁 Skill；不保留
别名、兼容层或回退路径。`technical-solution-design` 的 Step、executor 与推进语义完全不变。

审核结论：删除 8 个、增加 1 个、改名 2 个、重排 2 个存续 Step、executor 变化 0 个；所有归组变化
均在上表列出。

## 需求确认与三阶段文档

`confirm_requirements` 保持 Agent Step 和原 `agent_approved` Route。当前 Agent 可以委托一个无实现上下文
的独立 Sub-agent 复核 `requirements.md`，Evidence 必须记录理由与文件引用；只在没有新的产品决策
frontier 时批准。仍需真实决定时保持 blocked，继续使用原 Panorama、人工 approved/rejected、消息与
Evidence Route，不把笼统同意、机器人结论或静默当批准。

三个 Stage 各有稳定本地产物和飞书文档：

- 需求确认：`requirements.md` / `requirement_document_url` / “需求确认报告”；
- 研发实现：`implementation-report.md` / `implementation_document_url` / “研发实现报告”；
- 验收交付：`acceptance-report.md` / `acceptance_document_url` / “验收交付报告”。

同一 Requirement 使用稳定标题：零命中创建、唯一命中更新、多命中 blocked；回流只更新原文档。
创建或更新后必须语义回读正文非空、当前 HEAD 与结论一致，才上报 URL。Panorama 继续使用通用 URL
Output 渲染，根据 YAML `description` 自动展示三阶段文档；不新增 Card 专用代码。

## 验证、合并与本地安装

Review 在最终候选工作树运行聚焦命令、`./tests/run-unit` 和 `./tests/run-e2e`，更新本地和飞书研发实现
报告，工作树干净后冻结 `candidate_head`。候选任何变化都会使验证、Review 和下游验收失效。

`execute_agent_acceptance` 不使用机器人或人工端到端审核。它在一次性 `FANLOOP_DATA_HOME` 和四个临时
Skill Root 中从精确 candidate_head 执行 `npm run install:local`，回读隔离 current、version commit 与
Doctor，同时证明全局 current 未变。随后派发恰好一个无实现上下文的全新 Sub-agent；该 Agent 只获得
隔离 CLI 绝对路径、环境变量和需求中 1 至 3 个公开 CLI 场景及独立预期，为每个场景创建全新
Requirement。它可读取叶子 Help，但不得读取源码、内部 Go、私有 helper、历史 Root、机器人/Botmux、
用户凭据或远端 Trace/Card，也不得修改候选、PR、合并或全局 CLI。产品失败按最早责任层回流；基础设施
故障只 blocked。

`merge_code` 合并候选发布、Ruleset/required checks 与自动合码职责：幂等推送并创建或更新 head 分支到
main 的唯一 PR，要求 Ubuntu/macOS test、requirement-e2e、install-doctor、governance 等 checks 在精确
candidate_head 成功，然后执行：

~~~bash
gh pr merge "$PR_URL" --repo zeefan1555/fanloop --auto --squash --match-head-commit "$CANDIDATE_HEAD"
~~~

禁止 `--admin`、直接 push main、第二个 PR 或人工端到端验收。只有回读 `MERGED` 和非空 merge commit
才进入 `update_local_cli`。

最终 Step 先用从旧验收 Skill 移入的新 Skill 脚本把当前 Requirement 控制器固定到
`bound-release-home`，再验证 merge commit 可达 `origin/main`，从该提交的干净 detached worktree 清除
Fanloop/Botmux 覆盖并运行 `npm run install:local`。全局 current 的 version commit 精确等于 merge commit
且 Doctor healthy 后才上报 `local_cli_updated` 并 terminal；安装失败保持当前 Step blocked，成功后不
回滚 current。

## YAML 与边界

生产五文件包含 31 Conditions、10 Flow Route objects、18 Loop Route objects 和 39 Prompts。每个 Step
必须有至少一条 Loop；最终更新 Step 仅在三份交付证据齐全且发现真实 `requirements_changed` 时回需求
澄清，安装失败本身没有伪造的 failed Condition。Flow 由 `code_merged` 进入 `update_local_cli`，再由
`local_cli_updated` terminal。

不修改 Thrift IDL、Workflow Schema、Runtime、State/Event/Storage、Card Runtime、公开 CLI、GitHub
Actions、Ruleset或两个公开测试入口。本变更选择 `e2e` 验证档：聚焦 Bundle、Route、Panorama、Release
Manifest、安装 Contract，以及最终工作树的 `run-unit`、`run-e2e` 和独立 Sub-agent 公开 CLI 黑盒证据。

人工审核记录：用户在对 3 Stage / 9 Step、Agent 或独立 Sub-agent 需求复核、单 Sub-agent 真实 CLI
验收、唯一 PR 自动合并和 merge commit 本地安装的总改造计划明确回复“同意总改造计划，开始实现”；
随后要求每个阶段产物成为飞书文档并进入 Panorama，又在收到新增 Condition 与 Route 明细后回复
“同意，去做吧”。
