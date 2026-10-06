# 学习卡契约

普通学习卡（原子术语卡、组合卡）的规则只维护在本文。制卡 Skill、审卡、写入和复盘按本文执行，不另写一套背面、标题、主题、Tag 或 Card Cluster 规则。

`prompt.yaml` 的 `draft_quality_condition` 与 `persist_and_verify_flow` 复述下面的失败条件，用来挡住 Route。改清单时这三处一起改。Step id、Route 和 Skill 绑定不变。

原文导入和原文摘录不套本文的费曼、STAR、儿童口吻和箭头禁令；它们仍逐段保真。本文也不约束普通文章。

运行时是 RemNote Direct：`remnote-cli` 经本机 WebSocket 连到 SDK 插件。普通学习批次没有第二套写入入口。

## 何时读取

起草、修改、审查或写入普通学习卡之前读本文。写入前填 `preflight`。独立审卡回执按「儿童口吻」和「STAR 原文」填写。

## 卡型

- **原子术语卡**：一个术语，记定义、公式或单点原理。背面是标题卡的直接子 Rem，按顺序：一行正式定义、一行白话类比、可选一行例子。没有例子就不要这一行。不要写 STAR。
- **组合卡**：两个及以上术语的对比、因果链或决策规则。背面用 STAR。学习用组合卡四段都写，R 是一句口诀。
- 同一批先排原子术语卡，再排组合卡。组合卡用到的术语必须已经有原子卡；复用的标题卡算已有。
- 方法论文章把有名字的方法写成短标题原子卡，例如 `做看疑`、`四个阶段`，再用组合卡把方法接到术语上。只有软叙述、没有这些原子卡，门禁 `named_methods` 失败。
- 一篇文章或一个主题优先 3–7 张，允许 0..N。每张卡一个记忆目标。
- 用户没有明确要求时，不为 `spec` 或「策略规格说明书」建卡。

白话类比写给大约 8–10 岁、没有该领域背景的小朋友：一句短句、一个摸得到的日常场景、这一行没有未解释的术语，角色对得上正式定义里的量。例子若保留，是能口算的算术，不是把公式里的符号换成数字。

组合卡的问题和 S、T、A、R 每一段，写给识字的大约 5 岁小朋友，读一遍就要懂。S 是有人物和物件的具体场景，并点名哪里容易看错。T 是一句，句首是「你要学会」，并写出要避开的错。A 用小数目走一遍演算。R 是一句口诀。问题从刚学的原子卡接过来，只问一个决定。

算术写成「20 除以 2 等于 10」，或只用 `=`、`÷`。

## 标题、别名、引用

原子卡标题是短名，例如 `PE`。标题里出现 `？` 或 `是什么` 就失败。长搜索句是这张标题卡的别名，例如 `市盈率（PE）是什么？`，在标题卡创建之后再用 `getOrCreateAliasWithText` 或 `update <标题卡 ID> --add-aliases` 加上。别名 Rem 只供搜索。

组合卡只引用标题卡 Rem。直接写 RichText 用 `{"i":"q","_id":"<标题卡 Rem ID>"}`。`flashcards apply` 的答案段写 `{"remId":"<标题卡 Rem ID>"}`。行内显示短名，例如 `[[PE]]`，点击打开带费曼子 Rem 的卡。不要引用别名 Rem，不要写字面括号，不要写 `pageKey`，不要编造 ID。`question` 是纯字符串，不会解析括号；正面用 Markdown 导入或 `update --title`，并写 `[[id:<标题卡 Rem ID>]]`。

## STAR 原文

学习用组合卡的四个标题必须逐字是：

- `S — Situation：背景`
- `T — Task：任务／目标`
- `A — Action：行动`
- `R — Result：结果`

破折号是 `—`，冒号是 `：`，斜杠是 `／`。只有中文、没有字母，不算通过。每个标题下可以有一整段。段间空 Rem 含一个 `U+200B`。

## 箭头

普通学习卡的正面和答案行不得出现 `→`、`←`、`⇒`、`⇐`、`>>`、`<<`、`⟶`、`⟵`、`➔`、`➜`、`➡`、`⇨`、`⇦`、`↔`。这些符号会让答案再生成一张嵌套卡。`>>`、`<<` 只属于其他卡型的分隔符，不写进多行 `===` 学习卡。

## 主题

准备回执同时记下 `source_h1` 和 `theme_title`。

