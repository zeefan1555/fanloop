---
status: accepted
date: 2026-08-19
amends: ADR-0026, ADR-0049, ADR-0057, ADR-0058
---

# 在 Claude Skill Root 暴露 Workflow Skill

Fanloop 在现有 Codex、Agent 和 Trae Skill Root 之外，新增固定的 Claude Skill Root：
默认路径为 `~/.claude/skills`，测试和非默认客户端可用
`FANLOOP_CLAUDE_SKILLS_ROOT` 覆盖。四个 Root 都只暴露 `fanloop-workflow`，并指向
Manifest 中该 Artifact 的 `current/<artifact-path>`；当前路径为
`current/skills/fanloop-workflow/common/fanloop-workflow`。不为其他原子 Skills 创建全局
软链。

安装、原生更新、Doctor 和匿名发布冒烟必须把 Claude 与现有三个客户端作为同一组受管
入口处理。普通文件和目录仍拒绝覆盖；外部软链的接管、失败回滚、旧受管原子 Skill 链接
清理及链接目标保护继续使用 ADR-0026、ADR-0041、ADR-0057 和 ADR-0058 的现有事务与
分组路径语义。

本决策修订 ADR-0026、ADR-0057 和 ADR-0058 中列举的客户端集合，并修订 ADR-0049 的
发布冒烟根目录数量。Release Manifest、Flow Skill `path`、Workflow YAML、IDL、
State/Event 和默认数据目录均不改变；不新增通用客户端注册表、自动发现或兼容路径。

具体确认来源、影响和验收条件见 `docs/specs/claude-workflow-skill-link.md`。
