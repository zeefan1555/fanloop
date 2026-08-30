---
status: accepted
date: 2026-08-30
amends: ADR-0063, ADR-0066, ADR-0076, ADR-0087
---

# 为维护流程增加 Agent 与人类端到端验收

fanloop-maintainer 从 3 Stage / 8 Step 扩展为 3 Stage / 10 Step：

~~~text
需求定义：工作区准备 → 需求澄清 → 需求确认
需求实现：方案设计 → 代码实现 → 本地验证 → 代码审查
变更交付：Agent 自动化验收 → 人类端到端验收 → MR 交接
~~~

现有八个 Step 不删除、不改名、不改变扁平顺序或 executor。review_code 只从 Delivery 移入
Implementation；新增 execute_agent_acceptance(agent) 与 confirm_human_acceptance(human)。

现有 30 个 Condition 全部保留，新增 technical_solution_changes_requested、
agent_acceptance_passed、agent_acceptance_failed、acceptance_report_written、
human_acceptance_passed、human_acceptance_skipped、human_acceptance_result_recorded，合计 37。
Flow Route 从 9 条变为 12 条，Loop Route 从 13 条变为 20 条，Prompt 从 35 个变为 43 个。
Review 通过后进入 Agent 验收；Agent 通过后进入人类验收；人明确通过或跳过后才进入 MR 交接。
Review、Agent 验收和人类验收分别可按已声明原因回到需求、方案或实现，目标 Step 及其下游 Output
继续由通用 Runtime 失效。

ADR-0063 的本地验证档仍保留：execute_test_cases 只负责 review 前 targeted 或 e2e 的确定性本地
验证和 local-test-report.md。本次改变 Workflow 推进语义，固定使用 e2e，并从同一工作树运行聚焦
Contract、./tests/run-unit 和 ./tests/run-e2e。候选安装、Feature Map 维护和真实机器人黑盒不再混入
该 Step，而在 Review 通过后执行。

新增两个 Release-bound Skill：

- fanloop-dev-maintain-verification 以根 FEATURE_MAP.md 的每一行为 Feature 单位，串行执行 source +
  live 校准，只能修改验证资产，结果为 clean、changed 或 blocked。
- fanloop-dev-agent-acceptance 只验收 local-test-report.md、review-report.md 与当前 reviewed HEAD
  一致且维护 clean 的候选；从同一工作树运行 npm run install:local，回读 version commit 与 doctor，
  再由“使用 Fanloop 机器人” app cli_aafadbc67e799cdc 在固定群
  oc_d532c3a5eda84c60728ab174b0ef671a 真实驱动“FanLoop 机器人”
  app cli_a9245f0fddf8dbc8 的全新话题和全新 Requirement。

Q8=B：继续使用正式 current slot 的 npm run install:local，不引入 Treeloop managed Dev
install/uninstall、自动回滚或兼容层。Q10=A：保留 local-test-report.md 与 review-report.md，只新增
acceptance-report.md；不创建实现或验收飞书文档。外部每日调度、Cloud Agent 和每 Feature 子 Agent
不在本次范围。

ADR-0076 的 Test Seam / Tracer Bullet 规则继续成立。Pstack Create 的 repo interview 与
Launch/Doctor/Drive/Evidence/Cleanup，以及 Maintain 的 feature-unit source + live、只修验证资产、
clean/changed/blocked，被吸收到现有 Feature Map 与两个新 Skill；不复制 Cursor slash command、
平行 Eval 框架或无边界 Hill Climbing。

本决策只收窄 ADR-0087 的适用范围：既有 confirm_requirements 继续保留 agent_approved Route；
新 confirm_human_acceptance 明确没有该 Route。机器人、当前 Agent、旧消息或含糊回复不能代替固定
人工审批人的全新通过、跳过或反馈结论。ADR-0086 的单源 Panorama 继续作为人工 Route 的前置事实。
ADR-0071 的 MR 人工审核、不得自动合并或发布保持不变，ADR-0088 的 main 更新后发布保持不变。

不修改 Thrift IDL、生成物、通用 Workflow loader/runtime、State/Event/Output Schema、公开 CLI
Request/Response 或两个公开测试入口。五份生产 YAML 是全部新增推进语义的唯一真值。

人工审核记录：2026-08-30，用户先确认 Q8=B、Q10=A 和 Pstack 融合；随后收到 3/4/3 十步
workflow、30→37 Conditions、9→12 Flow Routes、13→20 Loop Routes、35→43 Prompts、
新增 SkillBinding、Human Step 无 Agent 代批及 e2e 影响说明后，明确回复
“批准，然后现在端到端测试机器人也可用了”。机器人“可用”仅是环境事实，最终候选仍必须实时校验
driver/target/chat/Release/网络/接单与可读回执。
