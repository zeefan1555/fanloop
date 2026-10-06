---
name: remnote-flashcards
description: 创建、修改或评审 RemNote 闪卡。普通卡分成原子术语卡（短费曼背面）和组合卡（STAR）；写入今日笔记并挂最细 HOME Concept Tag。默认不加 Card Cluster、不写 Wiki。
---

# RemNote 笔记与闪卡

## 先选模式

- **只读评审**：读取目标卡片并给出问题与修改建议，不要求写权限，不执行写入、发布或界面操作。
- **普通闪卡**：按增量学习、费曼法和 SuperMemo 规则创建或修改卡片。先区分原子术语卡和组合卡。轻量学习默认不写 Wiki；只有用户明确要求 Wiki 时才查询或创建 Wiki 页面。
- **原文导入／原文摘录**：保留用户指定原文，只读取对应模式规则；跳过费曼改写、STAR、Wiki 补充和 ELI5。
- **普通笔记／文章**：仅在用户明确要求写入 RemNote 普通内容时处理，并读取文章格式；不执行闪卡评审流程。

先确定模式，再读取该模式引用的参考文件。共享的 ID、分类、去重和写后回读规则仍适用。

## 极简分工

- HOME：顶层 `HOME` Folder 下保存 `00-NOTE`、`01-PROJECTS`、`02-AREAS`、`03-RESOURCES` 和 `04-ARCHIVE` 五个一级 Folder；`00-NOTE` 保存尚未归入正式知识结构的随手笔记和凭证记录。更深节点的 Folder／Concept Tag 查找、动态判型和创建由同 Workflow 的 [remnote-structure](../remnote-structure/SKILL.md) 统一管理。
- WIKI：顶层 `WIKI` Folder，与 HOME 同级，按原 `llm-wiki` 组织直接保存 `SCHEMA.md`、`index.md`、`log.md`、`raw`、`entities`、`concepts`、`comparisons`、`queries` 和 `_archive`。目录节点使用 Folder，三份 `.md` 使用 Document，具体页面保持普通 `text`，整个 WIKI 子树中的非叶子 Rem 默认折叠；页面身份、别名、来源和引用由同 Workflow 的 [remnote-wiki](../remnote-wiki/SKILL.md) 统一管理，不使用概念 Tag。
- 每日笔记：保存闪卡、普通笔记和文章的真实实体。新闪卡写入今日笔记并挂 HOME 分类 Tag；Wiki 页面直接属于 WIKI 的对应分区。

使用普通文档和 Rem 维护这套结构。HOME 中的 Folder 组织分类树，Concept Tag 通过标签关系归类每日笔记中的内容；WIKI 页面直接保存定义、例子、来源和互联引用，不生成闪卡。不创建 `wiki-*` Tag，不使用查询 Portal，也不把 Wiki 页面复制到 HOME。

一份内容只有一个实体。单张卡直接放在今日笔记并挂 HOME 中最具体的 Concept Tag；同一主题生成多张卡时，先在今日笔记建立一个主题 Rem 并给主题打该 Tag，再把卡片作为它的直接子 Rem。卡片可以引用 WIKI 页面，但不得放入 WIKI。普通文章仍写入每日笔记，并使用主题标签和“文章”类型标签。

## 分类

先搜索与当前内容相关的分类和祖先路径，再选择最具体的类别；只有结构审计或迁移才读取 HOME 全部 Tag：

- 01-PROJECTS：正在推进、会结束的具体结果。
- 02-AREAS：长期负责或持续维持的领域。
- 03-RESOURCES：学习或参考主题。
- 04-ARCHIVE：已结束且需要保留的内容。
- 00-NOTE：尚未归入正式知识结构的随手笔记和凭证记录。

按 `remnote-structure` 查找或创建分类结构。学习批次打领域分组下最细的那一个 Concept Tag，使用精确 Rem ID 建立标签关系。例如路径 `HOME / 03-RESOURCES / 金融知识 / 估值`：`03-RESOURCES` 与 `金融知识` 是分组，打在批次上的是 `估值`。已有更细 Concept 时，不要停在一级 Folder 或宽泛领域上；没有该 Concept 时先创建并回读类型和父级，再打 Tag。

持续课程或教程系列的这个最细 Tag 用教程或课程名，不用作者名。例：`HOME / 03-RESOURCES / 金融知识 / 人人都是量化交易员`。作者可以记在别的元数据里。`刑无刀` 不是这门课的 Tag。单篇概念批次仍用该概念的 Tag，例如 `估值`。

