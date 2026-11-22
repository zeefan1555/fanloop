# Fanloop 当前技术设计

Fanloop 的产品定位是通用 Loop 引擎：执行图来自配置，Go 代码只提供稳定的解释器与持久化能力。

## 真值与边界

- `idl/*.thrift` 定义公开 CLI、durable storage 与 Workflow YAML 结构；`internal/idl/` 只保存生成物。
- 每套 Workflow 由 `workflow.yaml`、`condition.yaml`、`flow.yaml`、`loop.yaml`、`prompt.yaml`
  五个文件共同定义，Go Runtime 只负责通用加载、校验与执行。
- Agent 提交事实和 RouteSelection；CLI 校验结构、输出类型、互斥条件与唯一 Route，不证明业务事实。
- Flow State 与 Output Registry 是本地事实，Trace 与 Card 是相互隔离的投影。
- `cmd/`、`internal/` 与 `scripts/` 的生产代码不得出现生产 Workflow、Step、Condition、Output 或
  原子 Skill ID；测试会从当前 Bundle 动态提取并扫描这些事实。

当前生产产品流程不再使用 Workflow。`second-brain-agent/plugins/` 中的 Codex 插件直接执行账户早报、TradingView 回测、日记和 RemNote 等产品能力；原 `second-brain-skills/fanloop-workflow/` 目录及全部生产包已退役，见 [ADR-0148](./adr/0148-retire-production-workflows-for-codex-plugins.md)。

本地 Release 只暴露 `fanloop-workflow` 兼容入口；`routes.yaml` 使用 `schema_version: 2`，`scenarios` 为空。禁止新 `flow init`、恢复已退役包或把插件包装成 Workflow。已有 Requirement 的 status、report、skills、trace、card render、Card projection 和 Doctor 需求检查读取各自 `.fanloop/configuration-snapshot`，因此 pinned 历史 Run 仍可只读或继续。

Trace Registry 是独立的部署配置：`internal/traceconfig/registry.yaml` 按 profile 提供默认策略和
可选 Workflow 覆盖。Runtime 只通过 `Resolve(profile, workflowID)` 取得 provider、endpoint、远端字段名、
CLI 日志要求和 Output 字段映射。完整通用化决策见
[ADR-0080](./adr/0080-make-runtime-workflow-agnostic.md)。`provider: none` 时 `flow init` 不建 Trace 文档，
Registry 投影跳过，见 [ADR-0133](./adr/0133-trace-registry-provider-switch.md)。production profile 现已全部为
`provider: none`（飞书投影关闭，运行记录改由 Notion run-ledger 承担），test profile 保留占位 Lark 表，见
[ADR-0134](./adr/0134-turn-off-lark-trace-projection.md)。provider 为 lark 时，`FANLOOP_TRACE_REGISTRY_URL`、`FANLOOP_TRACE_REGISTRY_BASE_TOKEN`、`FANLOOP_TRACE_REGISTRY_TABLE_ID` 与 `FANLOOP_TRACE_REGISTRY_VIEW_ID` 覆盖仓库中的占位坐标。

## 推进模型

`flow status` 返回当前 Step 的 context、execution、Prompt/Skills、Conditions、available routes、
Workflow 级 common Skills/Conditions、skipped Steps 和有效 Outputs。Agent 只从该响应选择一组
Condition 与一条 Route，再调用：

```text
fanloop flow report progress
fanloop flow report result
```

`when.any_of` 外层为 OR、内层为 AND。Flow 前进到 `next_step_id` 或 terminal；Loop 回到
`back_step_id`。Loop 只失效本次 Result 提交前已经存在、且由目标 Step 或其下游产生的 Outputs；本次
Result 新接受的反馈或交接 Outputs 在失效完成后写入并保留。写命令在同一 Requirement lock 下提交
State、Output Registry 与 Event；dry-run 只计算响应，不落盘。

声明 `step_start` 与 `jump` 的 Workflow 在每次进入 Step 时先进入 `awaiting_confirmation`。此时只展示
开工确认 Prompt、start Route 和面向全部真实 Step 的 jump Routes；人的明确确认形成独立 Event 后，
同一 Step 才进入 `in_progress` 并展示业务 Prompt。Jump 可在任意运行中 Step 使用，向前跨过的 Step
进入 `skipped_step_ids`，目标及下游 Outputs 失效；Card 与 Trace 将 skipped 显示为“已跳过”。完整
决策见 [ADR-0102](./adr/0102-add-common-step-controls.md)。

