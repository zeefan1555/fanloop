---
name: fanloop-dev-implement
description: 在 Build Step 持续完成实现、真实验证、失败诊断、修复和 verification-report.md。
---

# Build Until Verified

1. 确认源码位于独立 worktree，工作树基于 requirements.md 记录的 main。不得撤销其他 checkout 的改动。
2. 按最小纵向修改实现需求。跨模块协调、存在明显方案分支或需要独立并行任务时才生成 Spec/Tickets。
3. 持续执行：建立或复现基线、最小修改、聚焦测试、`./tests/run-unit`、Feature Impact Set 真实公开 CLI
   验证、证据与副作用回读、失败分类和修复。
4. 用户表面变化时读取 `fanloop-dev-maintain-verification/SKILL.md`，同步维护 Verification Skill 与
   Feature Map。doc drift 或 harness gap 随候选修正；product gap 作为实现缺陷修复。
5. 只有存在新的可证伪假设时继续重试。同一失败重复且没有新证据，或环境、授权不可达时保持 blocked。
   目标、范围或验收标准变化时回 Define。
6. 全部通过后提交当前分支，要求工作树 clean。记录 Acceptance/Impact 覆盖、run ID、manifest/snapshot、
   命令、退出码、失败分类、修复和 ADR impact 到 `verification-report.md`。
7. 记录严格候选身份：`candidate_head`、`git_tree`、候选 `binary_sha256`、
   `verification_contract_sha256` 和 `feature_impact_sha256`。

完成后返回 `self_validation_passed`、`verification_report_written` 和 `candidate_identity_recorded`。
本 Step 不 push、不创建 PR。
