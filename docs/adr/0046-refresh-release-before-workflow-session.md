---
status: accepted
date: 2026-08-17
amends: ADR-0009
amended_by: ADR-0049, ADR-0053
---

# Workflow 推进会话开始前刷新配套 Release

每次新的 Fanloop Workflow 推进会话，正式 `fanloop-workflow` Skill 在读取
Status 或执行其他业务命令前，只执行一次官方安装入口：

```bash
NPM_CONFIG_REGISTRY=https://bnpm.byted.org \
  npx --yes --package=fanloop-cli@latest -- fanloop install
```

官方安装入口是本机安装与版本判断的唯一责任方：未安装时安装完整
Release，当前版本较旧时安装并激活 `@latest`，目标版本已健康时幂等复用。
Skill 与人设不预先执行环境或版本探测，也不实现第二套更新逻辑。

安装成功后重新读取当前 Skill 入口的 `SKILL.md` 与 `ref/role.md`，使本轮规则
与刚激活的 CLI、Skills 和 Workflow Release 一致；重新读取不得触发同一会话的
第二次安装。刷新失败时不清理或改绑现有安装，报告错误并停止。

ADR-0009 的普通 CLI 业务命令仍只通过 `_notice` 提示更新，不在命令内部隐式
修改安装。本修订只增加正式 Workflow Skill 在首个业务命令前的显式安装步骤。
继续复用 ADR-0009、ADR-0024、ADR-0026、ADR-0032 和 ADR-0041 的完整制品校验、
Doctor 门禁、事务切换及失败保留当前版本；不在 Skill 或人设中新增版本判断、
更新器、后台任务、配置或兼容路径。
