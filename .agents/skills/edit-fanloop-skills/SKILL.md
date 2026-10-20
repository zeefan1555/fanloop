---
name: edit-fanloop-skills
description: 普通 Skill 的 write-skill 或 Workflow 的 Runtime manage-fanloop-workflow 当前步骤调用的内容辅助规则，覆盖已批准的非可执行指令和关联文档；验证及发布由对应步骤负责，不用于引擎、IDL 或脚本。
---

# Edit Fanloop Skills

新普通 Skill 需求选择 `write-skill`，新 Workflow 包需求选择 Runtime `manage-fanloop-workflow`。
先创建独立绝对 Root 并执行 `fanloop flow status --root <Root>`；只有 `NOT_INITIALIZED` 才用所选 ID 初始化并回读 Status。
已有 Run 按最新 Status 继续；其他错误先解决。本文件是当前步骤需要时调用的内容方法，不是独立维护或发布入口。
代码耦合 Skill 改真值源并按当前步骤交接独立源 PR 和镜像同步；已有 Run 按固定快照继续。

## 1. 固定范围

读取目标仓库 `AGENTS.md`、相关 ADR 和受影响的完整 Skill；涉及 Workflow 时读取同包五份 YAML 及绑定资源。确认实际 Git 根、远端、`main`、当前工作树和用户停止位置，从最新 `origin/main` 建隔离分支或 worktree，只改当前步骤已批准路径并保留无关改动。实际绑定的配置源在推送前保持已交付版本。

范围限于 `fanloop-workflow/<workflow-id>/` 的 YAML、非可执行 Skill Markdown，`<category>/<skill-id>/` 的仓库级 Skill 及必要关联文档。脚本、Runtime、IDL、Schema、安装或测试基础设施改动停止这条路径，说明代码范围后另行处理；目录名不是代码豁免。计划请求止于计划，不进行 Git 写入或业务写入。

## 2. 修改并检查

修改 Agent 指令时读取当前步骤绑定的 `writing-for-agents/SKILL.md` 和其 `SKILL-MECHANICS.md`：写清触发条件、顺序动作和可核对的完成标准，方法保持单一来源。该写作 Skill 不可用时报告缺失，不把其他 Skill 冒充它。

按改动语义复用现有检查，命令参数从当前 `--help` 读取：

- 错字、排版：检查 diff、链接和引用，说明行为试用不适用。
- 普通指令、模板：实际自检代表性输入与结果；Workflow 包使用现有 CLI 加载检查，仓库级独立 Skill 检查 frontmatter 和引用。
- Step、Condition、Route、绑定：用 Fanloop 代码仓库的 `scripts/validate-configuration.py` 和现有兼容 CLI，只选择受影响的 `--workflow`，产物放仓库外；保留完整包摘要。
- 路由、绑定、权限、安全、审批或外部写入规则：增加一次全新上下文 Agent 的独立试用，限定隔离目录、允许副作用和停止点，回读其原始产物。

校验失败先修复；候选变化后重跑受影响检查。使用 `git diff --check` 并核对最终 diff、ADR 和已确认范围。不为内容修改调整引擎、Golden 或重建 CLI，不以静态检查冒充行为试用。

## 3. 提交并推送

仅暂存已检查文件；`.scratch/`、`docs/research/`、`docs/specs/` 不得新增或修改入库，不使用 `git add -A`。将实际 diff、候选 head 和证据交回对应 Runtime 当前步骤；按该步骤创建分支 PR，独立审核后发布。
普通 Skill 的冻结快照安装、回滚与仓库交付归 write-skill，Workflow 的父 PR 和 gitlink 交付归 manage publish。
不向 `main` 直接提交或推送；用户只要求编辑或候选时止于明确边界。

## 4. 回读已交付配置

记录 second-brain-skills main 的精确提交与完整资源摘要。显式作者源用 FANLOOP_CONFIG_SOURCE 指向该仓库根并以公开 CLI 校验。默认运行源是不可变快照，不编辑 config/current 指向的文件。授权刷新时用全新隔离 Requirement 执行 flow init，回读 Doctor 的提交和摘要、Status 的绝对 Skill 路径；固定控制器保持原配置。没有刷新授权时报告已交付 main，下一次新 init 会自动拉取。

完成时给出候选 head、分支 PR、校验与已实际发生的回读证据；尚未审核、合并或同步时交回当前步骤并保留为未完成。不为配置正文修改重建 CLI。
