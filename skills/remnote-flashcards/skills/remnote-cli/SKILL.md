---
name: remnote-cli
description: 使用已安装的 RemNote Direct CLI 做有边界的查询、任务写入、能力发现和回读；内容规则由同包专项 Skill 管理。
---

# RemNote CLI

日常操作直接使用已安装的 `remnote-cli`。链路是 CLI、本机 WebSocket、SDK 插件和官方 SDK。
以当前命令帮助、运行时协议及能力发现为准，不要求 CLI 与插件版本号相同。

## 身份与授权

- 每批写入前运行 `remnote-cli status`，核对实际插件身份、协议、SDK、精确 KB 绑定、权限和写入开关。
  KB 名称只用于展示；不从示例、旧回执或另一知识库推断当前目标。
- 本次运行首次使用一个命令或输入形状时读取对应 `--help`；命令版本变化或参数错误时刷新。
  已确认的同版本语法无需逐对象重查。Workflow 必需 Skill 仍按最新 Status 的精确路径读取。
- 明确的创建、修改或迁移请求授权其具体范围，不额外索取固定口令。只读分析不写入。
  删除、替换、移动和批量迁移先核对目标 ID、引用及影响；能力或授权不足立即停止。
- 每次 shell 执行只运行一个 `remnote-cli` 命令；本机命令串行，不争用同一监听端口。
  每条任务命令内部复用连接，退出关闭，不新增常驻服务。

## 读取与任务接口

先限定知识范围，再集中读取。名称只用于搜索，后续操作使用实际返回的 ID：

- `search` / `search-by-tag` 定位候选；`context --request <FILE|->` 批量读取已知 ID、
  祖先、标签和子树。需要正文的 ID 放入 `remIds`；其他选择器只返回结构元数据，不能当正文。
- 单点 `read` 从浅层开始，按返回的截断、游标和缺失证据补读；不要把整个 KB 正文装入上下文。
- 卡片复习证据来自搜索结果的原生 `cards` 或 `review-stats`，不可用就标为 unavailable。
- 创建术语卡之前先 `search "<query>" --cards-only`。命中且够用则复用标题卡 Rem ID。
- `flashcards apply --plan <FILE|-> --idempotency-key <KEY>` 处理一组新卡、原答案补充和复用卡。
  普通学习批次先写入新原子卡并回读标题 Rem ID，再在组合卡答案使用 `{"remId":"<标题卡 Rem ID>"}`。
  别名 Rem 不能当引用目标。`cardCluster: true` 只在用户明确要求且学习卡契约允许时写入。
- 默认学习路径不调用 `wiki apply`。只有用户本批明确要求 Wiki 时，才用
  `wiki apply --plan <FILE|-> --idempotency-key <KEY>` 处理同一分区的页面、aliases、
  索引和批次日志。不同分区或超出计划预算时按依赖拆成有序批次。
- `verify scope --expect <FILE|->` 用新鲜证据集中检查显式结构预期；完整范围或任何预期未通过均失败。

上述三种任务/验证命令支持 `--summary`，日常编排优先使用；保留状态、完整性、真实 ID、现有哈希、范围、计数及失败明细，失败退出码不变。不加该选项可取完整输出，摘要不能代替需要正文的 context 回读。
每计划最多 7 张新卡或 10 个 Wiki 页，且不超过 100 KiB；同批 Wiki 引用用 pageKey，跨批或 flashcards 计划必须使用真实 Rem ID。
不新增跳过验证选项，计划预览仍是只读检查而非完成回执。

任务命令默认预览；审阅计划及预览后才加 `--apply`。每个计划使用稳定键，批次之间键不冲突；
已有批次不能因重试、跨日或接续会话换键。遵守当前帮助的 JSON、对象数、正文长度和回执上限，
不以分批绕过授权上限，不并行处理有依赖的批次。没有工作时跳过命令并记录原因，不构造空写入。

若任务接口不能保真表达原文的层级、格式或全部必要动作，先判断现有公开命令是否完整覆盖，
不得压平正文、截短答案、改变模式或在写后才发现表达缺口。能力不足就记录精确缺口并阻塞，
由独立维护任务补齐。普通单点操作使用 `create`、`insert-children`、`outline` 等现有命令；
任务命令已完成的标签、折叠、索引或日志不再逐对象重复写入。`flashcards apply` 的 `cardCluster` 按学习卡契约：省略或 `false` 不添加内置 `cc`；只有 `true` 才添加，并且实现上还要有主题标题和至少 2 张新卡。主题 `title` 不是这个开关。用户没要求时不要写 `true`，也不要改调 `object-add-powerup`。写后仍用 `object-has-powerup ["cc"]` 回读，结果与 opt-in 不符就是 remnote-direct 回归。

