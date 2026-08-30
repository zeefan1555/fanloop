---
name: fanloop-dev-eval-candidate
description: 在随机隔离目录中并行运行黑盒评测 Case，并保存可独立裁判的原始证据。
---

# 候选评测执行

只按冻结的 `eval-playbook.<sha256>.md` 工作，不读取其他候选结果，不修改候选仓库。执行前验证
Playbook 文件名摘要，以及恰好两个 brief_sha256 和 rubric_sha256；任一不一致立即 blocked。

1. 每个 Case 使用独立随机目录、数据根和全新 Requirement。
2. 多个无依赖 Case 并行执行原始 brief；同一 Case 内按公开用户路径串行操作，不复制或改题。
3. 使用锁定到 candidate_head 的 Release、项目验证技能和功能地图。
4. 保存完整命令、退出码、stdout/stderr、前后状态、截图或快照路径和 Cleanup 结果。

只有全部 Case 都结束且证据可读时生成 eval-candidates-report.md。环境、权限或服务不可用时标记
blocked，不把未运行或 mock 结果记为通过。
