---
name: fanloop-dev-maintain-verification
description: 逐 Feature 校准 Fanloop 的 Feature Map 与验证 Skill。用于 reviewed HEAD 的 Agent 验收前置维护，也可由未来日程直接调用。
---

# Maintain Verification

以仓库根 FEATURE_MAP.md 的每一行作为一个 Feature 单位，由当前协调 Agent 串行完成；不派生每
Feature 子 Agent，不创建外部日程。

## 前置检查

1. 记录当前分支、HEAD、git status --short，以及 local-test-report.md 和 review-report.md 的
   reviewed HEAD。三者不一致或工作树已有未说明改动时输出 blocked。
2. 运行当前候选规定的健康检查；只有健康候选才执行 live 验证。
3. 检查近期 diff、公开 Help、Workflow/IDL/Release 真值，找出新增、删除或变化的用户表面。

## 逐 Feature 维护

对 FEATURE_MAP.md 每一行同时完成：

- Source：核对公开命令、稳定代码 Seam、相关五文件/IDL 与近期变化。
- Live：按该行的最小真实验证串行驱动公开入口，保存命令、退出码、stdout/stderr 和状态/文件变化。
- Safety：外部写入只能走已声明的 dry-run/read-only 配方；不能因命令名含 dry-run 就假设无副作用，
  仍需观察文件、网络、Git ref 和 State。
- Evidence：cleanup 前保存证据，cleanup 后确认该证据仍可读。

只允许修改 FEATURE_MAP.md、fanloop-dev-verify、fanloop-dev-maintain-verification 及它们自有的
验证 harness。产品行为与仍正确的 Map 不一致时是 product gap，不得改文档掩盖。

## 结果

- clean：每个 Feature 都有 source + live 证据，验证资产无需修改。
- changed：验证资产已修正，或发现 product gap。把修改、证据和原因写入 Issue Workspace 的
  acceptance-report.md；候选已变化，旧本地验证与 Review 立即失效，回 implement_code。
- blocked：健康检查、source/live 全覆盖或证据保留无法完成。写入 acceptance-report.md 并停在
  execute_agent_acceptance；不提交通过或产品失败 Result。

只有 clean 才能继续安装候选 Release 和真实机器人验收。不得在本 Step 提交或继续沿用旧 Review。