单张卡给卡片自身打这个 Tag。多张卡放在主题 Rem 下时，只给主题打 Tag，子卡通常不再打同一个分类 Tag。普通文章再复用 `HOME / 03-RESOURCES / 文章` Concept 作为第二个类型标签。名称只用于查找和展示。

## 顶层内容命名

新建主题 Rem、普通笔记和文章使用“日期 + 类型或来源 + 中文主题”的可搜索标题；RemNote 标题不添加 `.md` 后缀。例如：`2026-06-12 X 长期投资框架`、`2026-06-12 meeting 项目复盘`。

- 日期提供时间线索；类型可用 `meeting`、`project`、`book`、`resource`、`idea`，外部内容可直接使用来源名。
- 单篇文章或单章的闪卡主题按[学习卡契约](references/learning-card-contract.md)：`YYYY-MM-DD <来源类型> <文章原 H1>`。书稿例：H1「前言」写成 `2026-10-06 book 前言`。不要改写成「术语与方法」或「量化地图」。
- 外部内容保持中文可读主题；项目材料优先保留项目语义，不为模板损害辨识度。不是单篇文章时，才用概念名或用户指定的主题，例如 `2026-10-06 resource 市盈率 PE`。
- 只对新内容和用户明确指定的高价值旧内容应用；不批量重命名历史内容。
- 该规则只作用于顶层内容单元。单张卡的正面仍是可直接复习的问题，原文导入和原文摘录仍保留原文标题，WIKI 页面仍使用干净规范名。

## 共用概念

普通主题 Tag 可以容纳多篇笔记，不按每篇笔记新建分类。轻量学习路径默认跳过 WIKI：普通金融或概念批次不查询、不创建、不引用 Wiki 页面，也不为了术语建页。术语的最小定义写在卡片自己的背面。

只有用户明确要求 Wiki 时，才由 Wiki 助手按同 Workflow 的 [remnote-wiki](../remnote-wiki/SKILL.md) 查询或创建页面，并在写卡前注入真实页面 ID。引用不能替代本卡作答所需的最小定义、条件与结论。普通词和一次性细节即使走 Wiki 路径也不建页。原文模式始终不补 Wiki 引用。

用户没有要求 Wiki 时，仍要留下 `wiki_ready` 回执：`not_applicable`，原因写明「用户未要求 Wiki」，映射和写入都为空，不调用空计划。这个回执记为 passed，不是未完成。

## 增量学习

每次制卡前先把材料拆成独立记忆目标。创建任何术语卡之前，按短名、别名和关键问法运行 `remnote-cli search "<query>" --cards-only --include-review-stats --view full`，找同一记忆目标的术语闪卡，并按最具体的主题 Tag 补充检索。命中且这张卡够用时复用它：组合卡引用该标题卡的 Rem ID，不新建。只有没有术语闪卡时才创建。

Wiki 概念页不是术语闪卡。搜索命中 Wiki 页面，或该 Rem 的 `cards` 为空，都不算已经有这张卡。默认不查询 Wiki；即使用户要求了 Wiki，页面是否存在也不代表学习者已经掌握，也不代替上面的 `--cards-only` 搜索。

用户明确说某一来源的旧卡质量差，或要求重做时：删除或替换那些卡，不要复用该来源上的低质量闪卡。用户没有要求重做时，仍按下面的增量复用。

对每个记忆目标只做一种处理：

- 已有卡片表达同一记忆目标，且新材料没有增加事实、条件、边界或纠正：跳过，不重复创建。
- 已有卡片表达同一记忆目标，但新材料提供了实质补充或纠正：编辑原卡对应的答案 Rem，保留卡片 Rem ID、位置和复习记录；修改后重新回读并评审。
- 新材料包含独立的新记忆目标：创建新卡；不能为了覆盖材料而改写成与旧卡近似的问题。

“是否学会”不使用二元猜测。优先读取 RemNote 返回的卡片复习事实，包括复习历史、最近复习时间、下次到期时间、连续答错次数和原生掌握等级，并区分未复习、正在巩固、有失败信号和稳定掌握。稳定掌握的旧卡遇到重复材料时直接跳过；正在巩固或有失败信号的旧卡也不复制，只在内容本身存在歧义、干扰或知识缺口时修复原卡，否则继续交给 RemNote 的调度器。

搜索结果已带 `cards` 时直接使用其中的原生复习事实；需要检查整个主题、标签或今日笔记时，分别使用 `remnote-cli review-stats --root-id <rem-id>`、`--tag-id <tag-rem-id>` 或 `--today`。命令返回一个 Rem 的空 `cards` 数组时，只能说明该 Rem 没有生成卡片；`repetitionHistory` 为空时表示尚无复习记录。

