---
name: fanloop-dev-to-tickets
description: 将 Fanloop CLI 本地 Spec 拆成最少的纵向实施 Tickets。用于在 Issue Workspace 生成可独立验证且有明确阻塞关系的任务。
---

# To Tickets

从 `spec.md` 形成窄而完整、可独立验证的纵向切片。小改默认一个 Ticket；只有单个新鲜上下文
无法安全完成或存在真实阻塞边时才拆分。

按依赖顺序写到 Issue Workspace 根目录的 `ticket-<NN>.md`，包含 What to build、Blocked by、
`Status: ready`、`Demo / verification path`、`Approved test seam`、`TDD applicability` 和可验证
Acceptance criteria。每张 Ticket 都必须是完成后可独立演示或验证的 Tracer Bullet，贯穿该切片
所需的层；不要按代码、YAML、Skill、测试或文档做水平拆分，不重新要求人确认，也不得新增 Spec
没有确认的 Seam。
