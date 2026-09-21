---
name: technical-retrospective-planning
description: 从方案、落地和结果证据中提炼复盘判断与下一阶段规划。用于 technical-solution-design 的复盘与后续规划 Step；不改写已确认结果。
---

# 复盘并规划后续演进

读取第 1 至第 9 章、全部结果证据和
[What-Why-How 推导标准](../technical-solution-review/references/reasoning.md)，围绕三件事形成闭环：

1. 哪些判断被事实证明是正确的，证据是什么；
2. 哪些地方走了弯路，原因、影响和可复用教训是什么；
3. 下一阶段为什么现在应该推进，关键里程碑、前置条件、完成信号和风险是什么。

规划不得只写“平台化”“智能化”等方向词。每项演进必须说明它解决的新问题、与当前方案的关系、
投入条件和不做的后果。晋升场景突出判断与规划能力；分享场景突出可复用经验和适用边界。

## 产物列表

以下逻辑产物均写入 `.technical-solution/sections/10-retrospective-and-roadmap.md`：

| 逻辑产物 | 完整性要求 |
| --- | --- |
| 正确判断 | 有结果证据支持，不做事后包装。 |
| 弯路与经验 | 说明原因、代价和下一次可复用的改进。 |
| 后续规划 | 每项有动因、里程碑、完成信号和风险。 |

片段不得出现 `#` 或 `##`；允许 `###`，禁止 `####` 和手工 `1.1` 编号。发现上游变化时只上报
最早受影响的 feedback Condition；否则上报 `retrospective_and_roadmap_defined` 和片段路径。