材料是单篇文章或单章时，`source_h1` 是该页原 H1，逐字，不转述。`theme_title` 必须等于 `YYYY-MM-DD <来源类型> <source_h1>`，中间各一个空格。书稿的来源类型是 `book`。H1 为「前言」时主题是 `2026-10-06 book 前言`，不是「术语与方法」或「量化地图」。日期用本次写入日。

`source_h1` 与 `theme_title` 对不上，或 `source_h1` 不是原 H1，门禁 `theme_literal` 失败，不得 `--apply`。不能沿用上一稿的 pass。

不是单篇文章时，`source_h1` 写 `n/a`，主题用日期、来源类型和概念名，例如 `2026-10-06 resource 市盈率 PE`。材料里已经有 H1 却改写成概念名，仍算失败。

## Concept Tag

学习批次打领域分组下最细的那个 Concept，例如 `HOME / 03-RESOURCES / 金融知识 / 估值`。持续课程用教程名，例如 `人人都是量化交易员`。

拒绝：

- 作者名，例如 `刑无刀`
- 只停在粗 Folder，例如 `03-RESOURCES` 或 `金融知识`，而更细的 Concept 已存在或本应创建
- 把 Wiki 页面当作分类 Tag

`preflight.tag_path` 要带祖先路径证明：`home_id`、`tag_id`、`ancestor_ids`，以及从 HOME 到叶子的标题。叶子必须是 Concept，不是 Folder、Document 或 Wiki 页。

多张卡只给主题 Rem 打这个 Tag，子卡不重复打。单张卡打在卡片自身。

## 查重

创建术语卡之前，按短名、别名和关键问法运行 `remnote-cli search "<query>" --cards-only --include-review-stats --view full`。命中且这张术语闪卡够用，复用标题卡 Rem ID，不新建。Wiki 页和空 `cards` 不是命中。用户要求重做时，不复用该来源的旧卡。

## Card Cluster

默认不加。2–7 张卡、主题标题，都不是开关。

- 用户没有明确要求集群：计划省略 `cardCluster`，或写 `false`。两种都不会添加内置 `cc`。不要写 `true`。
- 用户明确要求集群，且计划有主题标题、新卡至少 2 张：写 `cardCluster: true`。只有这时 Direct 才添加 `cc`。
- 用户要求了集群，但没有主题标题或新卡少于 2 张：不要写 `true`，门禁 `card_cluster` 失败，向用户说明条件不满足。
- 普通学习批次不调用 `object-add-powerup` 去补 `cc`。帮助已经列出 `cardCluster`，没有「帮助没写就改走另一条命令」的路径。

写后对每个多卡主题运行 `object-has-powerup ["cc"]`，结果写入 `flashcards_written.cluster_powerup`。用户没要求时期望 `false`；用户要求且本批满足标题加至少 2 张新卡时期望 `true`。与期望不符，不得报 `result_verified=passed`。意外的 `true` 是 remnote-direct 回归。

细节命令见 [卡片集群](card-clusters.md)。

## 写入前 preflight

`--apply` 之前，准备回执里要有与当前 `draft_digest` 一致的 `preflight`。每个门禁只能是 `pass` 或 `fail`。有任一 `fail`、缺对象、或摘要对不上，就不得写入。

```yaml
preflight:
  draft_digest: <与 preparation_receipt.draft_digest 相同>
  term_search: pass
  atomics_first: pass
  short_titles: pass
  aliases_after_create: pass
  rem_ref_title_ids: pass
  star_labels_exact: pass
  arrow_scan: pass
  arrow_hits: 0
  theme_literal: pass
  tag_path: pass
  no_spec_unless_asked: pass
  card_cluster: pass
  named_methods: pass
  redo_reuse: pass
  card_cluster_requested: false
  card_cluster_plan: omit
  source_h1: 前言
  theme_title: 2026-10-06 book 前言
  tag_ancestor_path: [HOME, 03-RESOURCES, 金融知识, 人人都是量化交易员]
```

`card_cluster_plan` 取 `omit`、`false` 或 `true`。`arrow_hits` 是本批普通学习卡正面和答案行里，上文箭头符号的个数；大于 0 则 `arrow_scan` 为 `fail`。不是单篇时 `source_h1` 为 `n/a`。原文模式不填儿童口吻门禁，对应项写 `pass`，并在回执注明原文保真。

不是方法论文章时 `named_methods` 为 pass。用户没有要求重做时 `redo_reuse` 为 pass。

## 失败条件

下面任一条成立，普通学习卡不得报草稿通过，也不得 `--apply`。原文导入和原文摘录不因为短标题、别名、STAR、箭头、原子卡顺序、方法原子卡或 kid_clarity 失败；它们按制卡 Skill 的原文规则逐段保真。原文批次如果使用多卡主题，`card_cluster` 仍然适用。

