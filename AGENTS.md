# Fanloop 变更门禁

## 边界

1. Fanloop 本仓库维护 CLI、通用 Runtime、存储、加载器、验证器、Release 与分发。当前生产产品工作流已迁到 `second-brain-agent/plugins/` 的 Codex 插件；本仓库不再保存生产 Workflow 包。
2. `entrypoints/fanloop-workflow/routes.yaml` 保留空的 `scenarios`，仅用于既有 pinned Requirement 兼容和通用 Runtime 测试。禁止新增生产场景、恢复已退役包或把插件包装成 Workflow。
3. `.scratch/`、`docs/research/` 和 `docs/specs/` 只可作本地临时材料，不得提交。
4. 普通代码、配置、Skill 和文档任务直接执行；用户明确要求修改时才写，调研、解释和审阅默认只读。无关脏状态保留，不 stash、不 reset、不清理。
5. 审查归属与跨边界检查按 [docs/ownership.md](docs/ownership.md)；产品插件不在本仓库修改。

## 修改流程

1. 从最新 `origin/main` 建独立 `codex/<主题>` worktree 和分支；逐路径暂存，不提交无关文件。
2. 纯文档改动检查相对链接、命令和 `git diff --check`。代码改动先补受影响测试，再运行对应的聚焦检查和一次 `./tests/run-unit`。
3. 用户可见 CLI、安装、状态、投影或 Skill 行为变化，按 [`.cursor/skills/verify-fanloop/SKILL.md`](.cursor/skills/verify-fanloop/SKILL.md) 用同一候选的隔离 Release 验证；不能用源码阅读或旧 Release 代替。
4. 生产产品插件在父仓库自己的插件测试中验证，不在本仓库补第二套配置测试。合成 Bundle 只用于通用 Runtime 回归。

## IDL 与持久化

`idl/*.thrift` 是公开契约真值，默认只读。确需修改时，先在对话中说明字段、类型、枚举、方法、兼容影响、生成物、调用方、迁移策略和不改 IDL 的替代方案；用户明确批准后再以 IDL 为第一修改点生成代码。

State、Event、Output、Card 或存储 Schema 变化不保留旧路径兼容层。是否 migrate 由用户对具体方案确认；没有批准时采用历史快照继续只读，不静默改写旧数据。

## ADR

每次行为或架构变化先检索并读取 `docs/adr/` 的相关决定与取代链。与仍有效 ADR 冲突时新增或修订 ADR，再同步设计、代码、测试和文档；历史 ADR 保持原样。

## 发布与交付

开发构建使用 `./scripts/build-local.sh`，隔离安装使用 `./scripts/install-local.sh --config-source <合成配置>`。GitHub Release 继续由 `vX.Y.Z` tag、跨平台归档、校验和与薄 npm 包装交付；不发布到 npm registry。

交付使用同任务 PR，确认候选基于当前远端 `main`、所需检查通过且 head 未变化；不自动合并到用户未授权的外部系统。合并后按父仓库规则更新精确 gitlink 并同步本地 `main`。
