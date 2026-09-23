---
name: fanloop-config-deliver
description: 交付已检查的 Workflow/Skill 配置候选：PR、合码、同步已授权 live 文件并回读；仅在 Maintainer Deliver 的配置 Route 使用，不构建或安装 CLI。
---

# 交付配置

只交付 `configuration_check` 对应的同一 HEAD/tree。默认分支取 GitHub 实际设置，不猜 main/master。候选或基线变化需要重新检查时，用 `configuration_needs_fix` 回 Build；不要复用旧检查结果。

1. 用 gh 跨 open/merged 状态查找当前分支的唯一 PR，没有则 push 并创建。按本次已授权的交付范围继续 squash 合并，用 `--match-head-commit` 限定候选；实际 required checks 必须满足，不使用 `--admin`。未设置必需检查时不额外引入代码门禁，不声称未完成的 CI 通过。
2. 回读 MERGED、merge commit、默认分支可达性和候选 tree 相等。只将本任务的干净源码 worktree 更新到该 merge commit。
3. 从精确 merge commit 更新本次批准的 live Workflow/Skill 文件。先比较 Define 记录的旧摘要，发现并发修改保持 blocked；保留未修改文件，不自行播种额外 Workflow。写后核对每个文件与 merge commit 内容一致，并运行已有 CLI 的配置 Doctor。
4. 将 PR、merge/tree、源码位置、live 路径与文件摘要、Doctor 结果及耗时写入 `delivery-record.md`。成功后提交本配置 Route 的全部真实事实：`merge_request_published`、`code_merged`、`merge_tree_verified`、`source_repository_updated`、`live_configuration_synced`、`delivery_record_written`，由公共入口附加类别与全景回执并 terminal。

不构建、不安装 CLI，也不热改不可变 Release 内的 entrypoint/Manifest。需要这些变更时重新确认代码范围。合码后的 live 同步失败保持 Deliver blocked，只恢复同一 merge commit，不另造 PR或伪造同步成功。
