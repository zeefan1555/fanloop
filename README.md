# Fanloop

Fanloop 是由五份 YAML Bundle 驱动的通用本地 Loop 引擎。Go Runtime 不认识任何生产
Workflow、Step、Condition、Output 或原子 Skill ID，只负责严格加载、路由校验、状态转换、
回流失效、审计与投影：

- 命令：`fanloop`
- Requirement 状态：`.fanloop/`
- 用户数据：`~/.fanloop/`
- 环境变量：`FANLOOP_*`
- Go module：`github.com/zeefan1555/fanloop`

源码提供四套可用于首次安装的 Workflow，不设置默认 Workflow：

- `technical-solution-design`：完成问题定义、方向推导、正式方案写作、独立审校和三级人工确认。
- `fanloop-maintainer`：Fanloop 自迭代工作流。
- `remnote-flashcards`：从已理解的概念开始，依次定位 HOME Tag、写入 RemNote 闪卡、更新 Wiki 与引用、最终回读校验。
- `pstack-delivery`：原样绑定并编排原生 pstack skills 的交付流程。

每套业务配置都位于 `workflows/<workflow-id>/`，五份 YAML 与 `skills/<skill-id>/` 共享同一个
live 根目录。统一入口独立放在 `entrypoints/fanloop-workflow/`，场景映射由其中的 `routes.yaml` 配置。

新增一套流程只需要增加一个完整 Workflow 目录和一条场景映射；不注册 Go 代码。Trace Registry
的部署与 Workflow 差异位于 `internal/traceconfig/registry.yaml`，同样不进入业务 Runtime。

## 本地构建与安装

从源码使用，需要 Go 1.23+、Git 和 Bash。构建只针对本机，生成只包含 CLI、统一入口和
校验清单的不可变 Release；Workflow YAML 与其绑定 Skills 不进入 Release。
根级 `VERSION` 是 CLI 基础版本真值；本地源码构建使用 `VERSION-dev.<提交或内容摘要>`，同一提交
可幂等复用，不同提交不会覆盖同名不可变 Release。Workflow/Skill 与范文改动不改变未提交 CLI
构建的内容摘要。

只构建、直接运行：

```bash
fanloop_build_dir="$(./scripts/build-local.sh)"
"$fanloop_build_dir/bin/fanloop" version
```

构建脚本的标准输出只有目录路径，默认位于 `dist/local-*`。可传入一个尚不存在的输出目录：
`./scripts/build-local.sh /absolute/path/to/new-build`。

构建并安装到本机：

```bash
./scripts/install-local.sh
"$HOME/.fanloop/current/bin/fanloop" version
"$HOME/.fanloop/current/bin/fanloop" doctor
export PATH="$HOME/.fanloop/current/bin:$PATH"
```

安装先校验二进制、统一入口和配置源中的完整 Workflow 树。`~/.fanloop/current` 是稳定目录：
`current/release` 原子选择不可变 CLI Release，`current/workflows` 是真实、可写、不会被后续升级覆盖的
live 配置。首次安装从 `FANLOOP_CONFIG_SOURCE`（默认当前源码仓库）验证并播种该目录；之后 Fanloop
每次调用都从这里重新加载 Workflow YAML 与绑定 Skill，因此 Dashboard 或外部编辑会在下一次命令生效。

## 使用

新 Requirement 必须先显式选择场景：

- `technical-solution` → `technical-solution-design`
- `fanloop-maintenance` → `fanloop-maintainer`
- `remnote-flashcards` → `remnote-flashcards`
- `pstack-delivery` → `pstack-delivery`

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

项目级验证只走 [`.cursor/skills/verify-fanloop/SKILL.md`](./.cursor/skills/verify-fanloop/SKILL.md)。

`technical-solution-design` 的十六个 Step 按业务问题、技术判断和结果与规划三阶段推进。文章第 0 至
第 10 章各有一个独立 Agent Step 和 Markdown 产物，另保留三个 Human Gate、文档组装与独立审校。
主要产物写在 Requirement Root：

- `.technical-solution/sections/00-summary.md`
- `.technical-solution/sections/01-business-background.md` 至 `10-retrospective-and-roadmap.md`
- `technical-solution.md`
- `.technical-solution/architecture.mmd`
- `.technical-solution/review.md`

三个 Human Step 的结论与完整 Evidence 由 Flow Event 持久化，不额外维护重复审批文件。

流程状态只由 `.fanloop/flow/state.json` 管理。

本地编辑 live Workflow：

```bash
fanloop dashboard
```

命令在 `127.0.0.1` 的随机空闲端口前台启动并打印 URL。页面只列出
`~/.fanloop/current/workflows` 中已有的五份 YAML 和 `SKILL.md`，保存时执行路径白名单、SHA-256
并发冲突、Workflow 校验与原子替换；不提供远程访问、创建、删除或重命名。

## 新增 Workflow

```text
workflows/<workflow-id>/{workflow,condition,flow,loop,prompt}.yaml
workflows/<workflow-id>/skills/<skill-id>/SKILL.md
entrypoints/fanloop-workflow/routes.yaml
```

五份 YAML 定义完整执行图，`condition.yaml` 的 `output.description` 可作为 Card 展示名。安装和
Doctor 会拒绝缺失或多余文件、跨 Workflow SkillBinding、未知场景目标、符号链接、缺失 Route
或图不变量错误。

Human Step 的审批 Skill 展示审核材料，Panorama Skill 按当前宿主原样展示 renderer 生成的紧凑
全景，并把本次 `snapshot_path` 与审核结论一起上报。`technical-solution-design` 的三处审核要求展示后的
全新、明确 human 决定；`fanloop-maintainer` 的 Define 与 Certify 则由驱动 Flow 的执行子 Agent向主 Agent
提交需求契约或独立认证证据并等待决定。Flow Runtime 只校验并推进，不自动调用发送工具；Trace provision/sync 与显式 `card render` 保持独立。

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
```

完整 Requirement lifecycle 与全部生产 Workflow Route Matrix 由 CI 的 `requirement-e2e` check 运行。
本地用户路径只走 [`.cursor/skills/verify-fanloop/SKILL.md`](./.cursor/skills/verify-fanloop/SKILL.md)。
驱动步骤在 [`workflows/fanloop-maintainer/skills/fanloop-dev-verify/`](./workflows/fanloop-maintainer/skills/fanloop-dev-verify/)。
`.cursor/skills/verify-fanloop/features` 是该目录 `references/features/` 的符号链接。
`fanloop-dev-maintain-verification` 按 source 与 live 维护这张地图。产品行为失败作为产品问题报告。

Contract Golden 只在人工确认契约变化后更新：

```bash
go test -count=1 -buildvcs=false ./tests/contracts \
  -run TestPublicContracts -args -update-contracts
```

完整 Requirement E2E 报告保留在 `tests/requirement_e2e/runs/`。

## 源码边界

源码位于私有 GitHub 仓库 `zeefan1555/fanloop`。现有 CI 验证代码和本地安装，使用者自行从源码
构建；不发布 npm 包、跨平台归档或 GitHub 二进制制品。构建与安装边界见
[ADR-0095](./docs/adr/0095-local-source-builds.md)，live Workflow 与 Dashboard 边界见
[ADR-0108](./docs/adr/0108-use-live-workflows-and-local-dashboard.md)。代码授权仍为 `UNLICENSED`。

架构与契约说明见 [CONTEXT.md](./CONTEXT.md)、[docs/technical-design.md](./docs/technical-design.md)
和 [docs/adr/](./docs/adr/)。
