---
name: technical-summary-writing
description: 在正文全部完成后生成五至八行摘要，覆盖问题、方案、结果、贡献和规划。用于 technical-solution-design 的摘要 Step；不得引入正文外的新事实。
---

# 编写摘要

最后读取第 1 至第 10 章、其证据和
[What-Why-How 推导标准](../technical-solution-review/references/reasoning.md)，用 5 至 8 行写清：

1. 业务背景与为什么值得做；
2. 2 至 3 个核心问题的浓缩表达；
3. 技术方案一句话；
4. 最关键且可验证的结果；
5. 个人贡献与团队贡献边界；
6. 下一阶段规划。

晋升场景突出能力等级、owner 范围和影响力证据；技术分享突出方法、边界和复用价值；融合场景兼顾
两者。摘要只能压缩正文已有结论，不新增数字、承诺或因果关系。

## 产物列表

以下逻辑产物写入 `.technical-solution/sections/00-summary.md`：

| 逻辑产物 | 完整性要求 |
| --- | --- |
| 摘要 | 5 至 8 行覆盖背景、问题、方案、结果、贡献与规划，且每项能回链正文。 |

片段不得出现 `#`、`##` 或其他标题。发现上游变化时只上报最早受影响的 feedback Condition；否则
回读行数和事实一致性后上报 `summary_defined` 和片段路径。
