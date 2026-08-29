---
status: accepted
date: 2026-08-30
amends: ADR-0009, ADR-0010, ADR-0032, ADR-0034, ADR-0065, ADR-0079, ADR-0081, ADR-0082
supersedes: ADR-0083, ADR-0084
---

# 恢复 Fanloop 与 GitHub Packages 发布

通用 Workflow/Loop 引擎的唯一产品名恢复为 Fanloop。源码仓库使用
`zeefan1555/fanloop`，Go module 使用 `github.com/zeefan1555/fanloop`，CLI 与发布归档使用
`fanloop`，配套 npm 包使用 GitHub Packages 的 `@zeefan1555/fanloop-cli`。

本地 Requirement 状态目录使用 `.fanloop`，用户数据目录使用 `~/.fanloop`，环境变量前缀使用
`FANLOOP_`。不保留 Commonloop 命令、目录、环境变量、Workflow ID、Skill ID、package identity、
alias、迁移、探测或 fallback；既有 Commonloop 状态与已发布包只作为历史事实保留。

统一入口恢复为 `fanloop-workflow`。维护场景、Workflow 与原子 Skill 分别使用
`fanloop-maintenance`、`fanloop-maintainer` 和 `fanloop-dev-*`，并继续满足
`workflows/<workflow-id>/` 与 `skills/<workflow-id>/` 一一对应。维护 Workflow 的 Step
`id`、`name`、顺序和 `executor`、Condition、正常/回流 Route 与推进语义均不改变，只修改
Workflow identity、场景映射和 SkillBinding identity。`technical-solution-design` 的 Step、
Condition、Route 与 Skill ID 不变；两套 Workflow 新增的 Panorama Skill 绑定继续保留。

正式发布恢复 ADR-0082 的 GitHub Packages 边界，只由私有仓库的 `Release` GitHub Actions
Workflow 手工触发。Workflow 使用仓库内置 `GITHUB_TOKEN` 和 `packages: write` 发布
`candidate`，依次完成精确版本鉴权安装冒烟、`latest` 提升与真实 `latest` 鉴权安装冒烟；最终
冒烟失败时恢复旧 `latest`。不使用 `NPM_TOKEN`，不向 npmjs 发布，不增加双发布或镜像。
读取私有包的用户仍需为 GitHub Packages 配置具备 `read:packages` 权限的 classic PAT。

Thrift 只同步公开文本中的 CLI 名、`.fanloop` 状态路径和 Fanloop 名称；field ID、可选性、
类型、枚举、Service、Annotation、Error Code 与 Schema Version 均不改变。生成代码、Runtime、
持久化路径、Release Manifest、测试 Contract 和文档在同一提交中同步硬切换。
