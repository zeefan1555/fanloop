---
status: accepted
date: 2026-09-22
amends: ADR-0110
supersedes_in_part: ADR-0112, ADR-0113, ADR-0114, ADR-0115
---

# 第四套生产 Workflow 改为 pstack-native 六步剧本交付

人已批准用六步剧本型交付替换 `pstack-delivery`。本 ADR 修订 ADR-0110 的第四套 id，并取代 ADR-0112、ADR-0113、ADR-0114、ADR-0115 里把该套 Step 写成 `intake` / `select_bind_skills` / `confirm_skill_plan` / `execute` / `verify_report` 且四元组不变的句子。ADR-0117 已经把制卡套换成 `remnote-flashcards`；本 ADR 不改那套 Step。`fanloop-maintainer` 与 `technical-solution-design` 的拓扑也不变。47 个原生 pstack `SKILL.md` 正文不改，随目录移到 `workflows/pstack-native/skills/`。

编号避开 ADR-0117：该号已用于 RemNote 替换 Decks。

## 库存

四套仍是：

- `technical-solution-design`
- `fanloop-maintainer`
- `remnote-flashcards`
- `pstack-native`（原 `pstack-delivery`）

场景键同步改为 `pstack-native`。

## Step 对比

人已批准下列 Step 四元组变化。旧五步全部删除，新六步全部新增，顺序与 executor 都变。`confirm_skill_plan` 的 human executor 不再存在。

旧 `workflows/pstack-delivery/workflow.yaml`：

| id | name | executor |
| --- | --- | --- |
| intake | 交付意图与约束 | agent |
| select_bind_skills | 选择并绑定原生 pstack skills | agent |
| confirm_skill_plan | 确认技能绑定计划 | human |
| execute | 按绑定执行原生 skills | agent |
| verify_report | 验证与交付报告 | agent |

新 `workflows/pstack-native/workflow.yaml`：

| id | name | executor |
| --- | --- | --- |
| select_playbook | 选择工作剧本 | agent |
| define_acceptance | 定义验收标准 | agent |
| design_solution | 方案设计 | agent |
| implement_increments | 增量实现 | agent |
| verify_behavior | 行为验证 | agent |
| merge_and_release | 合并与发布更新 | agent |

## 证据门禁

`delivery_verified`、`post_merge_smoke_passed` 与 `post_merge_smoke_failed` 的 Output source 仍是 `verify.evidence.outcome`。`delivery_verified` 与 `post_merge_smoke_passed` 只接受 `passed`。`post_merge_smoke_failed` 只接受 `failed`。

## 加载器补齐

已批准草案的 `loop.yaml` 没有 `select_playbook`。加载器要求每个 Step 至少一条 Loop，空列表也不通过。因此 `select_playbook` 在 `playbook_wrong` 时用已有 Prompt `back_to_playbook` 回到自身。这不改变 Step `id`、`name`、顺序或 `executor`。

## 未做

不改 `fanloop-maintainer` 的 Stage、Job、Step。不改 `remnote-flashcards` 的 YAML。不改 pstack `SKILL.md` 正文。不把 `fanloop-dev-merge-and-update-local` 跨套绑定进来。
