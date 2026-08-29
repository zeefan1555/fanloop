---
status: accepted
date: 2026-08-29
amends: ADR-0009, ADR-0010, ADR-0032, ADR-0034, ADR-0065, ADR-0077, ADR-0082
---

# 将 Fanloop 硬切换为 Commonloop

通用 Workflow/Loop 引擎的唯一产品名改为 Commonloop。源码仓库使用
`zeefan1555/commonloop`，Go module 使用 `github.com/zeefan1555/commonloop`，CLI 与发布
归档使用 `commonloop`，GitHub Packages 包使用 `@zeefan1555/commonloop-cli`。

本地 Requirement 状态目录从 `.fanloop` 改为 `.commonloop`，用户数据目录从
`~/.fanloop` 改为 `~/.commonloop`，环境变量前缀从 `FANLOOP_` 改为 `COMMONLOOP_`。
旧目录、旧环境变量、旧命令和旧 npm package identity 不提供 alias、迁移、探测或 fallback；
已经发布的旧私有包仅作为不可变历史制品保留，不再由代码或发布 Workflow 引用。

统一入口改为 `commonloop-workflow`。维护场景、Workflow 与原子 Skill 分别改为
`commonloop-maintenance`、`commonloop-maintainer` 和 `commonloop-dev-*`，并继续满足
`workflows/<workflow-id>/` 与 `skills/<workflow-id>/` 一一对应。维护 Workflow 的八个 Step
`id`、`name`、顺序和 `executor` 全部不变；Condition、正常/回流 Route 与推进语义不变，
仅 Workflow identity、场景映射和 SkillBinding identity 改名。`technical-solution-design`
Workflow 不变。

Thrift 只同步公开文本中的 CLI 名、`.commonloop` 状态路径和 Commonloop 名称；field ID、
optional/required、类型、枚举、Service、Annotation 与 Error Code 均不改变。生成代码、Runtime、
持久化路径、Release Manifest、测试 Contract 和文档在同一提交中同步切换。
