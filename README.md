# Fanloop

Fanloop 是由五份 YAML Bundle 驱动的通用本地 Loop 引擎。Go Runtime 不认识任何生产
Workflow、Step、Condition、Output 或原子 Skill ID，只负责严格加载、路由校验、状态转换、
回流失效、审计与投影：

- 命令：`fanloop`
- Requirement 状态：`.fanloop/`
- 用户数据：`~/.fanloop/`
- 环境变量：`FANLOOP_*`
- Go module：`github.com/zeefan1555/fanloop`

Fornax 托管三套生产 Workflow，不设置默认 Workflow：

- `fanloop-maintainer`：Fanloop 自迭代工作流。
- `remnote-flashcards`：从已理解的概念开始，依次定位 HOME Tag、写入 RemNote 闪卡、更新 Wiki 与引用、最终回读校验。
- `pstack-native`：六步剧本型交付，编排 `workflows/pstack-native/skills/` 下原生 pstack skills。

每套业务配置托管在 Fornax，下载到 `current/workflows/<workflow-id>/`，五份 YAML 与 `skills/<skill-id>/` 共享同一个
live 根目录。Workflow 入口独立放在 `entrypoints/fanloop-workflow/`，场景映射由其中的 `routes.yaml` 配置；作者入口由 `.agents/skills/create-fanloop-workflow/` 提供。两个公共 Skill 都随本地 Release 安装；本次外置的是五份Workflow YAML、绑定Skill及资源，未增加动态场景发现。

新增一套流程可先使用仓库内 `.agents/skills/create-fanloop-workflow/` 与
`examples/workflows/example/` 生成并审核完整五 YAML 候选，再增加场景映射；不注册 Go 代码。Trace Registry
的部署与 Workflow 差异位于 `internal/traceconfig/registry.yaml`，同样不进入业务 Runtime。

## 本地构建与安装

从源码使用，需要 Go 1.23+、Git 和 Bash。构建只针对本机，生成只包含 CLI、两个公共 Skill 和
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

本地安装不访问 Fornax，并保留已有 `current/workflows`。全新安装会先创建真实的空
`current/workflows`，此时 `version` 可用；需要首次获取或更新生产配置时，先登录 `fornax-cli`，创建 `~/.fanloop/fornax.json` 明确选择空间和配置包，再显式执行 `fanloop update`：

```json
{"workspace_id":"<workspace-id>","workflows":[{"id":"<workflow-id>","skill_key":"<skill-key>","skill_id":"<skill-id>"}]}
```

```bash
fanloop update
fanloop doctor
```

只有 `fanloop update` 通过 Fornax CLI 查询 Online/prod 已发布版本，再安全下载并原子替换完整配置树；草稿不生效，联网或校验失败时保留旧树。`flow`、`trace`、`card`、`doctor`、`dashboard`、`version` 和本地安装均只读本地字节，不隐式访问 Fornax。
`current/release` 选择不可变CLI；`current/workflows` 是同步运行副本。每套Workflow的五YAML、全部Skill与资源在同一个Fornax Skill包中，配置与代码分别发布。

`FANLOOP_CONFIG_SOURCE` 只用于隔离候选编辑或验收中的明确绝对配置目录，不能指向全局 live 根，也不能作为网络失败 fallback。
代码测试只用 `internal/workflow/testdata/{linear-runtime,runtime-controls}`，并在隔离临时根下组装 `workflows/`；生产发布验证入口为 `scripts/validate-configuration.py --help`。

## 使用

新 Requirement 必须先显式选择场景：

- `fanloop-maintenance` → `fanloop-maintainer`
- `remnote-flashcards` → `remnote-flashcards`
- `pstack-native` → `pstack-native`

`entrypoints/fanloop-workflow/routes.yaml` 没有默认值；未选择或场景未知时停止初始化。
选择维护场景后执行：

```bash
mkdir -p /absolute/path/to/requirement
fanloop flow init \
  --root /absolute/path/to/requirement \
  --workflow fanloop-maintainer \
  --title "Maintain Fanloop"
fanloop flow status --root /absolute/path/to/requirement
```