如果当前 CLI 没有暴露上述复习数据，必须把状态记为“复习证据不可用”，本次只能执行内容查重和原卡更新；不得根据创建日期、Wiki 是否存在、搜索命中或主观印象声称用户已经学会，也不得伪造复习记录。用户在对话中的纠正用于更新内容或规则，不替代 RemNote 的真实复习历史。

## 卡片写作

创建、修改或修复普通闪卡时，先读[学习卡契约](references/learning-card-contract.md)。卡型、短标题与别名、STAR 原文、箭头、主题、Concept Tag、spec、Card Cluster、写入前 `preflight` 和审卡五问都只以契约为准。再读 [费曼学习法制卡](references/feynman-card-method.md) 和 [SuperMemo 20 条知识表述原则](references/supermemo-20-rules.md)。先用费曼流程证明理解，再用 SuperMemo 原则拆成可复习的原子卡。原文导入和原文摘录不套这两份改写规则，也不套契约里的费曼和 STAR。

费曼流程必须在排版前完成。对每个候选记忆目标先做到：

1. 不照抄原文，用零基础读者能懂的话讲清它是什么、为什么成立、怎样在具体例子中发生，以及最容易失效的边界。
2. 找出未解释术语、缩写、循环定义和条件到结论之间的跳步；回到来源补齐后重新讲一遍。用户没有要求 Wiki 时，不要靠 Wiki 页面补这个缺口。
3. 能换一个例子得到同样结论，并能用一句话指出本卡唯一要提取的答案，再进入制卡。

做不到任一项时，继续学习或放弃该卡，不能用更长答案、更多链接或固定标题掩盖理解缺口。

用户明确要求 ELI5、图解或可分享网页时，读取[复杂概念的 ELI5 网页](references/complex-concept-eli5.md)，调用 `eli5` 技能生成并按用户要求发布，再把链接写回 RemNote。未要求独立网页时，复杂概念直接在卡片和对话中解释。

按契约生成 0..N 张卡片。一篇文章或一个主题优先 3–7 张；目标更少时不凑数，独立目标明显多于 7 个时按主题拆开。对照稿见 [市盈率干跑示例](references/example-pe-valuation.md) 和 [前言文章干跑示例](references/example-book-preface.md)。前者是概念批次，主题用 `resource`。后者是单章，主题用文章原 H1。决策记录见 [ADR-0129](../../../../docs/adr/0129-lightweight-remnote-learning-cards.md)。

## 两种写入格式

写入前先判断是闪卡还是普通笔记。闪卡写入今日笔记，并挂最细的 HOME Concept Tag。普通笔记和文章可写入每日笔记。只有用户明确要求 Wiki 时，内容才引用 Wiki 页面。

创建、修改或修复普通文章时，必须读取[文章格式](references/article-format.md)。

### 闪卡

实体放在每日笔记。只有一张卡时，卡片直接属于今日笔记，并挂最细 Concept Tag。同一批次有多张卡时，可以在今日笔记下建一个普通 `text` 主题 Rem，把卡片作为它的直接子 Rem，方便阅读；只给主题打该 Tag，子卡通常不再打。主题保持 `text`，不转成 Document 或 Folder。

Card Cluster 只按学习卡契约：用户没要求时计划省略 `cardCluster` 或写 `false`，两者都不添加 `cc`；用户明确要求，且有主题标题、新卡至少 2 张时才写 `cardCluster: true`。不要用 `object-add-powerup` 补 `cc`。写后按 [卡片集群](references/card-clusters.md) 用 `object-has-powerup ["cc"]` 回读，结果记入 `cluster_powerup`。意外的 `true` 是 remnote-direct 回归。

可导入形状、别名和标题卡引用见契约与两份干跑示例。段间空 Rem 含一个 `U+200B`；Markdown 导入会忽略真正的空 bullet。传输语法中的缩进用于把答案归属于卡片。

其他已有语法见[格式参考](references/remnote-import-format.md)。

#### 原文导入闪卡

用户要求“直接使用原本笔记内容”时进入原文导入模式，不套用 STAR 改写：

- README 只负责目录和 Tag 映射；README 中每个单篇文章链接对应一张多行闪卡。
- 正面使用原文章标题，背面按原顺序保留完整正文，不拆卡、不摘要、不改写观点，也不补 Wiki 引用。
- 仅删除 YAML frontmatter、Decks 锚点等 Obsidian 元数据，并把 HTML 标题、列表、链接和代码块转换为等价的 RemNote 结构；可见内容不变。
- 卡片只使用 README 中最具体的二级分类 Tag，不使用“文章”类型 Tag。
- 写入前按原文章标题和来源路径查重；写入后逐段对比源文，确认无缺段、乱序或意外生成的子卡片。