历史 pinned Workflow 中，Human Step 的审核与可选 Panorama 仍由该快照的五份 YAML 驱动。选择 Panorama 的 Workflow 通过普通 Condition 和绑定 Skill 按宿主原样展示 renderer 输出并返回本次快照；CLI 只校验类型与 Route，不代替审批或证明业务事实，也不注入固定 Panorama 节点。当前生产配置没有 Workflow。

## 当前配置实例

生产实例为零。通用 Runtime 的测试仍使用 `examples/workflows/` 和 `internal/workflow/testdata/` 中的合成 Bundle；它们不是可恢复的产品入口。既有 Requirement 的 pinned Bundle 是历史事实，不由本仓库升级、重写或迁移。

## 当前持久化版本

- Workflow / Flow / Condition / Loop / Prompt：`7 / 5 / 3 / 4 / 2`
- Flow State / Event / Output Registry：`16 / 16 / 4`
- Card Projection / Card Binding / Trace Config / CLI Log：`7 / 2 / 2 / 2`

Requirement 文件集中在 `.fanloop/{flow,output,trace,card,log}`；公开命令、文件位置和恢复提示均不
提供旧产品身份的兼容入口。

## 本地构建与验证

`./scripts/build-local.sh [OUTPUT_DIR]` 只编译本机二进制，生成携带 `bin/fanloop`、一个 `fanloop-workflow` 兼容 Skill 和
Manifest 的本地 Release；标准输出是构建目录的绝对路径。`skills/` 与 `exemplars/` 不进入 Release，
也不参与 CLI 的 dirty 版本摘要。显式输出目录必须尚不存在，默认在 `dist/` 下新建唯一
`local-*` 目录。

Release Manifest schema 4 以 `cli.binary_sha256` 校验本机二进制，同时固定版本、Storage Schema 支持与
两个 Skill 的 SHA-256；没有 Workflow artifact 或平台归档。`./scripts/install-local.sh [OUTPUT_DIR]` 构建后
复用目录安装，离线切换 CLI Release；显式配置源导入干净提交快照，否则保留已有配置。

`flow init` 默认在稳定锁内用 HTTPS 拉取 `https://github.com/zeefan1555/second-brain-skills.git` 的 `main`，即使存在 `fornax.json`。HTTPS 尝试为 15 秒且不提示输入凭据；失败时同一次操作回退一次 SSH 地址。`configuration.json` 记录实际成功的地址。已有快照按 commit 复用。见 [ADR-0141](./adr/0141-https-config-remote-with-ssh-fallback.md)。Git 或网络失败不替换 `config/current`；已有快照时继续使用它，没有快照时初始化失败。`FANLOOP_USE_FORNAX=1` 才让 init 使用 Fornax 快照，目录名是排序后 `key@version` 行的 sha256。显式 `FANLOOP_CONFIG_SOURCE` 指向作者仓库根并跳过刷新。完整包目录是 `fanloop-workflow/<id>`。配置校验失败不创建 Requirement，也不替换 current。新 Requirement 把快照目录写到 `.fanloop/configuration-snapshot`。

`fanloop update` 切换到另一个 GitHub Release 后，由新二进制执行 `update config`。同一版本修复仍在当前进程同步。`fanloop update config` 不重新执行二进制。npm 安装在切换 `current` 之后运行同一次配置同步，失败只打印警告。二者默认同步 `second-brain-skills` `main`。commit 未变则跳过 checkout，否则发布 `config/snapshots/<sha>` 并激活。见 [ADR-0140](./adr/0140-update-reexecs-new-release-for-config-sync.md)。`fanloop config drift` 默认比较该快照与 `main`：内容与 commit 都一致是 `in_sync`，commit 不同是 `local_behind`，文件与 manifest sha256 不同是 `local_modified`，`ls-remote` 失败是 `unavailable`。只有 `FANLOOP_USE_FORNAX=1` 且 `fornax.json` 是普通文件时，才通过用户已登录的 `fornax-cli` 安装或比较 Online/prod 发布。个人 `read_write` 源在 workflow id 冲突时胜出。程序不内置 AK/SK。见 [ADR-0139](./adr/0139-update-config-from-github.md)。Doctor 对 Git 快照打印 `commit_sha`，对 Fornax 快照打印 `release_set_digest`。

