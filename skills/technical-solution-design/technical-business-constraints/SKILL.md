---
name: technical-business-constraints
description: 从业务特点和目标问题中提炼真正约束技术选择的边界与优先级。用于 technical-solution-design 的业务特点与技术约束 Step；不调研或选择方案。
---

# 定义业务特点与技术约束

读取 `01-business-background.md`、`02-goals-and-problems.md` 和
[What-Why-How 推导标准](../technical-solution-review/references/reasoning.md)。逐项判断以下维度是否真实适用：

- 流量形态、峰谷、热点和容量边界；
- 数据一致性、时效性、幂等和资损边界；
- 延迟、可用性、资源与成本预算；
- 安全、隐私、合规和权限边界；
- 历史系统、兼容、迁移与交付窗口；
- 团队职责、跨团队依赖和组织协作边界。

只写会影响后续选型的事实约束。每项说明来源、违反后果、可接受边界和冲突时的优先级；不适用项
可省略，不为凑模板制造限制。后续方案必须能逐项回扣这里的约束。

## 产物列表

以下逻辑产物均写入 `.technical-solution/sections/03-business-constraints.md`：

| 逻辑产物 | 完整性要求 |
| --- | --- |
| 业务特点 | 说明场景区别于普通问题的真实特征及证据。 |
| 技术约束 | 每项约束有来源、边界和违反后果。 |
| 取舍优先级 | 冲突时明确优先顺序和允许牺牲的范围。 |

片段不得出现 `#` 或 `##`；允许 `###`，禁止 `####` 和手工 `1.1` 编号。发现上游变化时只上报
`background_changed` 或 `goals_and_problems_changed` 中最早的一项；否则上报
`business_constraints_defined` 和片段路径。
