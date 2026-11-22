# 验证与交付

Fanloop 代码变更直接在任务 worktree 中推进。先运行受影响的最小测试，再对稳定候选运行一次 `./tests/run-unit`。用户明确要求时，可使用独立 Codex 子 Agent 做范围明确的验证；不得把未请求的 Fanloop Workflow 当作协作前置。

用户可见 CLI、安装、状态或投影变化按 [`.cursor/skills/verify-fanloop/SKILL.md`](../.cursor/skills/verify-fanloop/SKILL.md) 使用同一候选的隔离 Release 验证。证据必须绑定当前 HEAD；完成后再刷新完整 diff、ADR、生成物和所需 PR checks，候选变化则补验受影响范围。

交付只使用同任务 PR，不自动合并到未授权的外部系统。合并后更新精确 gitlink、同步本地 `main` 并回读。
