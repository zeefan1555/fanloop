---
status: accepted
date: 2026-08-25
amends: ADR-0066
amended_by: ADR-0090
---

# 自迭代采用预先确认的 Test Seam、选择性 TDD 与 Tracer Bullet

`fanloop-maintainer` 保留 ADR-0066 的三阶段八步拓扑和唯一 `confirm_requirements` Human Step，
但把其中笼统的 “Implement + TDD” 收紧为选择性 TDD。需求澄清必须提出尽可能少、尽可能高层且
优先复用现有入口的公开 Test Seams，并记录覆盖行为、基线失败信号、选择理由、不覆盖项和 TDD
适用性；这些 Seam 随完整改造计划由现有需求确认门禁一次批准，不新增审批节点。

Spec 只传递已确认 Seam，不在方案阶段重新选择。每张 Ticket 都是可独立演示或验证的 Tracer
Bullet，写明 Demo / verification path、Approved test seam 与 TDD applicability，不按 YAML、Skill、
测试或文档水平拆分。只有 Ticket 同时具有已确认公开 Seam 和独立于实现的正确预期时才执行
“一条失败测试 → 最小实现 → 转绿”；没有合格 Seam 的部分直接最小实现，不为满足流程制造测试。

Code Review 对照 requirements、Spec 与 Tickets 检查测试和切片没有漂移，并阻断未确认 Seam、
私有实现或内部调用次数断言、用实现算法重算预期、批量先测后做、无 Seam 造测试和缺少独立验证
路径的水平切片。阻断项继续沿 ADR-0066 的现有 Route 回到实现。

本决策只修改 `fanloop-maintainer/prompt.yaml` 文本、Release-bound self-iteration Skills、ADR 与
聚焦契约测试。八个 Step 的 id、name、顺序、executor，全部 Route、Condition 与 SkillBinding 集合
保持不变，`fanloop-dev-tdd` 继续 `optional: true`。不修改其他四份生产 YAML、Thrift IDL、生成代码、
Runtime、State/Storage/Output、公开 CLI、发布机制或测试入口。验证继续服从 ADR-0063；本次具备两个
已确认公开 Seam 且不改变推进语义，使用 `targeted`。ADR-0071 的人工 MR 交接和 ADR-0072 的 Session
复用不变。

生产 YAML 前后示例、Test Seams、单一 Ticket 与验收条件由张菲帆在确认卡
`om_x100b67fa3d9c04a0b1562aec8d6c230` 审阅，并通过消息
`om_x100b67fa3417a484b4b3b462b7f74fd` 批准进入实现。
