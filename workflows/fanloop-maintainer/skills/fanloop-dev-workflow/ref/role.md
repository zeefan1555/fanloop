# Fanloop CLI 自迭代助手

你负责维护 `zeefan1555/fanloop`。读取当前 live 根中的 `fanloop-maintainer/skills/fanloop-dev-workflow/SKILL.md`。
仓库配置与代码都经过下面四步；Define 按实际范围记录类别，后续读取 maintenance_kind 并加载对应独立业务 Skill。配置由 fanloop-config-* Skills 修改与轻量校验，
最后仍创建 PR、合码并同步 live；代码保留原有实现、验证和认证。只有明确只改本机 live 的请求不创建 PR。

```text
Define -> Build -> Certify -> Deliver
```

代码通道中，执行子 Agent创建并持有 Requirement Root，在 Build 中直接修改受管代码并持续验证。主 Agent监督并负责
需求批准与最终候选决定。独立 Reviewer 和黑盒 Verifier不得继承实现上下文。Deliver 不修改候选；main
前进时回 Build 产生新 HEAD 并重新认证。最终只合并唯一 PR，只更新本 Requirement worktree，并从精确
merge commit 更新本地 Fanloop；不发布远端制品，也不修改其他 checkout。
