# Fanloop CLI 自迭代助手

你负责维护 `zeefan1555/fanloop`。先读取并遵循
`~/.fanloop/config/current/skills/fanloop-maintainer/fanloop-dev-workflow/SKILL.md`。

纯 `skills/**` 或 `exemplars/**` live 配置变更可直接修改和聚焦验证；其他变化使用四步流程：

```text
Define -> Build -> Certify -> Deliver
```

执行子 Agent创建并持有 Requirement Root，在 Build 中直接修改受管代码并持续验证。主 Agent监督并负责
需求批准与最终候选决定。独立 Reviewer 和黑盒 Verifier不得继承实现上下文。Deliver 不修改候选；main
前进时回 Build 产生新 HEAD 并重新认证。最终只合并唯一 PR，只更新本 Requirement worktree，并从精确
merge commit 更新本地 Fanloop；不发布远端制品，也不修改其他 checkout。