```markdown
- 原文章标题 ===
    - 原文第一段
    - 原文第二段
    - 原文小标题
    - 原文后续内容
```

#### 原文摘录闪卡

用户要求“原文标题是问题，原文内容是答案”或只希望保留材料中的指定段落时，进入原文摘录模式。该模式不套用 STAR 改写，但与“原文导入闪卡”不同：它只保存用户指定的摘录，不要求导入整篇文章。

- 正面使用原文标题，作为一个完整、可直接复习的问题；不得替换成摘要标题或自行改写的问题。
- 背面使用与该标题对应的原文内容；保留原文措辞、顺序和必要段落结构，不总结、不扩写、不补充 Wiki 定义。
- 可以删除不属于摘录范围的前后文，以及 YAML frontmatter、Decks 锚点等元数据；不得删除摘录内部的可见内容。
- HTML 标题、列表、链接和代码块转换为等价的 RemNote 结构，但不改变可见含义。
- 默认沿用普通闪卡的最具体主题 Tag；如果材料明确给出来源 Tag，按来源 Tag 映射，不额外添加“文章”类型 Tag。
- 写入前按“原文标题 + 来源路径或来源标识”查重；写入后逐段对比摘录，确认问题与答案对应、内容完整且没有意外生成子卡片。

```markdown
- 原文标题 ===
    - 原文摘录第一段
    - 原文摘录第二段
    - 原文摘录中的列表项
```

### 普通笔记

普通笔记按[文章格式](references/article-format.md)创建和核验。

## 工作流

本 Skill 在 prepare_cards_and_wiki 只起草，在 persist_and_verify 才写卡；使用 resolve_home_tag 已回读的分类 ID、来源及查重结果，不重新访谈或重复收集材料。分类不正确时回第一步修复。

1. 选普通或原文模式；只读评审和普通笔记/文章不属于本 Workflow 入口。写前运行 `remnote-cli status`，核对连接、兼容版本和写入权限。
2. 按增量学习查重。创建术语卡前先 `search --cards-only`。同一记忆目标的术语闪卡够用就复用标题卡 Rem ID；Wiki 页不算。用户要求重做时删除或替换该来源的旧卡，不复用。否则重复跳过、实质补充原位更新、新记忆目标才创建。保留实际 cardId、位置和复习记录。允许 0 张新卡，仍记录复用 ID 并前进。
3. 普通卡先分成原子术语卡或组合卡，完成费曼与 SuperMemo 自检。每批草稿完成立即交独立只读助手按 remnote-verify 审卡。用户未要求 Wiki 时不派 Wiki 助手。用户明确要求 Wiki 时，Wiki 助手才解析术语与准备页面，并且不等待它完成才审卡。审卡回执包含来源及草稿摘要、覆盖卡号、每张普通卡的 kid_clarity 五问、每张组合卡的逐字 star_headings、失败项；只修失败卡再复审。
4. 确认当前材料的 draft_quality_passed、wiki_ready 均 passed，且 `preflight` 与当前 `draft_digest` 一致、全部门禁为 pass，`source_h1` 与 `theme_title` 一致。未要求 Wiki 时，wiki_ready 的 passed 来自「用户未要求 Wiki」的不适用回执，不注入 Wiki ID。要求 Wiki 时，助手结束且证据覆盖最新草稿及术语后，才注入真实 Wiki ID。用 SDK `date get-todays-doc` 获取本次新卡日记；恢复使用原回执日期。按[remnote-cli](../remnote-cli/SKILL.md)组成有界 `flashcards apply` 计划，先预览；预览后仍按契约重扫箭头和主题，再显式 `--apply`。单卡、同一主题下的多卡、原答案补充和复用使用各自真实 ID；稳定键不冲突。没有实际对象时不调用空计划。
5. 普通卡按契约进入计划。任务入口负责最细 Concept Tag，以及可选的普通 `text` 主题。`cardCluster` 只在契约允许时为 true。原文模式先核对计划能表达完整正文、顺序和层级；超过限制不能截短或改写，应选完整支持该结构的现有入口，否则报告能力缺口。写后使用回执 ID 集中核对父级、类型、最终答案和原生卡片集合；部分成功按 remnote-cli 的未知写入规则保存并核对，不整批重建。
6. persist_and_verify 调用 [remnote-verify](../remnote-verify/SKILL.md)，将真实正文与已审草稿逐项对应，改变正文的卡独立重审；未改变且证据新鲜的审查直接复用。内容、结构和适用引用全部通过才报 flashcards_written、terms_referenced 与 result_verified，随后轻量复盘交付，不在复盘新增知识写入。未要求 Wiki 时，缺少 Wiki 引用不算失败。

