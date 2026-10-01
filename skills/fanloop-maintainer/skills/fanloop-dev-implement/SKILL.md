---
name: fanloop-dev-implement
description: 在 Build Step 持续完成实现、真实验证、失败诊断、修复和 verification-report.md。
---

# Build Until Verified

1. 确认源码位于独立 worktree，工作树基于 requirements.md 记录的 main。不得撤销其他 checkout 的改动。
2. ADR 文件名的四位数字前缀允许重复；同号但完整文件名不同不是冲突，保留两份文件及原编号，不得
   因此改号、阻塞、回流或让候选重新验证。
3. 按可独立验证的小批纵向切片实现需求；每批只关闭一个风险或验收缺口并立即运行最小聚焦检查。
   跨模块协调、存在明显方案分支或需要独立并行任务时才生成 Spec/Tickets。
4. 稳定候选再执行：`./tests/run-unit`、Feature Impact Set 真实公开 CLI 验证、证据与副作用回读、失败分类和修复。
   失败归类为 product、test/harness、environment 或 contract drift；没有新证据时不得重复同一尝试。
5. 用户表面变化时读取 `fanloop-dev-maintain-verification/SKILL.md`，同步维护 Verification Skill 与
   Feature Map。doc drift 或 harness gap 随候选修正；product gap 作为实现缺陷修复。
6. 只有存在新的可证伪假设时继续重试。同一失败重复且没有新证据，或环境、授权不可达时保持 blocked。
   目标、范围或验收标准变化时回 Define。
7. 全部通过后提交当前分支并确认工作树 clean。记录 Acceptance/Impact 覆盖、run ID、manifest/snapshot、
   命令、退出码、失败分类、修复和 ADR impact 到 `verification-report.md`。Pre-Certify 由下一 Certify Step
   对这个已提交 HEAD 执行，不在提交前冻结身份。
8. 记录严格候选身份：`candidate_base`、`candidate_head`、`git_tree`、`patch_sha256`、候选 `binary_sha256`、
   `verification_contract_sha256` 和 `feature_impact_sha256`。

完成后返回 `self_validation_passed`、`verification_report_written` 和 `candidate_identity_recorded`。
本 Step 不 push、不创建 PR。
