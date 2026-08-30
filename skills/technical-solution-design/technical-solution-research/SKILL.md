---
name: technical-solution-research
description: 按统一维度调研不改基线、公司内部与业界同类方案，形成公平可追溯的对比片段。用于 technical-solution-design 的方案调研 Step；不直接确定总体架构。
---

# 调研候选方案

只以已批准的 `01-background.md`、`02-problem.md`、`03-objectives.md` 为边界。先从问题、指标和约束
导出同一组评价维度，再调查：

1. 不改基线及小幅演进方案；
2. 公司内真实可复用的同类方案；
3. 有一手资料支持的业界成熟方案；
4. 当前候选方案相对上述方案的优势、缺点、适用条件和失败边界。

候选必须势均力敌，全部使用相同维度比较，不拿完整方案与残缺样例对比。来源优先使用代码、正式
文档、论文或产品官方资料；无法验证的比较项写“未知”。把延迟、可用性、一致性、成本、复杂度、
复用性和交付风险中与本问题相关的维度量化，避免只列功能。

将评价维度、证据链接、同维度对比表和可行候选写入并回读
`.technical-solution/sections/04-research.md`。片段不得出现 Markdown 标题；可以给出候选排序及依据，
但不得提前画目标架构或展开难点解法。

若调研改变背景、问题或目标，只选择最早层级上报 `background_changed`、`problem_changed` 或
`objectives_changed`，Evidence 写清影响。否则上报 `solution_research_completed` 和片段路径。