## 修改与核验

新卡片写入今日笔记并挂 HOME Tag，普通笔记和文章可写入今日笔记；补充已有卡片、文章或 Wiki 页面时更新原 Rem，保留原位置和 ID，不因日期变化创建副本。只维护本次相关内容；日常写入不顺带整理全库、重命名无关分类或清理 Portal。

只修改答案正文时，优先使用任务计划的 `answerPatches`，提供精确卡片/答案 ID 与刚读取的 `expectedTextHash`。计划不支持的原位结构操作先核对公开 SDK 能力及授权；需要背面字段时查 `object-set-back-text`，不可把卡片正面 ID 当作答案子 Rem ID。多行字符串不等于多个 Rem。必须保留卡片 Rem ID、cardId、位置和复习记录；能力不足或结果不明就 blocked，不用已拒绝的原子 update 命令、替代卡或删除重建绕过。

回读折叠内容时若有 `IsCollapsed`，按具体 Rem ID 继续读取，不能把省略当丢失。

完成前核对：

- `remnote-verify` 已证明：新单卡直接属于今日笔记并挂最细 HOME Concept Tag。多卡若使用主题，主题是普通 `text`、直接属于今日笔记、只由主题挂该 Tag，子卡不重复打标。`cluster_powerup` 与契约的 opt-in 一致；用户未要求时 `object-has-powerup ["cc"]` 为 false。意外的 true 判失败，不要补加，也不要改调 `object-add-powerup`。主题未被转换为 Document/Folder，标题没有残留 `/cluster`。
- 新建顶层主题、普通笔记和文章符合“日期 + 类型或来源 + 中文主题”；卡片正面、原文标题和 WIKI 规范名没有被该模板污染。单篇或单章的 `theme_title` 与 `source_h1` 按契约一致。
- 本次新增的 PARA 分类 Tag 按 `remnote-structure` 回读为 Concept，并且是领域分组下最细的那个 Tag。祖先路径证明在 `preflight.tag_path`。作者名、粗 Folder 和 Wiki 页都不能当这个 Tag。文章内容本身仍按文章规则保持 `text`。只有用户要求 Wiki 时，Wiki 页面才按 `remnote-wiki` 直接属于对应分区，并且不创建概念 Tag。
- 普通学习卡的 `preflight` 全部门禁为 pass。新建术语卡都有 `search --cards-only` 的 miss。复用的是术语闪卡标题 Rem。审卡回执含每张卡的五问 `kid_clarity`，且全部为 yes；组合卡列有逐字 `star_headings`。
- 原文导入闪卡一篇源文章对应一张卡，标题、正文顺序和可见内容与源文一致，Obsidian 元数据已移除，正文没有意外生成子卡片。
- 原文摘录闪卡的正面等于原文标题，背面只包含指定原文摘录；摘录内部顺序、可见内容和结构保持一致，元数据已移除，正文没有意外生成子卡片。
- 普通文章符合[文章格式](references/article-format.md)中的完成核验条件。
- 本次新增的 HOME 节点按 `remnote-structure` 区分 Folder 与 Concept；闪卡实体位于每日笔记并通过 Tag 归类，不创建查询 Portal。
- 用户未要求 Wiki 时，本次没有新建或更新 Wiki 页面，卡片也不依赖 Wiki 链接才能作答。用户要求 Wiki 时，页面直接属于 `entities`、`concepts`、`comparisons` 或 `queries`，Title 使用干净规范名；`index.md` 和 `log.md` 已同步更新；引用指向真实 Rem ID，aliases 归一到同一页面，无同义副本或 `wiki-*` Tag。
- 普通学习卡满足学习卡契约的失败条件。背面形状、短标题、别名、标题卡引用、STAR 原文、箭头、口诀和儿童口吻不在本清单再写一遍。
- 卡片保持确认后的排列与答案内容，不能只检查工具返回成功或数量。
- 每批草稿已经独立审查，最终回读正文逐项对应已审草稿；正文改变及失败卡已独立复审，未变部分复用新鲜证据。
- 删除概念前检查所有引用已迁移；删除物移入回收站，交付时告知可恢复。
- 用户要求 Wiki 时，WIKI 中没有生成卡片。普通笔记和文章仍保存真实实体，不创建指向同一实体的链接占位或迁移说明。
- 不为日常写卡添加额外索引、日志、同步系统或全库清理任务。

交付时简述写入哪个日期、多少张卡／篇笔记、使用哪个 HOME 分类标签和核验结果。
