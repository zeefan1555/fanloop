---
status: accepted
date: 2026-09-21
amends: ADR-0108
---

# 可视化 Live Workflow 拓扑

`fanloop dashboard` 在 ADR-0108 的安全 whole-file 编辑器上增加只读 Workflow topology。新增
`GET /api/topology?workflow=<workflow-id>`，从同一批当前 live 五 YAML 与绑定 Skills 投影 Stage、Job、
Step、Flow/terminal Route、Loop self/back Route，以及每个 Step 关联的 Prompt、Condition 与 Skill。
生产 Workflow YAML 仍是唯一拓扑真值，Go 与浏览器不承载任何特定 Workflow 的流程语义。
声明 common controls 的 Workflow 会把 `step_start`、`jump` 引用的 Prompt、Common Condition 和
Common Skill 合并到每个 Step 的对应 inspector；动态 jump 不伪造静态 graph endpoint，图边仍只来自
具有明确目标的 Flow 与 Loop Route。

Topology endpoint 同时返回 source anchors：allowlisted 相对路径、1-based inclusive 起止行和 UI 标签。
Go 使用现有 Workflow loader 得到经过完整校验的语义模型，并用 `go.yaml.in/yaml/v3` AST 对同一当前字节
定位真实 mapping entry 或 sequence item。无法建立精确 anchor 时整个只读请求失败；不返回生成的 YAML
snippet，不猜测范围，也不把 anchor 持久化为 Workflow 契约。

浏览器按源码顺序确定性排列 Stage、Job 与 Step，显示 solid forward/terminal edge 和 dashed return edge，
提供方向筛选、Step 聚焦、pan、zoom 与 fit。桌面布局优先保留图画布，fit 保持节点文字可读的最小缩放，
无法完整容纳时继续通过 pan 浏览，而不把整图压缩为不可读缩略图。Step inspector 包含 Step、Prompt、Forward、Return、
Conditions、Skills 六类 fragment；每类允许零到多个来源，按 source identity 去重并保留首次出现顺序。
选择 fragment 仍通过 `GET /api/file` 打开完整真实文件，只滚动并选择 anchor 行范围；从直接文件视图点击
当前 Step 会恢复该 Step 的 inspector 和默认 source fragment。

ADR-0108 的写边界全部保留：`POST /api/file` 是唯一写路径，提交完整当前文件和 observed SHA-256，服务端
继续执行 allowlist、无符号链接、stale、完整候选 Bundle、非空 Skill 和原子替换检查。图本身不能创建、
删除、改名、重排或连接节点，也没有 partial-save、fragment-save、YAML marshal round-trip、历史回滚、
远程访问、Daemon、新 frontend build chain 或 graph dependency。Topology 与文件每两秒轮询；失败保留
最后可用图和编辑器内容，脏编辑器遇到外部变化只进入 stale，不被覆盖。

本变更不修改二十份生产 YAML、任何 Step `id`、`name`、顺序或 `executor`、Route/Condition/Prompt/Skill
语义、`idl/**/*.thrift` 或生成物。它保留 ADR-0038 的五文件职责和 OR-of-AND Route 语义，也保留
ADR-0052 的 Thrift 契约边界。

人工审核记录：主 Agent 在 host turn `01a0c407-9ec4-79c0-87f2-4287edd2de13` 批准完整 Requirement，
包括只读 topology、精确 source anchors、六类 inspector、whole-file-only 保存、零生产 YAML/IDL diff 和
全部现有安全边界；receipt 为
`decision-92936ba93e776b702ad336329aa7036a4ebdc497a26d0c9ca18ef1dbf2032315`。

本决策的生产配置存储/发布、Dashboard编辑或代码验证地图位置条款由 [ADR-0120](0120-host-production-workflows-in-fornax.md) 按其明确范围修订；其余门禁继续有效。