- `term_search`：新建术语卡没有 `search --cards-only` 对同一记忆目标的 miss；复用没有标题卡 Rem ID；Wiki 页或空 cards 被记成 hit。
- `atomics_first`：组合卡用到的术语没有先于该组合卡的原子卡；已复用的标题卡算已有。
- `short_titles`：原子卡标题含 `？` 或 `是什么`，或不是短名。
- `aliases_after_create`：长搜索句被写成卡片标题或另一张卡，或在标题卡创建之前就去建别名。
- `rem_ref_title_ids`：组合卡引用的 Rem 不是标题卡 ID，包括别名 Rem、字面括号、`pageKey` 或编造 ID。
- `star_labels_exact`：组合卡缺少逐字 STAR 标题；T 正文不是以「你要学会」开头；A 不是小数目演算；R 不是一句口诀；用 STAR 写原子卡；用三行定义背面写组合卡；原子卡背面不是一行正式定义、一行白话类比、可选一行例子。
- `arrow_scan`：普通学习卡正面或答案行含箭头；`arrow_hits` 不是 0。
- `theme_literal`：单篇或单章的 `theme_title` 不是 `YYYY-MM-DD <来源类型> <source_h1>`，`source_h1` 不是文章原 H1，或二者不一致。
- `tag_path`：Tag 是作者名、只停在粗 Folder，或把 Wiki 页当 Tag；没有祖先路径证明。
- `no_spec_unless_asked`：用户未要求时出现 `spec` 或策略规格说明书卡。
- `card_cluster`：用户未要求时计划为 `true`；用户要求且本批有主题标题和至少 2 张新卡时计划不是 `true`；用户要求但张数或标题不满足时仍写了 `true`。
- `named_methods`：方法论文章没有有名字的方法原子卡。
- `redo_reuse`：用户要求重做时复用了该来源的旧卡。
- `kid_clarity`：审卡回执缺五问，或任一答 `no`。

## 审卡回执

独立只读审卡对每张普通卡填写 `kid_clarity`。五个键都要在，值只能是 `yes` 或 `no`，全部是 `yes` 才算这张卡通过：

- `one_goal`：能不能用一句话说出这张卡只记一件事？
- `one_pass`：目标读者读一遍就懂，不用停下来解码？原子卡的白话类比按大约 8–10 岁、没有该领域背景；组合卡的问题和每一段按识字的大约 5 岁。
- `concrete`：有摸得到的人物或物件，而不是抽象倍数说法？组合卡的 S 还要点名哪里容易看错。
- `plain_words`：白话行没有本行没讲清的术语？原子卡看类比那一行；组合卡看 S、T、A、R 正文。
- `no_arrow`：正面和答案行都没有箭头？

每张组合卡再列 `star_headings`，四条字符串与「STAR 原文」逐字相同。

```yaml
reviewer_receipt:
  card_keys: [pe, pe-or-pb]
  kid_clarity:
    - {card_key: pe, one_goal: yes, one_pass: yes, concrete: yes, plain_words: yes, no_arrow: yes}
  star_headings:
    - card_key: pe-or-pb
      headings:
        - S — Situation：背景
        - T — Task：任务／目标
        - A — Action：行动
        - R — Result：结果
```

作者自证不能代替这份回执。覆盖的 `card_keys` 必须与 `preparation_receipt.card_keys` 逐项相同，失败项为空。

## 复盘

`retrospective.this_run_violations` 列出本轮曾经失败的门禁 id；没有则为空列表。意外的 `cc`（用户没要求，`cluster_powerup` 仍为 true）是本轮发现：证据是主题 Rem ID 和 `object-has-powerup ["cc"]` 的结果；机制是 remnote-direct 的 `flashcards apply` 回归；建议改动指向 remnote-direct；验证是同一查询回到与 opt-in 一致的值。

## 历史

2026-10-06，主题 `2026-10-06 book 前言`（Rem `C4lWZBYMbeUIoL9Zd`）在计划省略 `cardCluster` 时仍得到 `cc=true`。那是当时旧路径的失败记录，不是当前 Direct 的行为。当前行为见上文「Card Cluster」和 [ADR-0129](../../../../../docs/adr/0129-lightweight-remnote-learning-cards.md)。

干跑对照：[市盈率](example-pe-valuation.md)、[前言](example-book-preface.md)。两份都不是已经写入的回执。
