---
name: fanloop-dev-eval-coordinator
description: 为冻结候选生成防作弊 Eval Playbook、10 分 Rubric 和隔离执行计划。
---

# 评测协调

只读取 requirements.md、验证技能、功能地图、报告和 candidate_head，不得修改候选。

1. 选择恰好两个覆盖最高风险用户行为的黑盒 Case；输入不透露评测目的。
2. 为每个 Case 定义可观察通过条件、硬红线和合计 10 分的 Rubric；只有 10/10 可通过。
3. 生成随机且无语义的隔离目录名，固定候选 SHA、Release、超时、证据路径和候选模型。
4. 指定裁判使用不同模型；候选之间禁止共享上下文和中间结果。

把全部内容写入并回读 eval-playbook.md。缺少独立正确预期时保持 blocked，不编造评分标准。
