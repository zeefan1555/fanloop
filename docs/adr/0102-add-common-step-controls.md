---
status: accepted
date: 2026-09-17
amends: ADR-0038, ADR-0052, ADR-0080, ADR-0098, ADR-0099
---

# 用通用控制让每个技术方案 Step 先确认再开工

Workflow Bundle 增加可选的 Workflow 级 `common_skills`、`common_conditions`、`step_start` 和
`jump`。未声明这些字段的 Workflow 保持原有行为；声明后，每次进入 Step 都先处于
`awaiting_confirmation`，Status 只暴露开工确认 Prompt、一个 start Route 和面向全部真实 Step 的
jump Routes。确认后以独立 `started` Event 进入同一 Step 的 `in_progress`，再暴露原有业务 Prompt、
Condition 和 Route。Jump 在等待或执行状态均可使用。

Start 和 Jump 都只接受各自唯一的 common Condition，并要求 `source=human` Evidence。Jump 的 Condition
输出必须等于 `jump_step_id`。Common ConditionResult 只进入 Event，不进入 Output Registry；普通业务
Result 不能与 common Condition 混报。Runtime 继续只解释 YAML 声明，不硬编码 Workflow、Step、
Condition 或 Skill ID。

Jump 总是失效目标 Step 及下游 Outputs。向前跳把源 Step 到目标前一 Step 记录到 durable
`skipped_step_ids`；跳到当前或更早 Step 时清除目标及下游 skipped 标记。Card 与 Trace 将 skipped
Step 显示为“已跳过”，不显示完成标记。完成状态仍保留 skipped 列表。

`technical-solution-design` 在不改变 16 个 Step 的 ID、名称、顺序和 executor 的前提下，绑定 required
`grill-with-docs` 与 optional `human-step-jump`。前者在每个 Step 开工前基于已有文档澄清目标、边界、
输入、输出和验收条件，并等待人的明确确认；后者在跳转前展示跳过、失效和缺失依赖影响并确认目标。

本决策升级 Prompt/Condition/Flow YAML Schema 到 2/3/5，Flow State/Event Schema 到 13，Card
Projection Schema 到 6，并扩展公开 Flow Thrift 的状态、RouteSelection、Effect、Transition 和 Status
投影。旧 Schema 不提供兼容读取、迁移或 fallback；已运行 Requirement 继续由其绑定 Release 解释。

人工审核记录：2026-09-17，用户批准完整 IDL/YAML 前后契约，并明确授权自主完成实现、验证、PR 与本地安装。
