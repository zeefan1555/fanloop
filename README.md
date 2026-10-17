# Fanloop

Fanloop 是由五份 YAML Bundle 驱动的通用本地 Loop 引擎。Go Runtime 不认识任何生产
Workflow、Step、Condition、Output 或原子 Skill ID，只负责严格加载、路由校验、状态转换、
回流失效、审计与投影：

- 命令：`fanloop`
- Requirement 状态：`.fanloop/`
- 用户数据：`~/.fanloop/`
- 环境变量：`FANLOOP_*`
- Go module：`github.com/zeefan1555/fanloop`

配置仓库 `zeefan1555/second-brain-skills` 的 `main` 托管生产 Workflow，不设置默认 Workflow：

- `pstack-native`：六步剧本交付，完整包来自配置中心。
- `code-change`、`write-skill`、`daily-journal`、`general`：独立的 8 节点 Workflow（由 second-brain-skills 的 `select-workflow` Skill 选择）；Fanloop、RemNote Direct 等仓库的代码变更走 `code-change`（`fanloop-maintainer`、`remnote-direct-maintainer` 已在 ADR-0135 退役）。
- `ai-daily-brief`：采集公开 X 原创帖并生成中文 AI 日报。
- `second-brain-skills-maintainer`：维护 Skills 仓库候选、验证与交付。
- `remnote-flashcards`：从已理解的概念开始，依次定位 HOME Tag、写入 RemNote 闪卡、更新 Wiki 与引用、最终回读校验。复盘若有一条可复用发现，用官方 Notion CLI `ntn login` 写入已有数据库「remnote workflow 经验表」；没有钥匙串时 `NOTION_KEYRING=0`。字段与默认 ID 见 `fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/notion-experience.md`。

每套 Workflow 是仓库 `fanloop-workflow/<workflow-id>/` 下的完整 Skill 包：根 `SKILL.md`、五份 YAML 与子 `skills/<skill-id>/**` 同目录保存。Workflow 入口独立放在 `entrypoints/fanloop-workflow/`，场景映射由其中的 `routes.yaml` 配置；作者入口由 `.agents/skills/create-fanloop-workflow/` 提供。两个公共 Skill 都随本地 Release 安装；Workflow YAML、绑定 Skill 及资源直接来自源仓库，未增加动态场景发现。

新增一套流程可先使用仓库内 `.agents/skills/create-fanloop-workflow/` 与
`examples/workflows/example/` 生成并校验完整五 YAML 候选；明确的包入口可直接使用 Workflow ID。
`examples/workflows/manage-fanloop-workflow` 是可加载的四阶段配置管理包，发布目标是个人 Fornax 空间，不进入 second-brain-skills 生产库存。
公共场景别名属于独立入口 Skill 的安装范围，不为新增配置隐式重建 CLI 或更新已安装场景表。不注册 Go 代码。Trace Registry
的部署与 Workflow 差异位于 `internal/traceconfig/registry.yaml`，同样不进入业务 Runtime。该文件里的 Lark 坐标是占位符；provider 为 lark 时用 `FANLOOP_TRACE_REGISTRY_URL`、`FANLOOP_TRACE_REGISTRY_BASE_TOKEN`、`FANLOOP_TRACE_REGISTRY_TABLE_ID`、`FANLOOP_TRACE_REGISTRY_VIEW_ID` 提供真实值。

## 安装发布版本

发布由推送形如 `vX.Y.Z` 的 tag 触发。GitHub Actions 在 Ubuntu 上交叉编译 macOS 与 Linux 的 amd64、arm64，并把平台归档、`checksums.txt`、`release.json` 和薄 npm 包装上传到该 GitHub Release。不使用 npm registry。

```bash
npm i -g --foreground-scripts --allow-scripts=fanloop https://github.com/zeefan1555/fanloop/releases/download/vX.Y.Z/fanloop-X.Y.Z.tgz
fanloop --version
```

较新的 npm 可能拦截 postinstall，需要 `--allow-scripts=fanloop`。npm 默认不显示成功的 postinstall 输出，`--foreground-scripts` 让 `export PATH=...` 出现在终端里。安装程序先校验 sha256 和完整的内部 release 清单，再解压到 `~/.fanloop/releases/<version>`，创建指向 `current` 的公共 Skill 链接，最后原子切换 `~/.fanloop/current`。包装里的启动器会执行 `current/bin/fanloop`。如果这个命令还不在 `PATH` 上，安装程序会打印一条可以直接执行的 `export PATH=...`；之后 `fanloop doctor` 也会在 hint 里给出同一条命令。

更新 CLI 与更新 Workflow 配置是两条命令：

