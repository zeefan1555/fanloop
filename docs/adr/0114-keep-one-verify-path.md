---
status: accepted
date: 2026-09-22
amends: ADR-0112, ADR-0113
supersedes_in_part: ADR-0091, ADR-0101
amended_by: ADR-0115, ADR-0118
---

# 园丁循环只留一条验证路径

Cut 1 与 Cut 2 已经要求验收枚举匹配 verify manifest。本决策删掉会把 Agent 带向第二条验证入口的生产提示，并用测试挡住同类条件再长出来。不改 Step `id`、`name`、顺序或 `executor`。不改移植的 pstack `SKILL.md` 正文。不改 Thrift。

## 一条路径

Cursor 入口是 `.cursor/skills/verify-fanloop/SKILL.md`。维护入口是 `workflows/fanloop-maintainer/skills/fanloop-dev-verify/SKILL.md`。Feature Map 只在后者的 `references/features/`。Cursor 的 `features` 是指向该目录的符号链接。

`pstack-delivery` 的 `verify_report_flow` 不再绑定 `create-verification-skill` 或 `maintain-verification-skill`。`principle-prove-it-works` 仍是必选，并仍指向 Cursor 入口。`verification_outcome_condition` 改为只接受 `fanloop verify smoke` 或 `fanloop verify snapshot` 写出的 schema-1 `manifest.json`。

## 静态检查

生产 Condition 在同时满足下面三条时，Output source 必须是 `verify.evidence.outcome`。

- 类型是 `enum_value`
- `values` 含有 `passed`、`failed` 或 `blocked`
- Condition id 或 Output key 按 `_` 切开后，含有 `verification`、`verified`、`smoke` 或 `blackbox`

允许例外只有 `main_agent_acceptance_passed` 与 `merge_tree_verified`。二者保持没有该 source。`main_agent_acceptance_passed` 是主 Agent 决定。`merge_tree_verified` 是 merge commit tree 比对，不是这次 verify 运行的 outcome。

`workflows/fanloop-maintainer/skills/` 的目录集合冻结为当前 16 个 `fanloop-dev-*` skill。新增一个目录必须在同一次变更里删除另一个，并更新 `TestMaintainerSkillCatalogIsFrozen`。

## YAML

`workflows/pstack-delivery/workflow.yaml` 的 Step 对比如下，四项都不变。顺序仍是 intake、select_bind_skills、confirm_skill_plan、execute、verify_report。

| id | name | executor |
| --- | --- | --- |
| intake | 交付意图与约束 | agent |
| select_bind_skills | 选择并绑定原生 pstack skills | agent |
| confirm_skill_plan | 确认技能绑定计划 | human |
| execute | 按绑定执行原生 skills | agent |
| verify_report | 验证与交付报告 | agent |

`prompt.yaml` 的 `verify_report_flow.skills` 从三条变为一条。删除的两条是可选的 `create-verification-skill` 与 `maintain-verification-skill`。留下的是必选 `principle-prove-it-works`。

ADR-0118 取代上表：该套已改名为 `pstack-native`，Step 四元组以 ADR-0118 的新表为准。`delivery_verified` 仍要求 verify manifest。

`verification_outcome_condition` 变更前是「仅在独立核验无阻塞项且交付报告已回读时上报 passed。」变更后要求 Evidence 指向本次 verify 写出的绝对路径 `manifest.json`，并且 `outcome` 为 `passed`。

## 取代

ADR-0091 中「项目验证能力的唯一真值迁入 `.agents/skills/verify-fanloop/`」不再有效。该路径保持删除。

ADR-0101 中「验证入口位于 `skills/fanloop-maintainer/fanloop-dev-verify/`」不再有效。现行路径是 `workflows/fanloop-maintainer/skills/fanloop-dev-verify/`。

ADR-0112 与 ADR-0113 的 manifest 门禁、Skill 禁手写 manifest、以及不改 Step 拓扑仍然有效。

本决策的生产配置存储/发布、Dashboard编辑或代码验证地图位置条款由 [ADR-0120](0120-host-production-workflows-in-fornax.md) 按其明确范围修订；其余门禁继续有效。
