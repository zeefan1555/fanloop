---
status: accepted
date: 2026-08-30
amends: ADR-0082, ADR-0085
---

# 每次 main 更新后自动发布

`Release` GitHub Actions Workflow 从手工触发改为监听 `main` 的 `push`。每次 MR 合入或其他
变更进入 `main` 后，Workflow 自动解析下一个未发布的 patch 版本，发布 GitHub Packages
候选包，完成精确版本冒烟后提升 `latest`，再对真实 `latest` 执行同一冒烟。

删除 `workflow_dispatch` 和重复的 Job 级 `main` 判断，避免同一提交被自动与手工各发布一次。
使用 GitHub Actions 原生 `queue: max` 串行保留待发布任务，避免并发分配相同版本或连续 push
替换较早的待发布任务；失败恢复直接使用 GitHub Actions 原生 Re-run。旧 `latest` 恢复、
不可变版本、Manifest/Doctor/Workflow/Skill 配套校验与 `GITHUB_TOKEN` 权限边界均不改变。
发布过程不向仓库推送生成文件或 Tag，因此不会递归触发新一轮发布。

本决策只修改仓库发布入口，不修改生产五文件 Workflow、Step/Route/Condition/executor、Thrift
IDL、Runtime、State、Trace 或 Card 契约。
