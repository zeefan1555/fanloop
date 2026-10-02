---
name: fanloop-dev-merge-and-update-local
description: 交付已认证候选；在 Deliver 内吸收目标分支漂移、解决冲突，合并唯一 PR，并从精确 merge commit 更新源码 worktree 和本机 current。
---

# Merge And Update Local

从 certification-report.md 覆盖的 certification base/head、Git tree、candidate patch identity、binary
SHA-256 和验证资产摘要开始。Deliver 内受控集成目标分支后，必须回到 Build，把集成后的实际 head 作为新候选
重新完成自主验证、独立 Review、黑盒验收和主 Agent批准；任何认证事实都不得从旧 certification head 复用。ADR 文件名的四位数字前缀允许重复；同号但完整文件名不同不是
冲突，必须保留两份文件及原编号，不得因此改号、阻塞、回流或让候选重新验证。

1. 读取可选 `delivery-record.md`，先回读当前分支到 main 的唯一 PR，并跨 open/merged 状态核对。多命中
   blocked。若 PR 已为 `MERGED`，校验记录一致后固定其非空 merge commit，跳过 base drift 和再次合并，
   上报 `delivery_pr_already_merged=merged`，直接从第 7 步继续合并事实核验与本地恢复；该路径不要求
   `delivery_main_unchanged`。
2. PR 尚未合并时执行 `git fetch origin`。main 未变化时，要求工作树 clean、HEAD 等于
   certification_head、tree、patch、binary 与报告一致，并把 certification base/head 作为有效交付
   base/head，返回 `delivery_main_unchanged`。
3. `origin/main` 不等于 certification base 时，在 Deliver 内执行普通 merge 集成该精确 `origin/main`；禁止
   rebase 和 force push。记录合并前 base/head、最新 base、冲突文件和冲突解决内容。只允许为保留已批准需求
   及两侧有效改动而解决冲突，不夹带其他修改。执行 `git diff --check`、受影响聚焦检查和 `./tests/run-unit`。
   若集成要求改变已批准需求、Workflow Step 身份、IDL、Schema 或公开语义，返回 `requirements_changed`；
   其他非集成修改或身份漂移返回 `candidate_changed`。通过后提交 merge，固定 delivery base/head/tree 和命令
   回执，返回 `delivery_main_integrated` 并回 Build。旧 Review、黑盒、主 Agent批准、PR checks 和交付证据全部失效；在新 head 完成完整 Build 与 Certify 前不得 push、创建或更新 PR、等待 checks 或合并。
4. 仅当当前 HEAD、tree、patch、binary 与本轮 certification-report.md 的新 certification identity 完全一致，且
   `origin/main` 等于该 certification base 时，才不使用 force push 地推送有效交付 head；零 PR 时创建，唯一 open PR 幂等更新。PR 描述包含背景、问题、解法、影响、验证、ADR impact、主 Agent结论和交付边界。
5. 通过 GitHub API 读取作用于 `main` 的 active Ruleset，从 require-status-checks 规则动态提取 required checks
   的精确名称集合；不得假定、补写或硬编码检查名。无论该集合是否包含 `changes`，均须回读有效交付 head 的
   `changes` 成功结论及其真实 `code=true|false` 输出；代码或混合变更必须为 `true`。不得用本地分流、检查名称
   或 skipped 推断输出，缺失、失败、取消或无效输出均阻止交付。逐项回读适用 required checks 的结论，并同步、回读
   独立 Review 结论及适用的集成回执。产品失败回 Build；基础设施失败 blocked。已合并恢复也必须对原有效
   交付 head 重做这项核验。
6. 合码前再次 fetch；main 再次前进时重复第 3 步并回 Build 重新认证。否则先回读 PR source
   commit，必须等于有效交付 head。同 HEAD 的分流已成功、输出符合类别、全部适用 required checks 已通过且 Review 已同步后，不再请求额外的人或主 Agent
   合码确认；立即执行 `gh pr merge <PR_URL> --repo zeefan1555/fanloop --auto --squash
   --match-head-commit <EFFECTIVE_DELIVERY_HEAD>`。禁止 `--admin`、直接 push main 或第二个 PR。
7. 回读 PR 为 MERGED、merge commit 非空且可达 origin/main。parent 形态只能是有效交付 base 的单父提交，
   或依次为有效交付 base、有效交付 head 的双父 GitHub merge；其他形态 blocked。比较 merge commit tree 与
   有效交付 head tree，必须相同。
8. 立即把 PR、checks、review receipt、认证身份、适用的交付集成身份和 merge commit/tree 写入
   delivery-record.md。
9. 固定当前 Requirement 控制器。只把本 Requirement 的 clean 源码 worktree 切到 detached merge commit；
   不切换、重置、清理或删除其他 checkout。
10. 清除 data、config、Skill Root 和 Botmux 覆盖，从 merge commit 执行 `./scripts/install-local.sh`。
   全局 current 的 version commit 必须等于 merge commit，Doctor healthy，并运行 `fanloop verify smoke`。
   该次 smoke 写出的 schema-1 `manifest.json` 是 `post_merge_smoke_passed` 的 Evidence。文件 `outcome` 必须是 `passed`。不要手写 manifest。`OUTPUT_INVALID` 时重跑 smoke，按文件里的 outcome 重报。
11. 补全并回读 delivery-record.md，记录 lead time、Build/Certify 次数、返工来源和对应改进；只沉淀重复、
    可执行且可验证的规则。PR 已合并后的失败只重试同一 merge commit，不创建或合并第二个 PR。

集成目标分支时只返回 `delivery_main_integrated`、`panorama_presented` 与 `code_change`，回 Build 重新认证。
未发生集成或恢复已合并 PR 时，全部通过后返回本次实际的 `delivery_main_unchanged` 或
`delivery_pr_already_merged`，以及 `merge_request_published`、`remote_checks_passed`、`review_comment_synced`、`code_merged`、
`merge_tree_verified`、`source_repository_updated`、`local_cli_updated`、`post_merge_smoke_passed` 与
`delivery_record_written`。本 Skill 不发布远端制品。
