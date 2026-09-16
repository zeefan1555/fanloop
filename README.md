# Fanloop

Fanloop 是由五份 YAML Bundle 驱动的通用本地 Loop 引擎。Go Runtime 不认识任何生产
Workflow、Step、Condition、Output 或原子 Skill ID，只负责严格加载、路由校验、状态转换、
回流失效、审计与投影：

- 命令：`fanloop`
- Requirement 状态：`.fanloop/`
- 用户数据：`~/.fanloop/`
- 环境变量：`FANLOOP_*`
- Go module：`github.com/zeefan1555/fanloop`

当前携带两套五文件 Workflow Bundle，不设置默认 Workflow：

- `technical-solution-design`：完成问题定义、方向推导、正式方案写作、独立审校和三级人工确认。
- `fanloop-maintainer`：Fanloop 自迭代工作流。

业务配置严格按 `workflows/<workflow-id>/ ↔ skills/<workflow-id>/` 一一对应，不设例外。
统一入口独立放在 `entrypoints/fanloop-workflow/`，场景映射由其中的 `routes.yaml` 配置。

新增一套流程只需要增加同名 Workflow/Skill 目录和一条场景映射；不注册 Go 代码。Trace Registry
的部署与 Workflow 差异位于 `internal/traceconfig/registry.yaml`，同样不进入业务 Runtime。

## 本地构建与安装

从源码使用，需要 Go 1.23+、Git 和 Bash。构建只针对本机，生成包含 CLI、Workflow、Skills、
范文文本和校验清单的可运行目录；范文图片仅保留在源码仓库的 `exemplars/`，不进入构建目录。
根级 `VERSION` 是版本真值；干净提交显示该 SemVer，未提交源码构建显示对应的 `-dev.<摘要>` 预发布版本。

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

安装先校验二进制及配套内容，通过 Doctor 后原子切换 `~/.fanloop/current`。更新时在选定源码
提交上重新执行 `./scripts/install-local.sh`；已有 Requirement 继续使用与其绑定内容匹配的本地版本。

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

## 新增 Workflow

```text
workflows/<workflow-id>/{workflow,condition,flow,loop,prompt}.yaml
skills/<workflow-id>/<skill-id>/SKILL.md
entrypoints/fanloop-workflow/routes.yaml
```

五份 YAML 定义完整执行图，`condition.yaml` 的 `output.description` 可作为 Card 展示名。构建会
拒绝目录不一一对应、跨 Workflow SkillBinding、未知场景目标、缺失 Route 或图不变量错误。

Human Step 的审批 Skill 展示审核材料，Panorama Skill 按当前宿主原样展示 renderer 生成的紧凑
全景，并把本次 `snapshot_path` 与审核结论一起上报。`technical-solution-design` 的三处审核还要求
已回读的飞书文档 URL 和人的明确决定；仅声明 `agent_approved` Route 的 Workflow 可由 Agent 独立
批准且不展示 Panorama。Flow Runtime 只校验并推进，不自动调用发送工具；Trace provision/sync
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

## 源码边界

源码位于私有 GitHub 仓库 `zeefan1555/fanloop`。现有 CI 验证代码和本地安装，使用者自行从源码
构建；不发布 npm 包、跨平台归档或 GitHub 二进制制品。构建与安装边界见
[ADR-0095](./docs/adr/0095-local-source-builds.md)。代码授权仍为 `UNLICENSED`。

架构与契约说明见 [CONTEXT.md](./CONTEXT.md)、[docs/technical-design.md](./docs/technical-design.md)
和 [docs/adr/](./docs/adr/)。
