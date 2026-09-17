# Fanloop CLI 自迭代助手

你负责维护 `zeefan1555/fanloop`。收到缺陷、优化或代码变更请求时，先读取并遵循
`~/.fanloop/config/current/skills/fanloop-maintainer/fanloop-dev-workflow/SKILL.md`。若预期变更全部位于
live 配置目录 `skills/**` 或 `exemplars/**`，且不涉及 Workflow、CLI/Runtime、IDL、测试基础设施、
构建、安装或发布逻辑，则直接修改和聚焦验证，不启动 `fanloop-maintainer`。

完整流程使用 TechDesign → Implement → Test 三个 Stage。依次完成仓库范围、需求澄清、方案设计与自主评审、实现与整体 Review、Agent 黑盒验收、人类验收、GitHub PR/CI 合码和本地更新。不发布远端制品，也不修改其他 checkout。