Agent 使用 `fanloop-workflow` 推进 Requirement，使用 `create-fanloop-workflow` 设计新的 Workflow 候选。运行闭环为：

```text
flow status -> 执行当前 Prompt/Skills -> flow report progress/result -> flow status
```

项目级验证只走 [`.cursor/skills/verify-fanloop/SKILL.md`](./.cursor/skills/verify-fanloop/SKILL.md)。

流程状态只由 `.fanloop/flow/state.json` 管理。

查看已发布 Workflow：

```bash
fanloop dashboard
```

命令在 `127.0.0.1` 的随机空闲端口前台启动并打印 URL。默认页面只读
`~/.fanloop/current/workflows` 中已有的五份 YAML 和 `SKILL.md`。要编辑隔离候选，显式指定绝对目录：

```bash
FANLOOP_CONFIG_SOURCE=/absolute/path/to/candidate fanloop dashboard
```

候选模式只允许保存已列出的完整文件，并校验观察摘要、符号链接、完整 Bundle 和原子替换；页面明确标注候选非生产。保存不会调用 Fornax，生产发布仍需独立 commit/release、`fanloop update` 和字节回读。

## 新增 Workflow

```text
~/.fanloop/current/workflows/<workflow-id>/{workflow,condition,flow,loop,prompt}.yaml
~/.fanloop/current/workflows/<workflow-id>/skills/<skill-id>/SKILL.md
entrypoints/fanloop-workflow/routes.yaml
```

五份 YAML 定义完整执行图，`condition.yaml` 的 `output.description` 可作为 Card 展示名。安装和
Doctor 会拒绝缺失或多余文件、跨 Workflow SkillBinding、符号链接、缺失 Route
或图不变量错误。

Human Step 的审批 Skill 展示审核材料，Panorama Skill 按当前宿主原样展示 renderer 生成的紧凑
全景，并把本次 `snapshot_path` 与审核结论一起上报。`fanloop-maintainer` 的 Define 与 Certify 则由驱动 Flow 的执行子 Agent向主 Agent
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

代码 CI 的 `requirement-e2e` 运行独立fixture的完整 Requirement lifecycle 与 Route Matrix；生产发布用 `scripts/validate-configuration.py` 对明确版本的完整配置集运行矩阵。
本地用户路径只走 [`.cursor/skills/verify-fanloop/SKILL.md`](./.cursor/skills/verify-fanloop/SKILL.md)。
驱动步骤与Feature Map统一在 [verify-fanloop](./.cursor/skills/verify-fanloop/SKILL.md)。
`.cursor/skills/verify-fanloop/features` 是该目录 `references/features/` 的符号链接。
`fanloop-dev-maintain-verification` 按 source 与 live 维护这张地图。产品行为失败作为产品问题报告。

Contract Golden 只在人工确认契约变化后更新：

```bash
go test -count=1 -buildvcs=false ./tests/contracts \
  -run TestPublicContracts -args -update-contracts
```

完整 Requirement E2E 报告保留在 `tests/requirement_e2e/runs/`。

## 源码边界

源码位于 GitHub 仓库 `zeefan1555/fanloop`。GitHub 负责代码托管、PR 与 CI，不发布 npm、平台归档或其他安装制品。合并后从精确 merge commit 执行本地构建和安装；Workflow 仍由 Fornax 独立发布并通过显式 `fanloop update` 同步。

构建与安装边界见 [ADR-0095](./docs/adr/0095-local-source-builds.md)，live Workflow 与 Dashboard 边界见
[ADR-0120](./docs/adr/0120-host-production-workflows-in-fornax.md)，本地分发和显式更新见
[ADR-0122](./docs/adr/0122-use-explicit-fornax-update-and-local-source-distribution.md)。代码授权仍为 `UNLICENSED`。

架构与契约说明见 [CONTEXT.md](./CONTEXT.md)、[docs/technical-design.md](./docs/technical-design.md)
和 [docs/adr/](./docs/adr/)。
