---
name: fanloop-dev-merge-and-update-local
description: 在 fanloop-maintainer 最终 Step 合并唯一 PR，并把精确 merge commit 更新到本 Requirement 源码 worktree 和本机 current。
---

# Merge And Update Local

只接受整体 Review、本地验证、Agent 验收和 human 验收覆盖的同一 `review_base` / `reviewed_head`。

1. 读取可选的 `delivery-record.md`，再用 `gh pr list --head <branch> --base main --state all` 查找唯一
   PR。若 PR 已 merged，要求远端 head 等于 `final_head`、merge commit 第一 parent 等于 `final_base`；
   记录存在时必须与远端一致，记录缺失时从这些远端事实重建并回读。源码 worktree tracked-clean 且
   `HEAD` 只能等于 `reviewed_head` 或该 merge commit；随后跳过候选和 PR 阶段，只重试未完成的源码/CLI
   更新。其他情况执行下列正常路径。
2. `git fetch origin`，要求源码 worktree tracked-clean、`HEAD == reviewed_head` 且 merge-base 等于
   `review_base`。`origin/main == review_base` 时保留 unchanged 事实；只有 main 前进时在本 Step
   合入精确 SHA，按 Prompt 完成集成 Review、测试和 human 核验。
3. 用同一条 `gh pr list --head <branch> --base main --state all` 结果处理 PR。零命中才 push/create，唯一
   open PR 幂等更新，唯一 merged PR 回读并复用其 merge commit；多命中 blocked。PR 描述必须幂等包含
   背景、问题、解法、影响、验证、`ADR impact`、human 结论和交付边界。回读 state、target、head SHA、
   base SHA、merge commit、描述和 merge-base。
4. 回读 active main Ruleset 的 required checks；`test (ubuntu-latest)`、`test (macos-latest)`、`requirement-e2e`、
   `install-doctor`、`governance` 必须在 final head 全部成功。同步并回读已有 Review 评论。
5. 合码前再次 fetch；main 前进时重新执行集成短门禁。随后执行：

   ```bash
   gh pr merge <PR_URL> --repo zeefan1555/fanloop --auto --squash --match-head-commit <FINAL_HEAD>
   ```

   禁止 `--admin`、直接 push main 或创建第二个 PR。只接受回读 `state=MERGED`、非空 merge commit、
   merge commit 第一 parent 等于 `final_base`，且该提交可达最新 `origin/main`。立即把这些不可变事实
   写入并回读 `delivery-record.md`，供失败重入使用。
6. 在切换全局 current 前执行
   `../fanloop-dev-workflow/scripts/pin-controller-release.sh <ABSOLUTE_REQUIREMENT_ROOT>`，并回读固定
   控制器 healthy。之后本 Requirement 的 Flow/Card/Trace/Doctor 只用固定控制器。
7. 要求本 Requirement 的源码 worktree 无 tracked 或 untracked 改动，随后将它切到 detached merge
   commit 并精确回读 HEAD。不得切换、重置、清理或删除其他 checkout。
8. 清除 `FANLOOP_CONFIG_SOURCE`、`FANLOOP_CONFIG_ROOT`、`FANLOOP_DATA_HOME`、四类 Skill Root、
   `BOTMUX_CHAT_ID` 和 `BOTMUX_SESSION_ID` 覆盖，从该 merge commit 执行 `./scripts/install-local.sh`。
   回读 `$HOME/.fanloop/current`、`fanloop version` 与 `fanloop doctor`；commit 必须等于 merge commit，
   Doctor 必须 healthy。
9. 补全并回读 `delivery-record.md`，记录 Requirement、branch、PR、final base/head、merge commit、
   Ruleset/checks、Review comment、源码 worktree、installed Release/current 和生成时间。

PR 已合并但源码或 CLI 更新失败时，保留精确 merge commit 并上报 blocked；重入时回读同一 PR 后只
重试未完成的本地阶段。不得再次创建或合并 PR，不得删除不可变 Release 或手改 current symlink。

全部事实一致后返回 `code_merged`、`source_repository_updated`、`local_cli_updated`、
`delivery_record_written` 与其他当前 Route 要求的 Conditions。本 Skill 不发布远端制品。