一级命令未覆盖时先查 `sdk capabilities` 和具体方法帮助，只调用公开、受支持的 SDK 方法。
不调用私有路径、任意 JavaScript 或持久对象句柄。原生事务是否可用以运行时及仓库契约为准；
不可用时不尝试事务或承诺回滚，不把逐条成功包装成原子成功。

## 回执与恢复

任务回执的 `ids` 将语义键映射到真实 Rem ID；`facts` 包含正文哈希、原生卡片 ID 和复习哈希。
不要猜创建顺序，不重新搜索刚创建的对象。更新答案使用精确答案 ID 和 `expectedTextHash`，
保留卡片 ID、父级和复习记录。原子卡标题是短名。长搜索句用 `getOrCreateAliasWithText` 或 `update --add-aliases` 加在这张标题卡上。组合卡引用标题卡 Rem，不引用返回的别名 Rem。直接写 RichText 用 `sdk rem object-set-text` 的 `{"i":"q","_id":"<标题卡 Rem ID>"}`。`flashcards apply` 答案 TextSegments 写 `{"remId":"<标题卡 Rem ID>"}`，不要写 `{"i":"q"}`。`question` 是纯字符串，不会解析括号。标题为 `PE` 时行内显示 `[[PE]]`。flashcards 计划不要使用 `pageKey`。

combo rem-refs use flashcards apply TextSegments {"remId":"..."} or object-set-text RichText {"i":"q","_id":"..."}; do not use create --content-file with [[id:]].

2026-10-07 失败例：`create --parent-id <theme> --content-file` 里的 `[[id:gTpsZTEYvBxB4S1PM]]`（分批加仓）把 T 正文写成 `[[DCF]]0`。不得再走这条。

- 写后以真实回执和精确 ID 回读确认；最终正文仍须语义评审，哈希不能替代它。
- Wiki 批次 verificationScope=affected_objects 的 verified 只覆盖本批受影响页面子树、aliases、索引日志及引用，不代表全 Wiki 验收。答案链接专用计划仅查答案、目标和必要权限前提，不触发整库扫描。
- 本任务有 Wiki 修改时，全部相关写入结束后用现有 verify scope 完整检查 Wiki 根及后代零卡片和残留结构。此后 Wiki 再变更就重查；无修改、仅复用时检查受影响页和引用。不能把一个批次 verified 上报成整个 Workflow 完成。
- 同一 KB、同一计划和对象范围内，完整且未被后续变更影响的检查可以复用。
  任一相关写入、外部编辑、重连、跨会话或不完整读取后，重新核对受影响范围及前置哈希。
- 预览不授予无限期写权限；每次实际写入仍由 CLI 检查活跃写入期限和精确 KB。
- `partial`、`unknown`、超时或断连后停止写入，保留原计划、键、已知 ID 和私有恢复日志。
  先只读核对，不能自动重放未知操作、换键、删除重建或把空搜索当作未写入。
- 只有结果已查明且现有恢复能力明确允许时，才继续确认缺失的动作；否则报告剩余不确定范围。
- 临时传输文件成功后清理；未知结果时保留必要的受限恢复材料。配对配置和日志遵守私有权限，
  不进入 Git、截图或 Workflow 正文证据。输出只记录 ID、哈希、计数、来源引用和结论。
- 不直接访问 RemNote 应用数据目录。修改正文、移位或删除都以官方 SDK 的真实读写为准。

## 故障与分工

区分 CLI 端口占用、插件未连接、认证、错误 KB、权限、协议/能力不兼容和写入关闭。
连接故障可按同包 `control-remnote` 观察桌面状态；不得盲目重启或重装。
需要改代码、安装或权限时保存当前任务位置，进入单独的 RemNote Direct 维护任务；
维护完成后先核对原 ID 再恢复，不在日常制卡中夹带发布、重放压力测试或原生事务探测。

HOME/类型由 `remnote-structure` 管理；制卡和最终校验由同包对应 Skill 管理。
Wiki 只在用户明确要求时由 `remnote-wiki` 管理。日常制卡走 Fanloop 的 remnote-flashcards，
使用本机 Direct CLI，不改走 MCP，也不在 Workflow 外手写创建。
本 Skill 只负责调用边界、任务计划及回执，不重复定义内容模板。
