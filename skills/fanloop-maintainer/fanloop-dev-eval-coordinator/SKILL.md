---
name: fanloop-dev-eval-coordinator
description: 为冻结候选生成防作弊 Eval Playbook、10 分 Rubric 和隔离执行计划。
---

# 评测协调

只读取 requirements.md、验证技能、功能地图、报告和 candidate_head，不得修改候选。

1. 选择恰好两个覆盖最高风险用户行为的黑盒 Case；每个原始 brief 单独写入文件，输入不透露评测目的。
2. 每个原始 brief 必须内含全新目录、全新 Requirement、公开 CLI、证据字段、内层 Botmux 环境隔离和
   merge_code 前停止的固定契约；为它另存合计 10 分的 Rubric 与硬红线，只有 10/10 可通过。
3. 为两个 brief 和两个 Rubric 计算 SHA-256。`eval-playbook.md` 记录 candidate_head、Release、超时、
   随机且无语义的隔离目录名，以及每个 Case 的 case_id、brief_path、brief_sha256、rubric_path、
   rubric_sha256 和候选模型。
4. 对清单计算 SHA-256 并重命名为 `eval-playbook.<sha256>.md`；文件名摘要必须等于文件内容摘要。
5. 指定裁判使用不同模型；候选之间禁止共享上下文和中间结果。

把内容寻址的 Playbook 路径作为 `eval_playbook_path` 上报并回读校验。缺少独立正确预期时保持
blocked，不编造评分标准。
