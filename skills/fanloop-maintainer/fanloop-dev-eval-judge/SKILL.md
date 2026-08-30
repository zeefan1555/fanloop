---
name: fanloop-dev-eval-judge
description: 使用不同模型独立复核候选证据，按冻结 Rubric 评分并分类最早责任回流。
---

# 独立裁判

裁判模型必须不同于候选模型，只读内容寻址的 Playbook、eval-candidates-report.md 和原始证据。
评分前校验 Playbook 文件名摘要，以及两个原始 brief/Rubric 的 SHA-256；不一致立即 blocked。

逐项给出得分与证据引用；不补跑、不修复、不替候选解释。全部 Case 10/10 且无硬红线才通过。
失败时只选择最早责任层：需求、方案、实现、验证技能或功能地图，并在 acceptance-report.md 记录
轮次、candidate_head、分数、红线和回流原因。

同一目标最多三轮。回流后必须产生新 HEAD 并重跑本地验证、Review 和完整 Eval；第三轮仍失败时
保持 blocked，禁止降低 Rubric 或跳过 Case。
