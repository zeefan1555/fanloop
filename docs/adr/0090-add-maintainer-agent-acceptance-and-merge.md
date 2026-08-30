---
status: accepted
date: 2026-08-31
amends: ADR-0063, ADR-0066, ADR-0072, ADR-0076, ADR-0077, ADR-0087
supersedes: ADR-0071
---

# 为维护流程增加 Agent 自动化验收与合码

`fanloop-maintainer` 从 3 Stage / 8 Step 直接切换为 3 Stage / 9 Step：

~~~text
需求定义：工作区准备 → 需求澄清 → 需求确认
需求实现：方案设计 → 代码实现 → 本地验证 → 代码审查
变更交付：Agent 自动化验收 → 合码
~~~

相对原八步生产拓扑，`review_code` 只从 Delivery 移入 Implementation，扁平位置和 executor 不变；
删除 `handoff_merge_request(agent)`，新增 `execute_agent_acceptance(agent)` 与 `merge_code(agent)`。
不存在新增、改名、重排或 executor 变化的其他 Step。`confirm_requirements` 是唯一 Human Step；
Agent 验收后不再设置人类端到端验收。

原 30 个 Condition 删除 `merge_request_created`、`merge_request_handed_off`、
`merge_request_handoff_failed`、`handoff_record_written`，新增
`technical_solution_changes_requested`、`agent_acceptance_passed`、`agent_acceptance_failed`、
`acceptance_report_written`、`code_merged`、`code_merge_failed`，最终为 32 个。Flow Route 为 10 条，
Loop Route 为 18 条，Prompt 为 38 个。Review 通过后进入 Agent 验收；验收通过后进入合码；
合码回读成功才 terminal。需求、方案、实现或合码失败仍由五份 YAML 的显式 Loop Route 回流，
目标 Step 及其下游 Output 由通用 Runtime 失效。

ADR-0063 的本地验证档继续成立：`execute_test_cases` 只负责 Review 前的 targeted 或 e2e 确定性
本地验证和 `local-test-report.md`。本次改变 Workflow 推进语义，固定使用 e2e，并从同一工作树
运行聚焦 Contract、`./tests/run-unit` 与 `./tests/run-e2e`。远端 PR checks 不作为 Workflow
验证事实或 Route 条件。

`execute_agent_acceptance` 先按根 `FEATURE_MAP.md` 逐 Feature 执行 source + live 维护，只允许修验证
资产；随后从同一 reviewed HEAD 运行 `npm run install:local`，回读 `fanloop version` commit 与
Doctor，再由“使用 Fanloop 机器人”真实驱动“FanLoop 机器人”执行全新黑盒 Case。

真实机器人黑盒保留外层 Botmux 会话，Candidate 的内层 Fanloop CLI 必须捕获本轮 Card Binding，
自动创建并同步 Trace 文档、CLI 日志文档和 Registry。所有 Lark 子命令只使用 bot identity；bot
scope 不足是 `infra_blocked`，不得回退用户身份。Judge 必须回读 Card Binding、Trace Integration、
`trace_document_bound`、`trace_sync_started`、成功的 `trace_synced` 与三个远端 target。Candidate
到达 `merge_code` 边界即停止，不得在黑盒 Case 内执行合码或发布。

新增 `fanloop-dev-merge-code`，并删除 `fanloop-dev-mr-handoff`。合码只接收三个报告与当前干净
reviewed HEAD 精确一致的候选：回读最新 `origin/main`，要求候选零 behind；幂等推送并创建或更新
该分支唯一的 GitHub PR；以 `--squash --match-head-commit <reviewed HEAD>` 合并；最后回读 PR 的
base、head、head OID、`MERGED` 状态和非空 merge commit。禁止 `--admin`、`--auto`、merge commit、
rebase、直接 push main、Botmux MR 话题、`handoff.json`、人工验收与其他仓库或身份 fallback。

本仓库没有 branch protection 且 GitHub auto-merge 关闭。合码门禁因此只使用当前 HEAD 已完成的
本地验证、代码审查与 Agent 自动化验收，不读取或等待远端 checks；`main` push 继续按 ADR-0088
异步触发 Release 的 `run-unit`、`run-e2e`、构建和发布，合码 Step 不等待或宣称发布成功。

Pstack Create 的 Surface / Run / Drive / Observe / Isolate 与 Launch / Doctor / Drive / Evidence /
Cleanup，以及 Maintain 的 feature-unit source + live、只修验证资产、clean/changed/blocked，被吸收
到现有 Feature Map 与验收 Skills；不复制 Cursor slash command、平行 Eval 框架或无边界 Hill
Climbing。Q8=B 继续使用正式 current slot 的 `npm run install:local`，不引入 Treeloop managed Dev
安装、回滚或兼容层。Q10=A 继续保留 `local-test-report.md`、`review-report.md` 与唯一新增的
`acceptance-report.md`，不创建实现或验收飞书文档。

ADR-0087 的 Agent 批准能力只继续适用于 `confirm_requirements`；技术方案 Workflow 的既有 Human
Step 不变。ADR-0072 的同一 Botmux Session 复用唯一 Requirement Root 保留，但其中 MR 交接边界陈述
被本决策取代。ADR-0071 的 Botmux MR 交接被完整取代，不保留 alias、兼容 Route、旧 Skill 或迁移。

不修改 Thrift IDL、生成物、Workflow loader、State/Event/Output Schema、公开 CLI
Request/Response 或两个公开测试入口。五份生产 YAML 是 Agent 验收、合码与回流语义的唯一真值；
Runtime 只保留通用解释和既有 bot-only Trace/Registry 执行接缝。生产 Registry 继续同步需求、技术方案、
Card Binding、Trace 文档与 CLI 日志；由于 Workflow 不再产生 `merge_request_urls` Output，删除该失效字段映射，
PR URL 与 merge commit 改由 `acceptance-report.md` 记录和回读。

人工审核记录：用户先于 2026-08-30 确认 Q8=B、Q10=A、Pstack 融合和 bot-only 完整 Trace 投影；随后
明确指出 MR 交接是 Treeloop 逻辑、Fanloop 不应复用，并在收到删除 Human/MR 节点、增加
`merge_code(agent)`、3/4/2 九步拓扑、32 Conditions、10 Flow Routes、18 Loop Routes、38 Prompts、
GitHub squash 合码及 e2e 影响说明后，于 2026-08-31 回复“同意，然后帮我合码”。
