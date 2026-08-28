# Fanloop

Fanloop 是从固定源码快照独立出来的本地工作流运行时。它复用既有的
Flow、Loop、Trace、Card、Release 与 Skill 机制，只使用新的 `fanloop` 身份：

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

## 从源码安装

当前仓库尚未发布 GitHub Release 或 npm 包。请从本地源码构建并安装匹配的 CLI、Workflow
与 Skills（Node.js 18+、Go 1.23+，系统 `tar` 需支持 XZ）：

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

仓库当前只有本地 `main` 分支，不配置远端、不推送。未来创建 GitHub 仓库后再设置
`origin`；`package.json` 中的 GitHub/npm 地址只是目标发布身份。代码目前为
`UNLICENSED`，选择许可证和公开前资料审计应在首次公开前单独完成。

架构与契约说明见 [CONTEXT.md](./CONTEXT.md)、[docs/technical-design.md](./docs/technical-design.md)
和 [docs/adr/](./docs/adr/)。