```bash
fanloop update
fanloop update config
```

`fanloop update` 下载最新 GitHub Release 并切换 CLI 的 `current`，然后从 `second-brain-skills` 的 `main` 同步 Workflow 配置。本机版本比 `latest` 新时 CLI 保持不变，不会降级；同一版本会按已校验归档修复被改过的发布内容。`fanloop update config` 只做这次 Git 同步。`git ls-remote` 与当前快照 commit 一致时跳过 checkout，并发同步共用一把锁，并打印完整 commit 和每个 Workflow id。缺少 `fornax.json` 不是错误。`FANLOOP_USE_FORNAX=1` 且 `fornax.json` 是普通文件时，才改为同步 Fornax 上已发布的包并打印 `id@version`。

## 本地构建与安装

从源码使用，需要 Go 1.23 或更高版本（`go.mod` 要求 `go 1.23.0`）、Git 和 Bash。构建只针对本机，生成只包含 CLI、两个公共 Skill 和
校验清单的不可变 Release；Workflow YAML 与其绑定 Skills 不进入 Release。
根级 `VERSION` 是 CLI 基础版本真值；本地源码构建使用 `VERSION-dev.<提交或内容摘要>`，同一提交
可幂等复用，不同提交不会覆盖同名不可变 Release。Workflow/Skill 与范文改动不改变未提交 CLI
构建的内容摘要。

安装了 Node.js 与 npm 后，可在仓库根目录使用以下本地快捷入口；无需先运行 `npm install`：

```bash
npm run build          # 只构建，不改变本机安装
npm run install:local  # 构建并安装；更新源码后重跑即可更新 CLI
```

这两个命令不打包或发布 npm；它们只是 Shell 脚本的快捷方式，仍需要上述 Go、Git 和 Bash。需要捕获构建目录时直接调用 Shell 入口：

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
export PATH="$HOME/.fanloop/current/bin:$PATH"
```

安装保持离线。`./scripts/install-local.sh --config-source /absolute/config-repository` 可从干净 Git 提交导入配置快照；不指定时保留已有快照。首次安装无配置也可运行 version，随后 flow init 自动拉取。

`flow init` 通过现有 SSH 凭据从 `git@github.com:zeefan1555/second-brain-skills.git` 的 main 拉取并验证完整集合，原子切换 `config/current` 到 `config/snapshots/<commit-sha>/`。失败保留旧配置且不创建 Requirement；dry-run 只验证临时候选。`configuration.json` 与 Doctor 给出 commit SHA 和完整资源摘要。已有 status/report/skills 保持离线。

显式绝对 `FANLOOP_CONFIG_SOURCE` 指向包含 `fanloop-workflow/` 的作者仓库根，并跳过远端刷新。默认 Dashboard 只读快照；显式作者源可编辑，下一次调用读取编辑内容。快照不原地编辑。

`fanloop config drift` 默认把 `config/current` 与 `second-brain-skills` `main` 比较。快照文件与 manifest sha256 一致且 commit 相同是 `in_sync`，commit 不同是 `local_behind`，文件字节不同是 `local_modified`，`ls-remote` 失败是 `unavailable`。来自 Fornax 的快照是 `local_behind`，提示运行 `fanloop update config`。只有 `FANLOOP_USE_FORNAX=1` 且 `fornax.json` 是普通文件时，才使用已经登录的 `fornax-cli`，以每个 source 的最新 Online/prod 发布包为参照。该配置只保存 workspace、key prefix 和 access，不保存凭证、Skill ID 或版本；Runtime 和安装器不创建它。见 [ADR-0139](./docs/adr/0139-update-config-from-github.md)。

```bash
fanloop config drift
```

代码测试使用 `internal/workflow/testdata/{linear-runtime,runtime-controls}`，在隔离源的 `fanloop-workflow/` 下组装完整包。生产包验证入口为 `scripts/validate-configuration.py --help`。

## 使用

新 Requirement 必须先显式选择场景：

- `fanloop-maintenance` → `code-change`
- `remnote-flashcards` → `remnote-flashcards`

`entrypoints/fanloop-workflow/routes.yaml` 没有默认值；未选择或场景未知时停止初始化。
RemNote Direct 的仓库维护同样走 `code-change` Workflow。
选择维护场景后执行：

```bash
mkdir -p /absolute/path/to/requirement
fanloop flow init \
  --root /absolute/path/to/requirement \
  --workflow code-change \
  --title "Maintain Fanloop"
