---
status: accepted
date: 2026-09-17
amends: ADR-0092, ADR-0097, ADR-0100, ADR-0106
---

# Maintainer 合并 PR 并更新本地 Fanloop

`fanloop-maintainer` 保持 ADR-0106 的前三个 Stage、前三个 Job、前八个 Step、顺序和 executor 不变，
只把第九个 Step 从：

```yaml
- {id: handoff_merge_request, name: MR 门禁与交接, executor: agent}
```

替换为：

```yaml
- {id: merge_and_update_local, name: PR 合码与本地更新, executor: agent}
```

该 Step 幂等创建或更新当前分支到 `main` 的唯一 PR，等待精确 final head 的 required checks，随后用
`gh pr merge --auto --squash --match-head-commit` 合并；禁止 `--admin`、直接 push `main` 或创建第二个
PR。只有 PR 为 `MERGED`、merge commit 非空、恰好一个 parent 且等于 `final_base`，并可达 `origin/main` 才视为
合码完成。

合码后先固定当前 Requirement 控制器，再只把本 Requirement 的干净源码 worktree 切到 detached
merge commit。不得切换、重置、清理或删除其他 checkout。随后从该提交执行
`./scripts/install-local.sh`，并要求全局 current 的 `version.commit_sha` 等于 merge commit、Doctor
healthy。若 PR 已合并但本地阶段失败，先从跨 open/merged 状态查询回读同一 merge commit；
`delivery-record.md` 存在时必须一致，缺失时从远端已合并事实重建；恢复时重新核验 required checks 和
Review comment receipt。允许源码 worktree 位于 `reviewed_head`、预合并 `final_head` 或该 detached
merge commit，并只重试未完成阶段。安装前清除 data、config、Skill Root 和 Botmux 环境覆盖。PR 描述保留
背景、问题、解法、影响、验证、ADR impact、主 Agent 结论和交付边界。

为避免不同 clean commit 争用同一个不可变 Release 目录，根级 `VERSION` 改为本地构建的基础版本：
clean build 使用 `<VERSION>-dev.<commit-prefix>`，dirty build 继续使用
`<VERSION>-dev.<source-digest-prefix>`。同一 clean commit 重建仍复用同一 Release。该规则取代
ADR-0097 中“clean build 直接使用 VERSION”的部分，不修改 Manifest、Thrift IDL 或公开 version 字段。

生产五文件变为 51 Conditions、38 个 Flow Route objects、45 个 Loop Route objects 和 57 Prompts。
不修改前八个 Step、任何 executor、Workflow Schema、State/Event/Output Storage 或 Thrift IDL。

验证包括聚焦 Workflow、Contract、Runtime 与 local-build 测试、`./tests/run-unit`、同一候选的隔离
Release Doctor/Feature 验证，以及无实现上下文的公开 CLI 黑盒验收。

人工审核记录：用户在收到上述第九个 Step 的精确 before/after YAML 后，于 2026-09-17 回复
“批准这个具体 YAML 变更。”；decision receipt 为
`decision-da7e6c1cf47bcd8a6a791b002cf2751a42edd04ab5a592835ab197770bf1eceb`。
