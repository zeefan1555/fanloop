# Fanloop

Fanloop 是从固定源码快照独立出来的本地工作流运行时。它复用既有的
Flow、Loop、Trace、Card、Release 与 Skill 机制，只使用新的 `fanloop` 身份：

- 命令：`fanloop`
- Requirement 状态：`.fanloop/`
- 用户数据：`~/.fanloop/`
- 环境变量：`FANLOOP_*`
- Go module：`github.com/zeefan1555/fanloop`

当前只发布两套五文件 Workflow Bundle：

- `promotion-design`：默认工作流，完成需求澄清、需求确认、晋升方案写作、陌生评委审校和最终确认。
- `fanloop-maintainer`：Fanloop 自迭代工作流。

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

新建一个 Promotion Requirement：

```bash
mkdir -p /absolute/path/to/requirement
fanloop flow init \
  --root /absolute/path/to/requirement \
  --workflow promotion-design \
  --title "My promotion design"
fanloop flow status --root /absolute/path/to/requirement
```

Agent 的默认入口是 `fanloop-workflow` Skill。它按以下闭环推进：

```text
flow status -> 执行当前 Prompt/Skills -> flow report progress/result -> flow status
```

`promotion-design` 的领域产物写在 Requirement Root：

- `require_points.md`
- `方案.md`
- `.promotion/brainstorm.md`
- `evidence.md`
- `.promotion/review.md`

流程状态只由 `.fanloop/flow/state.json` 管理；不会创建 `.promotion/state.json`。

维护 Fanloop 自身时显式选择：

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