fanloop flow status --root /absolute/path/to/requirement
```

Agent 使用 `fanloop-workflow` 推进 Requirement，使用 `create-fanloop-workflow` 设计新的 Workflow 候选。运行闭环为：

```text
flow status -> 执行当前 Prompt/Skills -> flow report progress/result -> flow status
```

项目级验证只走 [`.cursor/skills/verify-fanloop/SKILL.md`](./.cursor/skills/verify-fanloop/SKILL.md)。

流程状态只由 `.fanloop/flow/state.json` 管理。

查看本地 Workflow：

```bash
fanloop dashboard
```

命令在 `127.0.0.1` 的随机空闲端口前台启动并打印 URL。默认只读绑定仓库中的完整包。要编辑本地源，显式指定绝对仓库目录：

```bash
FANLOOP_CONFIG_SOURCE=/absolute/path/to/repository fanloop dashboard
```

显式编辑模式只保存已列出的完整文件，包括根 `SKILL.md`，校验观察摘要、符号链接与完整 Bundle 后原子写回同一仓库。可以显式编辑当前绑定源；CLI 和 Dashboard 共享同源目录锁。

## 新增 Workflow

```text
fanloop-workflow/<workflow-id>/SKILL.md
fanloop-workflow/<workflow-id>/{workflow,condition,flow,loop,prompt}.yaml
fanloop-workflow/<workflow-id>/skills/<skill-id>/**
entrypoints/fanloop-workflow/routes.yaml
```

五份 YAML 定义完整执行图，`condition.yaml` 的 `output.description` 可作为 Card 展示名。安装和
Doctor 会拒绝缺失或多余文件、跨 Workflow SkillBinding、符号链接、缺失 Route
或图不变量错误。

Human Step 的审批 Skill 展示审核材料，Panorama Skill 按当前宿主原样展示 renderer 生成的紧凑
全景，并把本次 `snapshot_path` 与审核结论一起上报。代码变更的需求批准与最终认证则由驱动 Flow 的执行子 Agent向主 Agent
提交需求契约或独立认证证据并等待决定。Flow Runtime 只校验并推进，不自动调用发送工具；Trace provision/sync 与显式 `card render` 保持独立。

选择 `fanloop-maintenance` 场景后，维护 Fanloop 自身时执行：

```bash
fanloop flow init \
  --root /absolute/path/to/maintenance-requirement \
  --workflow code-change \
  --title "Maintain Fanloop"
```

## 验证

```bash
./tests/run-unit
```

代码 CI 的 `requirement-e2e` 运行独立fixture的完整 Requirement lifecycle 与 Route Matrix；生产发布用 `scripts/validate-configuration.py` 对明确版本的完整配置集运行矩阵。
本地用户路径只走 [`.cursor/skills/verify-fanloop/SKILL.md`](./.cursor/skills/verify-fanloop/SKILL.md)。
驱动步骤与Feature Map统一在 [verify-fanloop](./.cursor/skills/verify-fanloop/SKILL.md)。
`.cursor/skills/verify-fanloop/features` 是代码拥有的真实目录。
`fanloop-dev-maintain-verification` 按 source 与 live 维护这张地图。产品行为失败作为产品问题报告。

Contract Golden 只在人工确认契约变化后更新：

```bash
go test -count=1 -buildvcs=false ./tests/contracts \
  -run TestPublicContracts -args -update-contracts
```

完整 Requirement E2E 报告保留在 `tests/requirement_e2e/runs/`。

## 源码边界

源码位于 GitHub 仓库 `zeefan1555/fanloop`。用户安装的 CLI 来自 GitHub Release；仓库不向 npm registry 发布。开发机仍可从精确 merge commit 执行 `./scripts/install-local.sh`。生产 Workflow 完整包在 `zeefan1555/second-brain-skills` 独立维护，配置更新不要求重建 CLI。

发布与安装边界见 [ADR-0136](./docs/adr/0136-distribute-cli-from-github-releases.md)。本地开发构建见 [ADR-0095](./docs/adr/0095-local-source-builds.md)，Workflow Git 真值、源绑定与 Dashboard 边界见
[ADR-0132](./docs/adr/0132-git-workflow-configuration-center.md)，个人空间镜像见 [ADR-0136](./docs/adr/0136-mirror-git-workflows-to-fornax.md)。代码授权仍为 `UNLICENSED`。

架构与契约说明见 [CONTEXT.md](./CONTEXT.md)、[docs/technical-design.md](./docs/technical-design.md)
和 [docs/adr/](./docs/adr/)。

Use `fanloop flow skills --root <ABSOLUTE_REQUIREMENT_ROOT>` to locate every Skill in the current Workflow package, including unbound Skills. The catalog guides optional discovery; current Step requirements remain in `flow status`.