`fanloop dashboard` 提供 loopback-only 文件与拓扑，保持源码锚点、缩放/聚焦/选择与轮询。默认绑定源只读；显式绝对 `FANLOOP_CONFIG_SOURCE` 才开启该仓库保存。写入只接受 allowlist 内完整文件，校验观察 SHA-256、无符号链接和整个候选 Bundle 后原子替换；可编辑当前绑定仓库，不调用远端配置服务。
五YAML仍是通用加载器的唯一执行图，配置内容不进入Requirement或CLI Release。
本地源绑定、完整包与 Dashboard 边界见 [ADR-0132](./adr/0132-git-workflow-configuration-center.md)。

用户安装的 CLI 来自 GitHub Release。形如 `vX.Y.Z` 的 tag 触发 Actions，只在 ubuntu 上交叉编译 darwin/linux amd64/arm64，上传平台 tar.gz、`checksums.txt`、分发清单 `release.json` 和薄 npm 包装。安装命令带 `--foreground-scripts --allow-scripts=fanloop`。安装先完成校验和公共 Skill 链接，再切换 `current`。`fanloop update` 下载最新 Release 并原子切换 CLI 的 `~/.fanloop/current`，但不会把较新的本机版本降级到 `latest`。版本发生变化时，新二进制执行配置同步；同一版本修复仍由当前进程同步 `second-brain-skills` `main`。`fanloop update config` 只做这次 Git 同步；commit 未变时跳过 checkout。`FANLOOP_USE_FORNAX=1` 且存在 `fornax.json` 时才同步 Fornax，已发布集合和已安装树都没变时跳过下载。平台包内部的 `release.json` 仍是 schema 4，不含平台归档字段。开发构建继续使用 `./scripts/build-local.sh` 与 `./scripts/install-local.sh`。已有 Requirement 的 `flow status/report/skills`、`trace`、`card render`、Card projection 更新和 Doctor 需求检查读取该 Requirement 固定的快照，不跟随之后的 `config/current`。投影文件能通过 pinned Workflow 校验但落后于已提交 State/Event 时，非 dry-run `card render` 从事件日志重建投影后再渲染；缺失、损坏或绑定错误的投影仍直接失败。见 [ADR-0142](./adr/0142-rebuild-stale-card-projection-from-pinned-snapshot.md)。完整决定见
[ADR-0140](./adr/0140-update-reexecs-new-release-for-config-sync.md)、[ADR-0137](./adr/0137-take-workflow-configuration-from-fornax.md)、[ADR-0136](./adr/0136-distribute-cli-from-github-releases.md) 与 [ADR-0132](./adr/0132-git-workflow-configuration-center.md)。

新增日常产品流程时，在 `second-brain-agent/plugins/` 维护 Codex 插件，不新增生产 Workflow、公共场景或作者入口。Fanloop Runtime 只继续支持通用执行器测试和已有 pinned Requirement。

仓库级本地门禁只有一个：

```bash
./tests/run-unit
```

它覆盖格式、IDL 新鲜度、静态检查、Go 测试与 Contract。完整 Requirement lifecycle 与独立fixture的
Route Matrix 由 CI 的 `requirement-e2e` 执行；固定生产发布的矩阵由 `scripts/validate-configuration.py` 执行。本地用户表面按 `.cursor/skills/verify-fanloop/SKILL.md` 用隔离候选验证。

`run-unit` 先完成格式与生成物检查，再并发执行互不共享可写状态的 vet、Go test 和本地 Route Matrix；
它等待所有已启动检查并聚合失败。脚本显式使用 Go 自动工具链选择，避免宿主 `GOTOOLCHAIN=local` 让仓库门禁在运行前失败。

`flow skills` remains available for a historical Requirement's pinned package catalog. The current production inventory is empty; product Skill discovery uses the installed Codex plugin marketplace instead.
