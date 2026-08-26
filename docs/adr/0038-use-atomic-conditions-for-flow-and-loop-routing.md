---
status: accepted
date: 2026-08-15
amended_by: ADR-0043
---

# 使用原子 Condition 驱动 Flow 与 Loop 路由

Fanloop 的下一版 Workflow Bundle 继续由五份不可拆分、不可变的 YAML 组成，但以 `condition.yaml` 取代 `guard.yaml`。`workflow.yaml` 继续只注册 Stage、Job、Step 与 executor；`condition.yaml` 定义可复用的原子 Condition、给 Agent 的 PromptRef、要求上报的 OutputSpec 及可选互斥组；`flow.yaml` 与 `loop.yaml` 都使用 `map[StepID][]Route` 和相同的 `when.any_of` 条件组合，分别以 `next_step_id`/`terminal` 与 `back_step_id` 表达正常方向和回流方向；`prompt.yaml` 继续保存 Prompt 与逐项 Skill 使用指导。Bundle digest 覆盖这五份文件的规范化语义。

Condition 是 Agent 可以上报的原子事实，不是 CLI 内置的业务判断器。Agent 通过 ConditionResult 提交 `condition_id` 和符合该 Condition OutputSpec 的值，并可附带 Evidence 与 Summary。CLI 只校验 Thrift 结构、当前 Step、Condition 引用、Output key/type/constraints、互斥组和 Route 唯一性；CLI 不执行单测、代码评审或人工审批，不证明 Evidence 的内容质量，也不把自然语言判断隐藏在运行时代码中。

Route 的 `when.any_of` 外层为 OR，内层 ConditionID 列表为 AND。Agent 每次 Result 都必须同时提交 ConditionResults 和一个 RouteSelection；RouteSelection 恰好选择 `next_step_id`、`back_step_id` 或 `terminal` 之一。CLI 只在与该选择同方向、同目标的当前 Step Route 中匹配 `when.any_of`，并要求恰好一条 Route 命中。事实与目标不一致、零命中或多命中均拒绝本次 Report，不提交 State 或 Event。发布前校验继续拒绝未知引用、空条件组、非法方向以及可静态识别的歧义。

`flow status` 返回当前 Stage/Job/Step 上下文、执行事实、解析后的 Prompt、当前相关 Condition、带方向的可选 Route 和可用 Output。`flow report progress` 继续只提交非终态执行事实；现有 Output 与 Loop 写入口由一个 ConditionResult Report 取代。成功响应只返回调用方需要的 `effect`、可选 `event_id`、`transition`、回流失效的 Output key 和最新 `state`，不回显内部 matched/accepted 计算过程。dry-run 不生成 EventID。

State 继续是当前事实的唯一运行时来源，Event 继续是不可变审计历史。Agent 上报的 OutputValue 只有 type/value；CLI 写入 State 的 RegisteredOutput 额外保存 `producer_step_id`，使 Condition 被多个 Step 复用时仍可按回流目标失效正确的下游 Output。Event 保存 ConditionResult、Evidence、Summary、Effect、Transition 和 Output 变化；Trace 与 Panorama Card 只投影已提交的 State/Event，不参与路由。远端投影仍在本地事实提交后 best-effort 执行，失败不得回滚本地推进。

本决策直接切换 Workflow Release 7.0.0、Flow Schema 4、Condition Schema 1、Loop Schema 4、State/Event Schema 8；`workflow.yaml` 与 `prompt.yaml` 的文档结构不变，分别继续使用 Schema 6 与 Schema 1。旧 Guard 模型、GuardResult、Flow Output/Loop Report、旧 Workflow Bundle 和旧 State/Event decoder 不保留 alias、adapter、双读写、Feature Flag 或迁移工具。运行中的旧 Requirement 继续由其绑定的旧 Release 解释。

本决策取代 ADR-0036 的 Guard、GuardResult、两阶段 Automatic/Manual Loop 与 Guard-driven Status 部分，并取代 ADR-0037 的 Artifact-only Output、独立 Loop Report 及对应 Flow IDL 部分。ADR-0034 的存储职责、ADR-0035 经 ADR-0037 修订后的 Panorama 触发、ADR-0014 的当前事实边界、ADR-0015 的 durable Event 边界和 ADR-0016 的本地事实优先仍继续有效。

真实生产 Workflow 五份 YAML 的完整变更前后版本、ConditionID 清单、Route 条件组合与影响说明已于 2026-08-15 完成人工审核；确认记录见 `docs/specs/workflow-condition-routing-review.md`。

2026-08-16 修订：Implement 阶段不再保留独立 `review_code` Step。`run_automated_checks` 一次收集单测与 Aime Code Review 两组二态事实；单测 passed/failed 均可继续，但只有 `code_review_passed` 才能前进到 `confirm_ready_for_test`，`code_review_failed` 必须回到 `implement_code`。Aime 的任务链接、Verdict 和评论数只作为 Evidence，不再作为 Route Condition 或必填 Output。API 测试和其他 Blocking Check 不再参与该 Step 的 Route。该修订只收敛生产 Workflow 配置，不改变 ConditionResult、OR-of-AND、State/Event 或 Output 失效契约。

2026-08-16 修订：Agent 不再只为 Loop 选择 `back_step_id`，而是在每次 Result 中显式提交 RouteSelection。CLI 不接受任意目标，也不替 Agent 判断业务事实；它验证所报 ConditionResults 是否唯一允许该 Route，然后原子提交 Effect、Transition、State/Event 和 Output 失效。本修订直接删除旧的可选顶层 `back_step_id` Request 字段，不保留兼容入口；五份生产 YAML、OR-of-AND、State/Event durable 事实和 Output 失效算法不变。
