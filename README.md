# Fanloop

Fanloop 是由五份 YAML Bundle 驱动的通用本地 Loop 引擎。Go Runtime 不认识任何生产
Workflow、Step、Condition、Output 或原子 Skill ID，只负责严格加载、路由校验、状态转换、
回流失效、审计与投影：

- 命令：`fanloop`
- Requirement 状态：`.fanloop/`
- 用户数据：`~/.fanloop/`
- 环境变量：`FANLOOP_*`
- Go module：`github.com/zeefan1555/fanloop`

当前发布两套五文件 Workflow Bundle，不设置默认 Workflow：

- `technical-solution-design`：完成问题定义、方向推导、正式方案写作、独立审校和三级人工确认。
- `fanloop-maintainer`：Fanloop 自迭代工作流。

业务配置严格按 `workflows/<workflow-id>/ ↔ skills/<workflow-id>/` 一一对应，不设例外。
统一入口独立放在 `entrypoints/fanloop-workflow/`，场景映射由其中的 `routes.yaml` 配置。

新增一套流程只需要增加同名 Workflow/Skill 目录和一条场景映射；不注册 Go 代码。Trace Registry
的部署与 Workflow 差异位于 `internal/traceconfig/registry.yaml`，同样不进入业务 Runtime。

## 使用 npx 安装

Fanloop 作为私有包 `@zeefan1555/fanloop-cli` 发布到 GitHub Packages。先创建具备
`read:packages` 权限的 GitHub classic PAT，再安装匹配的 CLI、Workflow 与 Skills。执行
`npm login` 后，`Password` 必须粘贴这个 PAT，不能使用 GitHub 密码或 npmjs token：

```bash
npm login --scope=@zeefan1555 --auth-type=legacy --registry=https://npm.pkg.github.com
NPM_CONFIG_REGISTRY=https://npm.pkg.github.com \
  npx --yes --prefer-online --package=@zeefan1555/fanloop-cli@latest -- fanloop install
fanloop version
fanloop doctor
```

后续升级执行 `fanloop update`。从源码安装需要 Node.js 18+、Go 1.23+，系统 `tar` 支持 XZ：

```bash
npm run install:local
fanloop version
fanloop doctor
```

安装结果位于 `~/.fanloop/current`。如只需开发二进制：

```bash
go build -o ./bin/fanloop .
./bin/fanloop version
```

## 使用

新 Requirement 必须先显式选择场景：

- `technical-solution` → `technical-solution-design`
- `fanloop-maintenance` → `fanloop-maintainer`

`entrypoints/fanloop-workflow/routes.yaml` 没有默认值；未选择或场景未知时停止初始化。
选择技术方案场景后执行：

```bash
mkdir -p /absolute/path/to/requirement
fanloop flow init \
  --root /absolute/path/to/requirement \
  --workflow technical-solution-design \
  --title "My technical solution"
fanloop flow status --root /absolute/path/to/requirement
```

Agent 的统一入口是 `fanloop-workflow` Skill。它按以下闭环推进：

```text
flow status -> 执行当前 Prompt/Skills -> flow report progress/result -> flow status
```

`technical-solution-design` 的七个 Step 各绑定一个独立 Skill，领域产物写在 Requirement Root：

- `.technical-solution/problem.md`
- `.technical-solution/proposal.md`
- `technical-solution.md`
- `.technical-solution/architecture.mmd`
- `.technical-solution/review.md`

三个 Human Step 的结论与完整 Evidence 由 Flow Event 持久化，不额外维护重复审批文件。

流程状态只由 `.fanloop/flow/state.json` 管理。

## 新增 Workflow

```text
workflows/<workflow-id>/{workflow,condition,flow,loop,prompt}.yaml
skills/<workflow-id>/<skill-id>/SKILL.md
entrypoints/fanloop-workflow/routes.yaml
```

五份 YAML 定义完整执行图，`condition.yaml` 的 `output.description` 可作为 Card 展示名。构建会
拒绝目录不一一对应、跨 Workflow SkillBinding、未知场景目标、缺失 Route 或图不变量错误。

Human Step 选择人工审核路径时，审批 Skill 展示审核材料，Panorama Skill 按当前宿主原样展示
renderer 生成的紧凑全景，并把本次 `snapshot_path` 与人工结论一起上报。选择 `agent_approved`
路径时不展示 Panorama。Flow Runtime 只校验并推进，不自动调用发送工具；Trace provision/sync
与显式 `card render` 保持独立。

选择 `fanloop-maintenance` 场景后，维护 Fanloop 自身时执行：

```bash
fanloop flow init \
  --root /absolute/path/to/maintenance-requirement \
  --workflow fanloop-maintainer \
  --title "Maintain Fanloop"
```

## 验证

```bash
./tests/run-unit
./tests/run-e2e
```

Contract Golden 只在人工确认契约变化后更新：

```bash
go test -count=1 -buildvcs=false ./tests/contracts \
  -run TestPublicContracts -args -update-contracts
```

完整 Requirement E2E 报告保留在 `tests/requirement_e2e/runs/`。

## 发布边界

源码位于私有 GitHub 仓库 `zeefan1555/fanloop`。在 GitHub Actions 手工运行 `Release`
Workflow 会执行完整测试、构建四个平台配套制品、发布 `candidate`、验证后提升 `latest`；
发布使用当前仓库的 `GITHUB_TOKEN`，不需要额外 npm secret。代码目前为 `UNLICENSED`，
选择许可证和公开前资料审计应在首次公开前单独完成。

架构与契约说明见 [CONTEXT.md](./CONTEXT.md)、[docs/technical-design.md](./docs/technical-design.md)
和 [docs/adr/](./docs/adr/)。
