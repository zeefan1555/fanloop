---
name: fanloop-dev-bootstrap
description: 为 Fanloop CLI 反馈创建隔离 Issue Workspace 与源码 Worktree，固定 main 诊断基线。用于 fanloop-maintainer Workflow 的 bootstrap_techdesign Step；不定义问题、不修改代码。
---

# 准备 Fanloop CLI 维护工作区

1. 从反馈生成稳定的 `<issue-slug>`，使用 `~/fanloop/issues/<issue-slug>/repo/` 作为源码
   Worktree。先确认目标路径不属于其他 Issue，避免覆盖现有工作区。
2. 读取本地 `main`；存在 `origin` 时先校验它指向 Fanloop GitHub 仓库，再运行
   `git fetch origin --prune` 并使用最新 `origin/main`，否则直接使用本地 `main`。以该 Commit
   创建或复用隔离 Worktree；已有 Worktree 非干净或基线不一致时停止并报告，不重置、不删除用户改动。
3. 记录 Issue slug、源码路径、`main@<sha>`、基线来源、当前分支或 detached 状态和
   `git status --short` 到 `<issue-workspace>/bootstrap.md`。
4. bootstrap 阶段只准备只读诊断环境。不得创建开发提交、修改产品文件、输出实施计划，
   也不得恢复 `fanloop_dev` build tag、Overlay 或专用安装器。
5. 回读文件与 Git 事实；一致时返回 `repository_workspace_prepared`，其 Output 为
   Issue Workspace 路径。

完成标准：诊断明确基于已记录的 `main` 基线；用户已有改动未被覆盖；后续
`clarify_requirements` 可以直接在该 Worktree 只读复现。
