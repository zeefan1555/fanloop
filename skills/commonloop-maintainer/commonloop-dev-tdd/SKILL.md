---
name: commonloop-dev-tdd
description: 对 Commonloop CLI 修复执行测试驱动开发。用于在 Ticket 声明的已确认公开 Test Seam 上进行 red-green，并保留能捕获原始症状的最小回归测试。
---

# TDD

只在 Ticket 声明的已确认的公开 Test Seam 且存在独立于实现的正确预期时使用本 Skill；否则回到
Implement 直接最小实现，不为满足流程制造测试。测试公开行为，不测试私有实现。每轮只做一条
Tracer Bullet 纵向切片：

1. 在修复契约选定的 seam 写一个能捕获原始症状的测试。
2. 实际运行并确认它因目标缺陷失败，而非环境或测试错误。
3. 只写让这条测试通过的最小实现。
4. 重跑该测试和原始复现，确认 red 转 green。

预期值来自 Spec 或已知正确行为，不能用与实现相同的算法重算。不要一次写完所有测试，也不要 mock 内部协作者来伪造覆盖。
