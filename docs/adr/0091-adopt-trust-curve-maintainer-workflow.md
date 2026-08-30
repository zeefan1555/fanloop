---
status: accepted
date: 2026-08-31
amends: ADR-0063, ADR-0066, ADR-0072, ADR-0076, ADR-0087
supersedes: ADR-0090
---

# 将维护流程改造成可验证、可评测、硬门禁和自动合码的信任曲线

`fanloop-maintainer` 从 3 Stage / 9 Step 改为 5 Stage / 12 Job / 16 Step：

~~~text
本地验证机制：工作区准备 → 需求澄清 → 需求确认 → 方案设计 → 代码实现 → 验证技能维护
功能图谱：功能地图维护 → 本地验证 → 代码审查
Agent 评测：评测编排 → 子 Agent 执行 → 独立裁判
硬性门禁：发布候选 PR → CI 硬门禁
云端交付：机器人端到端验收 → 自动合码
~~~

原九个 Step ID 与相对顺序保留；新增 `maintain_verification_skill`、`maintain_feature_map`、
`coordinate_eval`、`execute_eval_candidates`、`judge_eval`、`publish_candidate`、`verify_ci_gates`，不删除
Step。`confirm_requirements` executor 从 human 改为 agent；仍缺真实产品决策时 Agent 保持 blocked 并
向人询问，不把例行人工确认放在主路径。`merge_code` 仅把展示名从“合码”改为“自动合码”，
`execute_agent_acceptance` 展示为“机器人端到端验收”。`technical-solution-design` 完全不变。

Job 用于把需求、实现、验证资产、评测、CI、机器人验收和合码职责显示为独立执行单元。当前通用
Runtime 仍只有一个活动 Step，Job 不承诺原生 DAG 调度；本决策不伪造并行状态。无数据依赖的工作在
对应 Agent Step 内并行：`execute_eval_candidates` 并行 1 至 3 个隔离 Case，GitHub CI 使用 Matrix，
`execute_agent_acceptance` 并行两个全新机器人 Case。要实现多个活动 Job 的 durable 并行状态，必须
另行修改 State/IDL/CLI 并单独审核，不属于本决策。

生产五文件最终包含 45 Conditions、17 Flow Routes、44 Loop Routes 和 56 Prompts。新增 Condition
覆盖验证技能、功能地图、冻结候选、Eval 剧本/候选/裁判、PR、Ruleset 与 CI。Review 通过时冻结
`candidate_head`；其后所有 Stage 只读，任何产品、测试或验证资产变化必须按最早责任层回到需求、
方案、实现、验证技能或功能地图，形成新 HEAD 后重跑本地验证、Review、Eval、CI 和机器人验收。
基础设施故障只报告 blocked，不伪造产品失败。Eval 最多三轮，Rubric 必须 10/10，不能降分放行。

项目验证能力的唯一真值迁入 `.agents/skills/verify-fanloop/`，全部使用中文；根 `FEATURE_MAP.md` 删除，
避免双份地图漂移。Release-bound 的 `fanloop-dev-create-verification` 只在缺失时创建，
`fanloop-dev-maintain-verification` 在每次用户表面变化后维护 Skill 与 Feature 页面。真实配方必须覆盖
Launch、Doctor、Drive、Evidence、Cleanup，使用隔离数据目录和公开入口，Cleanup 后证据仍保留。

Agent Eval 拆为协调者、候选和不同模型裁判。协调者冻结 1 至 3 个 Case、随机目录、硬红线和 10 分
Rubric；候选在隔离目录并行执行，互不共享中间结果；裁判只读原始证据。评测失败必须选择唯一最早
责任回流，不在冻结候选上 Hill Climbing。

Review 后由 `publish_candidate` 发布唯一 PR。`verify_ci_gates` 要求 main Ruleset 使用 required PR、
strict、squash、linear history，禁止 force push、删除和 bypass，人工审批数为 0；required checks
必须在精确 `candidate_head` 上覆盖 Workflow 契约与 Route Matrix、生成物/IDL 新鲜度、run-unit、
run-e2e、隔离安装/Doctor、机器人身份隔离、本地材料禁入库和候选只读。该远端 CI Stage 补充而不
替代 ADR-0063 的本地验证。

机器人验收使用固定 driver `cli_aafadbc67e799cdc`、target `cli_a9245f0fddf8dbc8` 和群
`oc_d532c3a5eda84c60728ab174b0ef671a`，通过外层 Botmux 并行派发两个全新 Case。内层 Fanloop CLI
必须清除 `BOTMUX_CHAT_ID`、`BOTMUX_SESSION_ID`，不得使用用户 token/身份，也不得创建 Card Binding、
Trace Integration、远端 Trace Event、用户 Trace 文档或用户 CLI 日志文档。任一副作用为
`governance_failed`；基础设施阻塞不得回退用户身份。本条取代 ADR-0090 要求内层创建并同步这些远端
资源的相反约束，保留产品自身在明确授权场景下的 Trace/Card 能力。

所有门禁通过后，`merge_code` 对唯一 PR 执行：

~~~bash
gh pr merge "$PR_URL" --repo zeefan1555/fanloop --auto --squash --match-head-commit "$CANDIDATE_HEAD"
~~~

禁止 `--admin`、直接 push main、merge commit、rebase、第二个 PR、MR 交接和人工端到端验收。只有
回读同一 PR 为 `MERGED` 且 merge commit 非空才 terminal；启用 auto-merge 但仍等待 checks 不是完成。

不修改 Thrift IDL、Workflow Schema、通用 Runtime、State/Event/Output Storage、公开 CLI 或两个公开
测试入口。本变更按 e2e 档验证，要求聚焦生产 Bundle/推进 Contract、`./tests/run-unit`、
`./tests/run-e2e`、隔离安装/Doctor 与真实公开 CLI 证据覆盖最终 HEAD。

人工审核记录：用户在收到 5 Stage / 16 Step、45 Conditions、17 条正常 Route、约 40 条显式回流、
Pstack Create/Maintain、三角色 Eval、Dune CI、两机器人身份隔离和 GitHub 自动合码设计后明确同意，
并要求不启动自迭代流程，直接改代码提 PR。实现期间用户补充要求把可独立任务提升为 Job 并识别并行
机会；本 ADR 按当前 Runtime 能力采用 12 Job 展示与三个 Step 内并行批次，不扩展 IDL/Runtime。
