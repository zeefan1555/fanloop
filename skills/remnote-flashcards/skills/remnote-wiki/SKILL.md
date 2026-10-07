---
name: remnote-wiki
description: 构建、查询、摄取和审计 RemNote 中持续演进的互联 Wiki。用于用户明确要求创建或维护 Wiki、整合来源、基于 Wiki 回答问题或检查健康度。普通闪卡批次默认不调用本技能写页面。
---

# RemNote Wiki

在 RemNote 中维护一个持续积累、相互引用的知识库。本技能适配自 [Karpathy's LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) 与 [Hermes Agent `llm-wiki`](https://github.com/NousResearch/hermes-agent/blob/main/skills/research/llm-wiki/SKILL.md)。

与每次重新检索原始材料不同，Wiki 会把知识整理一次并持续更新：交叉引用已经建立，矛盾已经标出，综合结论反映所有已摄取来源。

**分工：** 人负责选择来源和决定分析方向；Agent 负责总结、交叉引用、归档和维护一致性。

## 当前 Workflow 分工

普通学习批次的默认路径不读取、不执行本 Skill。prepare 上的绑定是可选的。搜术语、建术语闪卡、在组合卡里引用标题卡 Rem ID，都不叫 Wiki，也不能靠一份「未要求 Wiki」回执通过 Route。`wiki_ready` 已不在 Route 上。

只有用户本批明确要求创建或修改 Wiki 时才执行本 Skill。那时只写 Wiki，不改卡片，也不把 Wiki 页面 ID 注入组合卡来代替标题卡引用。失败或未知写入报告 blocked。完成与否不改变 `terms_ready`。

## 何时启用

用户出现以下任一需求时使用本技能：

- 创建、搭建或开始一个 Wiki／知识库；
- 摄取、添加或处理一个来源；
- 在已有 Wiki 的领域内提问；
- lint、审计或健康检查 Wiki；
- 明确要求这次制卡写入或更新 Wiki。

普通金融或概念学习、卡片里出现可复用术语、以及 2–7 张一组的批次，都不启用写入。

## Wiki 位置与结构

Wiki 根是 RemNote 中标题为 `WIKI` 的顶层 Folder，与 `HOME` 同级。它直接保存 Schema、导航、日志、原始来源和 Wiki 页面，不使用概念 Tag 作为中间索引层。

```text
WIKI
├── SCHEMA.md
├── index.md
├── log.md
├── raw
│   ├── articles
│   ├── papers
│   ├── transcripts
│   └── assets
├── entities
├── concepts
├── comparisons
├── queries
└── _archive
```

- `SCHEMA.md`：领域、约定、页面门槛、页面类型和更新策略。
- `index.md`：全部 Wiki 页面的原生引用与一句话摘要。
- `log.md`：按时间追加的操作记录。
- `raw`：不可变的原始 URL、附件、原始 Rem 或逐字材料。
- 页面分区：标准页面直接属于对应分区；页面 Rem ID 就是唯一知识身份和引用目标。
- `_archive`：保存被完全取代的页面，并按原页面类型保留必要层级。

顶层 `WIKI` 设置为 Folder，并添加 RemNote 内置的 Disable Cards Power-up（代码 `u`），作为整个子树不进入复习队列的结构门禁；`raw`、`entities`、`concepts`、`comparisons`、`queries`、`_archive` 和 `raw` 下的来源分区保留在 WIKI 下并设置为 Folder，不平铺到顶层。`SCHEMA.md`、`index.md`、`log.md` 设置为非 Folder 的 Document。RemNote SDK 会让 Folder 同时保持 Document 能力，清除 Document 可能连带清除 Folder，因此目录只要求 `isFolder=true`，CLI 应报告为 `folder`。具体实体、概念、对比、查询和来源条目保持普通 `text` Rem，不能带卡片方向或生成复习卡。新闪卡写入今日笔记并挂 HOME Tag，由卡片引用 Wiki 页面。整个 `WIKI` 子树中所有有子内容的 Rem 默认折叠，不只折叠目录。

页面使用干净标题。全称、缩写、英文名、中文名、旧称和常见拼写作为真实 aliases，全部解析到同一页面 Rem ID。其他笔记和闪卡使用 `[[id:<PAGE_REM_ID>]]` 原生引用连接页面。

不要创建 `wiki-*` Tag，不用 HOME 分类 Tag 组织 Wiki 页面，也不把 Wiki 标准页面拆到每日笔记。真实知识只存在于 RemNote，不在仓库或 RemNote 应用数据目录 中建立内容副本。

## 架构：三层

```text
原始来源
  WIKI / raw 下的 URL、附件、原始 Rem、论文、文章或会议材料
        ↓
Wiki 页面
  WIKI 各页面分区下的实体、概念、对比和高价值查询结果
        ↓
Wiki Schema
  WIKI / SCHEMA.md 与本 Skill
```

**第 1 层——原始来源：** 只读。保留原 URL、文件或原始 Rem 引用；Agent 不静默改写原始材料。

**第 2 层——Wiki：** Agent 维护的标准知识页面。创建、原位更新并用原生 Rem 引用互联。

**第 3 层——Schema：** 领域、结构、页面门槛、页面类型和更新规则。`WIKI / SCHEMA.md` 是 RemNote 内可读版本；当前 Workflow 规则变化时按本次授权同步更新，不另造 Tag taxonomy。

RemNote 的稳定 Rem ID、aliases、父子层级、原生引用、`createdAt`／`updatedAt`、`index.md` 和 `log.md` 分别承担原版文件名、frontmatter、目录、wikilink、时间字段、索引与日志职责。

## 按操作定向

只读取当前操作需要的上下文：

1. 涉及项目术语或既有决策时，读取当前 Workflow Prompt 与本 Skill 相关段落，不固定加载整套文档。
2. 写操作前运行 `remnote-cli status` 并确认写权限；只读查询仅需目标可读。命令用法不确定时再查看对应 `--help`。
3. 已知页面 Rem ID 时直接读取；否则定位唯一顶层 `WIKI`，在相关 `index.md` 分区或 WIKI 根范围搜索规范名和 aliases。
4. 结构变更读取 `SCHEMA.md`；摄取、更新或审计读取相关索引分区和近期日志；普通查询只读取命中页面、必要来源和直接相关页面。

创建页面前必须完成规范名和 aliases 查重。只有 lint、迁移或需要排除大规模同义重复时才扩大到整个 `WIKI` 子树。

## 渐进式读取

制卡和普通查询不得一次递归读取整个 `WIKI`。使用由浅到深的固定策略：

1. **只读根目录：** `read <WIKI_ID> --depth 1 --child-limit 50 --content-mode structured --view full`，只取得固定目录及其 Rem ID。
2. **轻量定向：** `SCHEMA.md` 从 `depth 1` 开始；`index.md` 先读类型标题和相关分区；`log.md` 只读最近 20 至 30 条。只有出现截断或相关内容位于更深层时才增加深度。
3. **从材料提取查询词：** 复用准备阶段清单，归一规范名、缩写及 aliases，合并同义查询。默认只处理学习目标必要术语；用户要求全文术语时覆盖全文适用项。普通词、临时数值不查不建，必要相邻概念才扩展。
4. **限定根搜索：** 使用 `search "<query>" --parent-id <WIKI_ID> --depth 1 --ancestor-depth 3 --child-limit 20 --view full`。先看标题、aliases、父分区和一层摘要，不读取所有命中页全文。
5. **逐页加深：** 对最相关候选先 `read <PAGE_ID> --depth 1`；只有答案、来源或引用仍缺失时才升到 `depth 2`，再按具体子 Rem ID 继续读取。普通制卡不默认超过 `depth 2`。
6. **有限扩展：** 只跟进与当前记忆目标直接相关的前置概念、易混概念和下游概念；首轮最多展开 3 个相邻页面。
7. **查无结果再创建：** 规范名和 aliases 都没有有效页面，且达到页面门槛时，才在正确页面目录下创建并更新 `index.md`、`log.md`；否则跳过建页。

只有 lint、全库迁移、目录修复或用户明确要求全量盘点时，才扩大深度和页面数量。即使如此也按目录分批读取，不能用一次深层 `read` 把整棵 Wiki 装入上下文。

## 初始化新 Wiki

用户要求创建 Wiki 时：

1. 运行 `remnote-cli status`。
2. 明确 Wiki 覆盖领域；只有范围会实质改变 Schema 时才询问用户。
3. 创建唯一顶层 `WIKI` Folder。
4. 在根下创建 `SCHEMA.md`、`index.md`、`log.md`、`raw`、`entities`、`concepts`、`comparisons`、`queries` 和 `_archive`；在 `raw` 下创建四个来源分区。目录节点的查重、创建、Folder 设置和父级核验调用 `remnote-structure`；三份 `.md` 元文档设置 Document 属性。
5. 在 `SCHEMA.md` 写入领域、约定、页面门槛、页面类型和更新策略；在 `index.md` 写入分区目录；在 `log.md` 追加初始化记录。
6. 给 WIKI 根添加 Disable Cards，并回读验证：`remnote-cli sdk rem object-add-powerup --target-id <WIKI_ID> --args-json '["u"]'`，随后用 `object-has-powerup` 确认结果为 `true`。
7. CLI 会在创建时自动折叠新建的非叶子 Rem；回读确认 WIKI 与目录显示为 `folder`、目录仍嵌套在 WIKI 下、三份元文档仍为 Document，并用 `review-stats --root-id <WIKI_ID>` 确认没有卡片。再检查折叠状态、完整层级、`wiki-*` Tag 和重复 Wiki 根。只有发现既有或外部创建内容未折叠时，才运行一次 `outline collapse --root-id <WIKI_ID> --apply` 递归修复整个子树，包括嵌套 Folder、Document 与 Portal。

## Schema 约定

### 命名与元数据

- 页面标题：使用规范名，不添加 `wiki-` 前缀。
- 页面身份：使用页面自身 Rem ID；移动、更新和归档时尽量保留该 ID。
- 页面类型：由直接父分区表达，不用 Tag 重复表达。
- aliases：全称、缩写、翻译、旧称和常见拼写归一到同一页面，不为别名建重复页面。
- 原生引用：使用 `[[id:<PAGE_REM_ID>]]` 指向真实页面。
- 页面元数据：`created`、`updated` 使用 Rem 原生时间，`type` 由父目录表达，`sources` 使用来源子 Rem 或原生引用；需要时增加显式的 `confidence`、`contested` 和 `contradictions` 子 Rem。这些对象对应原 Skill 的 frontmatter，不用 Tag 模拟。
- 来源：专业规则、统计、研究结论、产品行为和易变化知识必须在 `sources` 下引用 `raw` 来源 Rem，并保留原始 URL、日期或版本。
- 质量信号：单一来源、观点性或快速变化的结论使用 `confidence: medium|low`；未解决冲突使用 `contested: true` 和 `contradictions` 原生引用。
- 原始来源元数据：`raw` 来源保存 `source_url`、`ingested` 和正文 `sha256`；重新摄取时用它们判断跳过或来源漂移。

### 页面门槛

- **创建页面：** 实体或概念出现在两个以上来源中，或者是一个来源的核心。
- **更新页面：** 新来源提到已有实体或概念。
- **不创建页面：** 顺带提及、普通词、次要细节、临时数值或领域外内容。
- **拆分页面：** 页面超过约 200 个内容 Rem，或包含多个可独立维护的主题；拆分后互相引用并更新 `index.md`。
- **归档页面：** 内容被完整取代时移入 `_archive`，保留页面 Rem ID、来源和必要引用。

### 页面类型

**实体页面：** 人物、组织、产品、模型或工具。包含概览、关键事实与日期、同其他实体的关系、来源。

**概念页面：** 制卡中新建默认简明术语页：清楚的定义、必要例子、来源及相关原生引用。不默认扩写背景、当前知识综述、开放问题或争议；真实冲突仍按更新策略保留。已有页面不因简明模板删减内容，只补有依据的增量。

**对比页面：** 并列分析。包含比较对象与目的、比较维度、结论或综合、来源；适合时使用表格。

**查询页面：** 保存值得长期复用的深度回答或新综合。简单查找不创建页面。

### Index 与 Log

- `index.md` 按 `entities`、`concepts`、`comparisons`、`queries` 分区；每个页面只出现一次，格式为页面原生引用加一句话摘要。
- 每个分区内保持稳定顺序；单个分区超过 50 页时按首字母或子领域拆分。总页数超过 200 时，在 `_meta/topic-map.md` 建立按主题聚合的导航。
- `index.md` 记录最后更新时间和总页面数；新增、归档或移动页面时同步更新。
- `log.md` 只追加，不改写历史；每条记录使用日期、动作和对象，动作包括 ingest、update、query、lint、create、archive、delete。
- `log.md` 超过 500 条时，将旧记录移动到同级 `log-YYYY.md`，再继续使用新的 `log.md`。

### 更新策略

新信息与现有内容冲突时：

1. 核对日期和适用范围；更新来源通常优先于旧来源。
2. 如果确实矛盾，保留双方立场、日期和来源。
3. 在页面中标记“有争议／待核验”，并引用冲突页面或来源。
4. 在 lint 结果中交给用户审阅，不静默覆盖成单一事实。

## 核心操作

### 有界计划

制卡的并行准备阶段先集中读取相关索引与候选页，统一完成规范名、aliases 和来源查重，再按
[remnote-cli](../remnote-cli/SKILL.md) 的任务协议执行。相同页面分区的改动优先组成一个
`wiki apply` 计划；同计划引用用已有 `pageKey`，跨批引用用已回读的真实 ID。Wiki 助手不提交答案补丁；真实 ID 交给主 Agent 注入卡片计划。独立的答案补链接任务用精确答案 ID 和前置哈希。
不同分区、原始来源或计划不支持的结构按实际依赖分批，各批键和 ID 单独记录；不能强行把来源页
放进概念分区，也不能为少一次调用丢失元数据。只需补引用时不凑页面条目，无增量则跳过写入。

分组索引使用当前 CLI 支持的精确目标父 ID，验证它属于正确索引树；已有条目保持原父级，
不按标题猜分组、不默认写到索引根。歧义、遍历不完整、目标不合法或计划无法表达必要落点时，
写前停止并说明缺口。任务回执已证明的 aliases、索引、日志和引用不再重复写入。

`wiki apply` 会维护计划页面的索引与一条批次日志。来源、元文档日期/计数等未由计划覆盖的动作
先列清，再以现有精确 ID 接口按依赖完成并回读；同批不重复追加相同日志。保留来源不可变、
页面门槛和冲突规则。批次回执 verificationScope=affected_objects 仅证明本批页面子树、aliases、索引和引用，不证明整个 Wiki。所有相关写入结束后，由 persist_and_verify 按 remnote-verify 统一完整核验一次；后续 Wiki 变更使该证明失效。无知识增量只复用，不重复补写来源、索引或日志。

### 1. 摄取

用户提供 URL、文件、粘贴内容或 RemNote 原始笔记时：

1. **保留原始来源。** 在 `raw` 的正确分区下保存原 URL、附件、原始 Rem 引用或逐字材料，并记录摄取日期和内容哈希。再次摄取同一来源时比较 URL／Rem ID、版本和哈希；相同则跳过，变化则标记来源漂移。原始来源一旦摄取，不就地润色。
2. **讨论要点。** 与用户确认哪些内容重要；自动化或批量任务可直接继续。
3. **检查已有内容。** 先查 `index.md`，再在 `WIKI` 子树内统一搜索规范名、aliases 和相关页面，不能边读边制造重复页面。
4. **创建或更新页面。** 新实体／概念达到页面门槛才在对应分区创建；已有页面原位补充，保留 Rem ID、aliases、位置和引用。冲突按更新策略处理。
5. **交叉引用。** 新建或更新页面使用原生引用连接相关页面，并检查必要的反向引用。确有相关页面时至少连接两个；没有真实关系时不伪造链接。
6. **更新导航。** 将新页面的原生引用和一句话摘要加入 `index.md` 的对应分区；更新页面数和日期；向 `log.md` 追加本次动作、来源及全部受影响页面。
7. **回读并报告。** 使用回执 ID 和 context 集中核对本批来源、页面、aliases、引用及索引日志。制卡 Wiki 分支只报受影响范围，最终完整检查交给 persist_and_verify；独立 Wiki 任务则在全部批次最后一次相关写入后运行 verify scope，证明根 Disable Cards、完整子树零卡片且无残留卡片结构。有效局部检查不逐页重做，不能把批次 verified 当整项任务完成。

单个来源触发多个相关页面的原位更新是正常结果；这正是 Wiki 持续积累的价值。

### 2. 查询

用户询问 Wiki 领域问题时：

1. 已知页面 Rem ID 时直接读取；否则读取相关 `index.md` 分区或在 WIKI 根内搜索。
2. 只有相关分区无法定位或需要排除同义页面时，才扩大搜索范围。
3. 读取相关页面、关键相邻页面和来源。
4. 基于已整理知识综合回答，并明确引用所依据的页面。
5. 如果答案是难以重新推导的深度分析、对比或新综合，将其写入 `queries` 或 `comparisons`；简单查找不落盘。
6. 写入时同步更新 `index.md` 和 `log.md`，随后回读。

### 3. Lint／健康检查

用户要求审计 Wiki 时检查：

1. **孤立页面：** 没有其他 Wiki 页面引用的页面。
2. **断裂引用：** 指向已删除、无权限或不存在 Rem ID 的原生引用。
3. **索引完整性：** 每个 Wiki 页面恰好在 `index.md` 出现一次，且摘要和分区正确。
4. **身份完整性：** 规范名和 aliases 是否归一到同一页面 Rem ID；是否存在同义重复页面。
5. **结构完整性：** 顶层 WIKI 与目录节点是否显示为 Folder，WIKI 根是否具有 Disable Cards Power-up（代码 `u`），目录是否仍嵌套在 WIKI 下，三份 `.md` 元文档是否为非 Folder 的 Document；整个 WIKI 子树的非叶子 Rem 是否折叠；页面是否直接位于正确类型分区并保持普通 `text`；`review-stats --root-id <WIKI_ID>` 是否确认全部 `cards` 为空；是否存在 `wiki-*` Tag、Tag 索引层或散落每日笔记的标准页面。
6. **陈旧内容：** 页面结论早于相关来源的新版本，或易变化事实没有日期／版本。
7. **矛盾与质量：** 所有“有争议／待核验”、低置信度和单一来源但未声明置信度的判断。
8. **来源漂移：** 同一 URL、文件或原始 Rem 的内容／版本与上次摄取记录不一致。
9. **页面大小：** 超过约 200 个内容 Rem、需要拆分的页面。
10. **日志维护：** 检查 `log.md` 是否遗漏动作；超过 500 条时在 `log.md` 下按年份创建 `log-YYYY.md` 归档节点。
11. **输出报告：** 按“断裂引用 > 重复身份 > 结构错误 > 孤立页面 > 来源漂移 > 冲突 > 陈旧内容 > 样式问题”排序，给出具体 Rem ID 和建议动作。
12. **记录审计：** 仅当用户要求保存审计记录时，向 `log.md` 追加日期、问题数量和报告引用。

Lint 默认只读。合并、删除或批量更新前列出受影响页面和引用方；预计触碰 10 个以上既有页面时先让用户确认范围。

用户明确要求修复 WIKI 卡片时，保留原页面及 Rem ID：先给 WIKI 根添加 Disable Cards，再对 `review-stats` 命中的具体 Rem 使用 `object-set-practice-direction --args-json '["none"]'` 清除卡片方向，最后重新检查整个 WIKI 子树为 0 张卡。不要删除或重建 Wiki 页面来解决复习队列问题。

写后回读受影响页面与引用；制卡由 persist_and_verify 调用同 Workflow 的 [remnote-verify](../remnote-verify/SKILL.md) 完成最终 Wiki 检查。批次失败只修失败对象，不能仅以 Disable Cards 已开启或局部 verified 判断整个 Wiki 通过。

## 使用 Wiki

### 搜索

- 定位根与元页面：`remnote-cli search "WIKI" --content-mode structured --view full`。
- 在 Wiki 根范围搜索：`remnote-cli search "<query>" --parent-id <WIKI_REM_ID> --view full --ancestor-depth 3`。
- 读取页面与引用：`remnote-cli read <REM_ID> --content-mode structured --view full`，必要时使用 `sdk rem object-rems-being-referenced` 和 `object-rems-referencing-this`。
- 读取 aliases：使用 `sdk rem object-get-aliases`。

命令参数、预览、写入授权、超时与私有恢复材料统一遵循同包 remnote-cli，不逐页面重复查帮助。
CLI 创建的非叶子 Rem 默认自动折叠；`outline collapse --root-id <WIKI_ID> --apply` 只用于既有内容、
外部创建内容或审计发现的异常。未知写入先按原计划、键和 ID 只读核对，不能仅靠重新预览就重放。
Portal 的折叠状态按“被展示页面 Rem + Portal Rem ID”回读。

### 批量摄取

同时摄取多个来源时：

1. 先读完所有来源。
2. 统一识别全部实体和概念。
3. 对全部候选做一次查重和别名解析。
4. 在当前命令的大小上限内，按页面分区和引用依赖执行计划；超限分批但不放松授权、完整性或查重。
5. 每批由任务入口维护相关 `index.md` 和 `log.md`，额外元数据仅补任务未覆盖部分；多个批次不重复记录同一动作。
6. 用一份结果清单报告整个批次。

### 归档

内容被完全取代或领域范围变化时：

1. 在 `_archive` 下按需创建原页面类型分区。
2. 将页面原位移动到对应归档分区，尽量保留 Rem ID。
3. 从 `index.md` 的活跃分区移除，并在归档部分保留引用和替代页面。
4. 检查全部引用方；需要历史上下文的引用保留并标注“已归档”，应指向新页面的引用更新为新 Rem ID。
5. 向 `log.md` 追加归档动作并回读全部改动。

### RemNote 集成

- 父子层级承担目录结构和页面类型。
- WIKI 与目录节点使用 Folder，三份 `.md` 元文档使用 Document；页面条目保持普通 `text`；整个 WIKI 子树的非叶子 Rem 默认折叠。
- 原生引用承担 `[[wikilinks]]` 的作用。
- aliases 承担同义词和旧文件名解析。
- `index.md` 承担全量目录和一句话摘要。
- `log.md` 承担活动时间线；原生时间字段辅助核验。
- `raw` 下的 URL、附件或原始 Rem 引用承担原始材料层。
- 不使用 Wiki Tag、Tag taxonomy、Tag 反查或查询 Portal。
- `remnote-cli` 经本机 WebSocket 提供无界面读写；只有 CLI 无法完成原生界面状态时才使用 `control-remnote`。

## 常见陷阱

- **不要修改原始来源。** 纠正写进 Wiki 页面，并保留来源原貌。
- **每次先定向。** 跳过 `SCHEMA.md`、`index.md`、近期 `log.md` 和已有页面检查会制造重复概念和漏链。
- **不要创建 Wiki Tag。** 页面直接位于类型分区，页面 Rem ID 就是身份。
- **不要为顺带提及建页。** 遵守页面门槛。
- **不要创建孤立页面。** 有真实关联时建立至少两个原生引用并检查反向关系。
- **必须更新 `index.md` 和 `log.md`。** 缺少两者会让 Wiki 失去导航和审计能力。
- **保持页面可扫描。** 页面过长或混入多个主题时拆分并互相引用。
- **批量修改前确认。** 摄取预计会更新 10 个以上既有页面时先确认范围。
- **显式处理矛盾。** 保留双方说法、日期和来源，不静默覆盖。
- **Wiki 不是掌握证据。** 是否学会只由 RemNote 原生复习数据判断。
- **引用不隐藏答案。** 闪卡即使无法点击 Wiki 链接，也必须独立可答。

## 相关工具

- `remnote-cli`：搜索、读取、写入、aliases、父子层级和引用关系的默认入口。
- `remnote-structure`：负责 Folder／Tag 的查找、动态判型、创建和父子层级核验。
- `remnote-flashcards`：用费曼法和 SuperMemo 原则制卡，并调用本技能解析可复用概念。
- `research`：需要外部来源时优先读取官方文档、论文或一手材料，再交给本技能摄取。

原版 Hermes Skill 的 `SCHEMA.md`、`index.md`、`log.md`、`raw/` 和页面目录在这里按原名一一映射为 `WIKI` 下的原生 Rem 层级；frontmatter 映射为 aliases、父分区、来源子 Rem、质量标记和原生时间字段。其“先定向、再查重、原位更新、交叉引用、保留来源、显式冲突、可审计”的工作流保持不变。
