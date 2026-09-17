---
name: fanloop-dev-mr-handoff
description: 在 GitHub PR 的候选身份、required checks 和 Review 评论全部通过后，生成并回读最终交接记录。
---

# MR Handoff

只接受 `fanloop-dev-mr-gate` 已回读的唯一 PR、final base/head、required checks 和 Review 评论。本 Skill 不 approve、不 merge、不发布、不更新本地 CLI。

1. 幂等更新 PR 描述，至少包含背景、问题、解法、影响范围、验证命令、ADR impact、主 Agent 验收结论和交付边界。
2. 在 Issue Workspace 写入 `handoff-record.md`，记录 Requirement、branch、PR URL、final base/head、Ruleset、required checks、Review comment receipt 和生成时间。
3. 回读 PR 和 `handoff-record.md`，再 fetch 一次确认 main、PR head 和 merge-base 未漂移。
4. 向主 Agent明确交付 PR URL、source/base、final head 和 checks，不直接联系用户；只在这些事实完整一致时上报 `merge_request_handed_off=passed` 和 `handoff_record_written=handoff-record.md`。

平台写入不确定时先回读，不重复创建 PR 或评论。只有明确的 remote check 失败或非 main 前进的候选漂移才回实现。
