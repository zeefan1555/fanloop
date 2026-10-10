---
name: edit-fanloop-skills
description: 直接维护 second-brain-agent 配置中心的 Workflow YAML、Skill 指令和关联文档，完成校验、PR 合并及本地配置回读；不启动自迭代。用于 skills/ 和 .agents/skills/ 的内容修改，不用于引擎、IDL 或可执行脚本。
---

# Edit Fanloop Skills

由当前 Agent 完成内容修改与 Git 交付。维护过程不创建 Requirement、不推进 Maintainer、不展示维护 Panorama；被修改的业务 Workflow 仍遵守自己的校验与审批规则。

## 1. 固定范围

读取目标仓库 `AGENTS.md`、相关 ADR 和受影响的完整 Skill；涉及 Workflow 时读取同包五份 YAML 及绑定资源。确认实际 Git 根、远端、默认分支、已有 PR 和用户停止位置，保留无关修改。以最新目标分支创建工作分支；实际绑定的配置源保持已交付版本，候选在独立 worktree 中修改。

范围限于 `skills/fanloop-workflow/<workflow-id>/` 的 YAML、非可执行 Skill Markdown、`.agents/skills/<skill>/` 的非可执行 Markdown 及必要关联文档。脚本、Runtime、IDL、Schema、安装或测试基础设施改动停止这条路径，说明代码范围后另行处理；目录名不是代码豁免。计划请求止于计划，不进行 Git 写入或业务写入。

## 2. 修改并检查

修改 Agent 指令时使用可用的 `writing-for-agents` Skill：写清触发条件、顺序动作和可核对的完成标准，方法保持单一来源。该写作 Skill 不可用时报告缺失，不把其他 Skill 冒充它。

按改动语义复用现有检查，命令参数从当前 `--help` 读取：

- 错字、排版：检查 diff、链接和引用，说明行为试用不适用。
- 普通指令、模板：实际自检代表性输入与结果；Workflow 包使用现有 CLI 加载检查，仓库级独立 Skill 检查 frontmatter 和引用。
- Step、Condition、Route、绑定：用 Fanloop 代码仓库的 `scripts/validate-configuration.py` 和现有兼容 CLI，只选择受影响的 `--workflow`，产物放仓库外；保留完整包摘要。
- 路由、绑定、权限、安全、审批或外部写入规则：增加一次全新上下文 Agent 的独立试用，限定隔离目录、允许副作用和停止点，回读其原始产物。

校验失败先修复；候选变化后重跑受影响检查。使用 `git diff --check` 并核对最终 diff、ADR 和已确认范围。不为内容修改调整引擎、Golden 或重建 CLI，不以静态检查冒充行为试用。

## 3. 提交并合并

仅暂存已检查文件；`.scratch/`、`docs/research/`、`docs/specs/` 不得新增或修改入库。提交和 squash 消息的最终 trailer block 中 `Co-Authored-By: Aiden` 恰好一次。推送工作分支，用 `gh` 创建或更新唯一 PR，写明范围、检查结果与 ADR impact。

合并前重新读取 PR head、目标分支与适用 Ruleset，确认同一 head 的 `changes` 成功及实际 `code` 输出，并等待全部适用 required checks。允许分流明确跳过引擎任务，不跳过失败、未知或缺失检查，不使用 admin 绕过。候选或相关基线变化时先重新验证；合并使用明确 subject/body 和 `--match-head-commit`。

按用户授权 squash 合并并回读 `MERGED`、merge commit 与候选内容；发起 auto-merge 不代表已合并。用户只要求编辑、提交或提 PR 时停在该处，不自行合并或更新配置。

## 4. 回读已交付配置

记录 second-brain-agent main 的精确提交与完整资源摘要。显式作者源用 FANLOOP_CONFIG_SOURCE 指向该仓库根并以公开 CLI 校验。默认运行源是不可变快照，不编辑 config/current 指向的文件。授权刷新时用全新隔离 Requirement 执行 flow init，回读 Doctor 的提交和摘要、Status 的绝对 Skill 路径；固定控制器保持原配置。没有刷新授权时报告已交付 main，下一次新 init 会自动拉取。

完成时给出配置提交、适用的 PR URL、校验和实际快照回读证据。不为配置正文修改重建 CLI。
