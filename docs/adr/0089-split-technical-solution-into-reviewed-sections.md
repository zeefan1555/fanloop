---
status: accepted
date: 2026-08-30
supersedes_in_part: ADR-0079, ADR-0087
---

# 将技术方案拆成分段产出、三级人工审核和九段式成文

`technical-solution-design` 仍使用三阶段五文件 Bundle，但由七 Step 改为十三 Step：

1. 问题定义：Agent `frame_requirement_background`、Agent `analyze_core_problem`、Agent
   `define_design_objectives`、Human `confirm_technical_problem`；
2. 方案设计：Agent `research_solution_options`、Agent `design_overall_solution`、Agent
   `design_key_solutions`、Human `confirm_solution_direction`；
3. 方案成文：Agent `evaluate_solution_benefits`、Agent `plan_solution_delivery`、Agent
   `write_technical_solution`、Agent `review_technical_solution`、Human
   `confirm_technical_solution`。

每个内容 Step 只负责一个正式章节及其内部调研清单。八个章节片段依次写入
`.technical-solution/sections/01-background.md` 至 `08-delivery.md`；总体方案同时产出
`.technical-solution/architecture.mmd`。写作 Step 生成 `09-appendix.md` 并组装
`technical-solution.md`，不重新推导上游结论。

最终正文除项目标题外只允许九个按序出现的二级 Markdown 标题：需求背景、核心问题、设计目标、
方案调研、总体方案、难点解法、方案收益、落地规划、附录。正文不得出现三级标题或 `1.1` 一类
可见子结构；原二级内容要求由对应原子 Skill 调研，并在片段中使用结论句、表格、列表、图和证据
表达。这样内部过程保留细粒度，评审呈现只有九个一级结构。

三个 Human Step 均为强制人工门禁，并分别发布稳定标题的飞书文档：`<项目>｜问题定义`、
`<项目>｜方案设计`、`<项目>｜技术方案`。发布前按标题精确查找，唯一命中更新、零命中创建、
多命中阻塞；发布后必须通过返回 URL 回读并验证章节完整。只有文档 URL、最新 Panorama 与人的
全新明确批准同时存在才可推进。`technical-solution-design` 删除 `agent_approved` Condition 和
三条 Agent 批准 Route；ADR-0087 的 Agent 批准能力继续只适用于 `fanloop-maintainer`。

所有审校或人工修改意见只提交最早受影响的一项：`background_changed`、`problem_changed`、
`objectives_changed`、`research_changed`、`overall_solution_changed`、
`key_solutions_changed`、`benefits_changed`、`delivery_changed` 或
`presentation_changed`。对应回流目标依次为八个内容 Step 和写作 Step，Runtime 继续按目标 Step
失效其全部下游 Outputs。Evidence 必须保存反馈原文、最早受影响层、保留内容、失效产物与回流
Step；不新增领域 State 或专用 Runtime 分支。

每个成功标记、人工批准、审校通过和反馈层级都属于同一个 `technical_decision_outcome` 互斥组；
文档 URL、Panorama、架构图和审校报告只作为配套事实。一个 Result 因而不能同时表达“成功推进”
与“需要回流”，也不能同时选择多个反馈层级。

Step 契约相对 ADR-0079 的精确变化如下：

- 删除 `frame_technical_problem`、`derive_technical_solution`；
- 新增 `frame_requirement_background`、`analyze_core_problem`、`define_design_objectives`、
  `research_solution_options`、`design_overall_solution`、`design_key_solutions`、
  `evaluate_solution_benefits`、`plan_solution_delivery`；
- 保留五个 Step ID，但依次改名为：`confirm_technical_problem`“问题审核”、
  `confirm_solution_direction`“方案审核”、`write_technical_solution`“方案成文”、
  `review_technical_solution`“方案审校”、`confirm_technical_solution`“方案终审”；
- 全部 Step 按本 ADR 第一段顺序重排；五个保留 Step 的 executor 不变，新内容 Step 均为
  `agent`，三个审核 Step 均为 `human`；Solution Stage 名称由“方案推导”改为“方案设计”。

删除已废弃的 `technical-problem-framing` 与 `technical-solution-derivation`，不保留别名或兼容层。
新增八个与内容 Step 一一对应的 Skill；保留并改写三个审批 Skill、写作、审校与 Panorama Skill。
本变更不修改 Workflow YAML Schema、Thrift IDL、Runtime、State/Event、Storage 或公开 CLI。

该 Step、Route 与 Condition 语义变化使用 `e2e` 验证档：先验证生产 Bundle 拓扑、Skill 绑定、
强制人工门禁、反馈回流与 Output 失效，再从同一最终工作树运行 `./tests/run-unit` 和
`./tests/run-e2e`。

人工审核记录：2026-08-30，用户先确认最终只呈现九个一级结构、原二级要求分散到各 Skill、
每阶段产出飞书文档人工审核、每次反馈重新评估整体影响；随后在收到上述十三 Step、三个人工门禁、
九层回流和无 IDL/Runtime 变化的方案后明确回复“同意， 去帮我改”。
