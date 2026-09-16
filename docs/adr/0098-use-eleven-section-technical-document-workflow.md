---
status: accepted
date: 2026-09-16
supersedes_in_part: ADR-0089, ADR-0093
---

# 技术文档按十一章逐章产出并以读者理解路径组装

`technical-solution-design` 保留三阶段五文件 Bundle 和三个人工门禁，但从十三 Step、九章正文改为
十六 Step、十一章正文：

1. 业务问题：Agent `frame_requirement_background`、Agent `define_goals_and_problems`、Agent
   `define_business_constraints`、Human `confirm_technical_problem`；
2. 技术判断：Agent `research_solution_options`、Agent `design_overall_solution`、Agent
   `design_key_solutions`、Agent `record_technical_decisions`、Human `confirm_solution_direction`；
3. 结果与规划：Agent `plan_solution_delivery`、Agent `evaluate_solution_benefits`、Agent
   `write_retrospective_and_roadmap`、Agent `write_summary`、Agent `write_technical_solution`、Agent
   `review_technical_solution`、Human `confirm_technical_solution`。

十一个文章标题分别对应一个 Agent Step 和一个独立 Markdown 产物：

| 章节 | 产物 |
|---|---|
| 0. 摘要 | `.technical-solution/sections/00-summary.md` |
| 1. 业务背景 | `.technical-solution/sections/01-business-background.md` |
| 2. 目标与问题定义 | `.technical-solution/sections/02-goals-and-problems.md` |
| 3. 业务特点与技术约束 | `.technical-solution/sections/03-business-constraints.md` |
| 4. 业界/业内方案调研 | `.technical-solution/sections/04-research.md` |
| 5. 总体方案设计 | `.technical-solution/sections/05-overall-solution.md` |
| 6. 关键模块设计 | `.technical-solution/sections/06-key-modules.md` |
| 7. 核心技术决策与取舍 | `.technical-solution/sections/07-decisions.md` |
| 8. 落地路径与风险控制 | `.technical-solution/sections/08-delivery-and-risk.md` |
| 9. 结果收益 | `.technical-solution/sections/09-results.md` |
| 10. 复盘与后续规划 | `.technical-solution/sections/10-retrospective-and-roadmap.md` |

执行顺序为第 1 至第 10 章、摘要、组装、审校、终审。摘要依赖完整结果，因此最后生成；最终
`technical-solution.md` 仍按第 0 至第 10 章排列。组装 Step 不再生成附录，也不重新推导章节结论。

同一骨架支持晋升文档、技术分享和融合文档，不增加配置字段或第二套 Workflow。晋升场景强调业务
价值、技术深度、个人 owner 范围、影响力和规划能力；技术分享强调问题抽象、方案复用、关键经验、
适用边界和踩坑；融合场景同时保留两类证据，并始终按评委或听众的理解路径组织而非时间线流水账。

第一次 Human Gate 审核第 1 至第 3 章，第二次审核第 1 至第 7 章和主架构图，最终门禁审核第 0 至
第 10 章。审校和人工反馈只选择最早受影响章节，Runtime 继续按目标 Step 失效下游 Output；只涉及
最终标题、顺序或图文呈现的问题回到组装 Step。三个 Human Gate 均继续要求稳定飞书文档、最新
Panorama 和人的全新明确批准，不恢复 Agent 自动批准路径。

正文固定保留十一个带 `0.` 至 `10.` 前缀的二级标题。允许在所属章节中按真实内容使用动态 `###`
标题，禁止 `####` 及更深标题、手工 `1.1` 式子编号、空模板标题和附录章节。主架构图仍是总体
方案的强制产物，辅助图只在正文明确引用时纳入。

相对 ADR-0089 的精确 Step 变化：

- 删除 `analyze_core_problem`、`define_design_objectives`；
- 新增 `define_goals_and_problems`、`define_business_constraints`、
  `record_technical_decisions`、`write_retrospective_and_roadmap`、`write_summary`；
- 保留其余十一个 Step ID，更新展示名称；
- `plan_solution_delivery` 调整到 `evaluate_solution_benefits` 之前；
- 保留 Step 的 executor 不变，五个新增 Step 均为 `agent`；
- Stage 名称改为“业务问题”“技术判断”“结果与规划”。

删除不再引用的 `technical-problem-analysis` 和 `technical-objective-setting` Skill，新增与五个新章节
Step 对应的 Skill；保留 Skill 按新章节路径和内容契约更新。不保留旧章节、Condition 或 Skill 的
alias、fallback 和迁移层。已绑定旧 Release 的 Requirement 继续由其固定 Release 解释。

本变更不修改 Workflow YAML Schema、Thrift IDL、Runtime、State/Event、Storage 或公开 CLI 结构。
由于改变 Step、Route、Condition、Prompt/SkillBinding 和产物契约，使用 `e2e` 验证档，要求聚焦
Contract 与 Route Matrix、`./tests/run-unit`、`./tests/run-e2e`、冻结候选的隔离公共 CLI 验收及同一
candidate HEAD 的 PR required checks 全部通过。

人工审核记录：2026-09-16，用户收到五份生产 YAML 的目标拓扑、16 Step、11 个独立章节产物、
摘要最后生成但最终置顶、保留三个人工门禁与独立审校、最早章节回流和无 IDL/Runtime 变化的说明后，
明确回复“同意去改吧”。
