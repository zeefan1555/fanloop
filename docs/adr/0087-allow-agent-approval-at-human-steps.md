---
status: accepted
date: 2026-08-30
amends: ADR-0066, ADR-0072, ADR-0076, ADR-0079, ADR-0081
amended_by: ADR-0090
---

# 允许 Agent 在 Human Step 独立批准

`fanloop-maintainer` 与 `technical-solution-design` 的每个 Human Step 都允许当前 Agent
在独立复核通过后直接推进。两套 Workflow 分别声明同一个原子 Condition：

```yaml
agent_approved:
  prompt_ref: {file: prompt.yaml, prompt_id: agent_approval_condition}
  output:
    key: agent_approval_decision
    type: enum_value
    values: [approved]
    description: 当前 Agent 已独立复核并同意继续
```

生产 Route 的人工批准基线与批准后的 Agent OR 组如下：

```yaml
# technical-solution-design，变更前
when: {any_of: [[panorama_card_published, technical_problem_approved]]}

# technical-solution-design，变更后；其余两个 Human Step 分别替换领域 approved Condition
when:
  any_of:
    - [panorama_card_published, technical_problem_approved]
    - [agent_approved]
```

```yaml
# fanloop-maintainer，变更前
when: {any_of: [[panorama_card_published, requirements_approved, requirements_approval_recorded, requirements_evidence_written, implementation_required]]}

# fanloop-maintainer，变更后；terminal Route 对应使用 implementation_not_required
when:
  any_of:
    - [panorama_card_published, requirements_approved, requirements_approval_recorded, requirements_evidence_written, implementation_required]
    - [agent_approved, implementation_required]
```

`technical-solution-design` 的三个 Human Step 在原人工批准 AND 组之外，各增加
`[agent_approved]` OR 组。`fanloop-maintainer.confirm_requirements` 的实现与无需实现两条
成功 Route 分别增加 `[agent_approved, implementation_required]` 与
`[agent_approved, implementation_not_required]` OR 组。Agent 路径不要求
`panorama_card_published`、人工批准消息或人工证据文件；ConditionResult 的 Evidence 记录复核
理由与产物引用。Agent 不能确认无阻塞项时不提交该 Condition，继续使用原人工审核、拒绝和回流
路径。

原人工路径保持不变：只要选择人工决定，Panorama 仍是批准、拒绝与反馈分类 Route 的前置事实，
原审批人、消息、回执和 Evidence 约束继续生效。MR 交接只发起后续人工审核而不等待决定，不属于
本次范围。

两套 `workflow.yaml` 与 `loop.yaml` 不变。`fanloop-maintainer` 的八个 Step 和
`technical-solution-design` 的七个 Step 均无新增、删除、改名、重排或 executor 变化；所有 Route
目标与回流失效边界不变。实现只同步 `condition.yaml`、`flow.yaml`、`prompt.yaml`、Release-bound
Skills 与契约测试，不修改 Workflow YAML Schema、Thrift IDL、Runtime、State/Event、Storage 或
公开 CLI。当前 Bundle digest 随内容改变；已绑定旧 Release 的 Requirement 继续使用原人工门禁，
不增加兼容层或迁移器。

该 Route/Condition 推进语义变化使用 `e2e` 验证档：聚焦验证生产 Bundle 和 Agent 审批路径，
并从同一最终工作树运行 `./tests/run-unit` 与 `./tests/run-e2e`。

人工审核记录：2026-08-30，用户在当前任务中先收到上述 Condition、两类 Route 前后示例、
四个 Human Step 范围、Step/executor 不变结论及 `e2e` 验证计划，随后明确回复
“同意提一个pr 去改”。该批准不覆盖额外 Step、Loop、Runtime 或 IDL 变化。
