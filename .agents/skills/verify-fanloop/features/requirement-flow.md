# Requirement 与 Flow

用户创建隔离 Requirement、查看当前 Step、提交类型化结果，并观察五份 YAML 配置的 Route 推进。

## 用户入口

- fanloop flow status --root ROOT
- fanloop flow init --root ROOT --workflow WORKFLOW --title TITLE
- fanloop flow report result 按最新 Status 提供的 Condition 和 Route 提交

## 真实驱动

1. 在空的一次性绝对目录执行 status，要求非零退出且 stderr 含 NOT_INITIALIZED。
2. 执行 init --dry-run，要求零退出且不产生 flow/state.json；CLI 日志可以新增。
3. 真正 init 后读取 status，要求当前 Step 为 bootstrap_techdesign，Stage 为本地验证机制，Job 为需求与方案。
4. 对 repository_workspace_prepared → clarify_requirements 先 report result --dry-run，比较 State 哈希不变。
5. 真正提交同一结果，再读取 status，要求当前 Step、issue_workspace_path、producer 和 Event 均正确。

验证 `technical-solution-design` 时使用另一份全新 Requirement：初始化后要求当前 Step 为
`frame_requirement_background`，公开 Prompt 同时包含“具体业务场景”“定量事实”“来源和证据状态”；
随后回读 Status，确认 Condition、Route 与 Skill 路径仍来自同一候选 Bundle。

保存两次 Status、命令输出、State/Output/Events 和哈希。不得手改 State、复用旧 Requirement，或从
内部 Go 结构推导成功。
