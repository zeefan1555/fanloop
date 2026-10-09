# 干跑示例：前言 / Tag 人人都是量化交易员

这是给 Agent 对照的导入草稿，不是一次已经写入 RemNote 的回执。来源是 https://xingwudao.github.io/xquant-beginner/book/preface 。没有真实写入回执时不要声称已经写卡。卡型规则见[学习卡契约](learning-card-contract.md)。

## 本批规则

- 领域分组：`HOME / 03-RESOURCES / 金融知识`
- 打在批次上的最细 Concept Tag：`人人都是量化交易员`。这是教程名。不要打作者名 `刑无刀`，不要停在 `金融知识`，也不要把 Wiki 页当 Tag。
- 这一页的 H1 是「前言」。`source_h1` 就是 `前言`。`theme_title` 必须是 `2026-10-06 book 前言`。写成「术语与方法」或「量化地图」时 `theme_literal` 失败，不得写入。
- 主题放在当日每日笔记。术语卡在主题下。组合卡是主题的最后一个子 Rem，自己不挂教程 Tag。主题的 Tag 索引这些卡。
- 用户没有要求 Card Cluster。计划省略 `cardCluster`（写 `false` 也可以）。当前 Direct 这两种都不会添加 `cc`。写后 `cluster_powerup` 必须为 false。若为 true，记为 remnote-direct 回归。
- 不写 WIKI，也不在卡片里放 Wiki 链接。用户没有要求。Wiki 概念页也不能代替术语闪卡。离开准备步骤靠 `terms_ready` 和 `learning_preflight_passed`，不靠 Wiki 不适用回执。
- 创建 `回测`、`四个阶段`、`做看疑` 之前，各自运行 `remnote-cli search "<短名>" --cards-only --include-review-stats --view full`。本草稿假定三次都没有命中同一记忆目标的术语闪卡，所以新建。若命中且够用，引用那张标题卡的 Rem ID，不新建。
- 用户没有要求重做，也没有要求 `spec` 或「策略规格说明书」。
- 这是方法论文。原子卡里既有术语 `回测`，也有有名字的方法 `四个阶段` 和 `做看疑`。本批 4 张。顺序、短标题、STAR 原文、箭头和儿童口吻按契约。

## 干跑 preflight

```yaml
preflight:
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

`2026-10-06 book 术语与方法` 配 `source_h1: 前言` 是失败，不能 `--apply`。审卡回执的五问和四条 STAR 标题与市盈率示例同一形状。

```yaml
term_pipeline:
  - {query: 回测, cards_only: true, outcome: create, term_key: backtest, title_rem_id: pending}
  - {query: 四个阶段, cards_only: true, outcome: create, term_key: four-stages, title_rem_id: pending}
  - {query: 做看疑, cards_only: true, outcome: create, term_key: do-look-doubt, title_rem_id: pending}
combo_rem_refs:
  - {card_key: backtest-doubt, refs: [backtest, do-look-doubt, four-stages]}
```

`pending` 表示这张标题卡还没有 Rem ID。术语卡先 apply。组合卡计划不写 `title`，答案里的 `{"remId"}` 用回读到的标题卡 ID。不要把 `term_key` 写成 `{"remId"}`。组合卡 apply 之后用 `move-note` 移成主题的最后一个子 Rem，再用 `object-remove-tag` 去掉它自己的 Tag。不要把组合卡留在每日笔记上。这些 ID 不是 Wiki 页面，也不是别名 Rem。别名命令是 `remnote-cli sdk rem object-get-or-create-alias-with-text --target-id <ID> --args-json '[["<alias>"]]'`，每张标题卡一条。`update --add-aliases` 取决于 SDK 版本；SDK 0.0.46 不可用。

别名在标题卡创建之后再加上。`回测是什么？` 这种长句不是标题。组合卡只引用标题卡。下面草稿里的 `[[回测]]` 是显示结果，还没有真实 Rem ID。

- `回测` 的别名：`回测是什么？`
- `四个阶段` 的别名：`四个阶段是什么？`
- `做看疑` 的别名：`做、看、疑是什么？`

## 阅读草稿

下面的 Markdown（`问题 ===` 加缩进子 bullet）只帮助看背面形状。它不是 `flashcards apply` 读的计划。计划是 `question` 字符串加 `answer` 的 `Segment[][]`，字段见 [FlashcardsPlan](../../remnote-cli/references/flashcards-plan.md)。`title` 为 `2026-10-06 book 前言` 的术语卡计划新建主题。组合卡不在这份带 `title` 的计划里。段间看起来空的 Rem 含一个 `U+200B`。原子术语卡没有这些空行，也没有 S/T/A/R 标题。

## FlashcardsPlan 形状

占位符。下面带 `title` 的计划只有术语卡。组合卡是另一次不写 `title` 的计划，答案里的 `{"remId"}` 用已回读的标题卡 ID。apply 之后 `move-note` 到主题末尾，再 `object-remove-tag`。

```text
{
  schemaVersion: 1,
  dailyRemId: "<今日笔记 Rem ID>",
  homeRootId: "<HOME Rem ID>",
  tagRemId: "<人人都是量化交易员 Concept Rem ID>",
  title: "2026-10-06 book 前言",
  newCards: [
    {
      key: "backtest",
      question: "回测",
      answer: [
        ["正式定义：回测是用过去已经发生的记录，按写好的买卖规则重走一遍，看当时会赚还是亏。"],
        ["白话类比：小明拿出上周的天气本，按「晴天才摆摊」重走一遍，看那周会赚几块。"]
      ]
    }
  ]
}
```

## 阅读草稿正文

```markdown
- 2026-10-06 book 前言
    - 回测 ===
        - 正式定义：回测是用过去已经发生的记录，按写好的买卖规则重走一遍，看当时会赚还是亏。
        - 白话类比：小明拿出上周的天气本，按「晴天才摆摊」重走一遍，看那周会赚几块。
        - 例子：上周晴了 4 天，每天赚 2 块，4 乘 2 等于 8 块。
    - 四个阶段 ===
        - 正式定义：一个想法要走四步：先圈出看哪些东西，再写成买卖规则，再真的去买卖，最后问为什么赚或亏。
        - 白话类比：小明卖柠檬水，先选出看哪三罐，再写下怎么买怎么卖，再真的去买，最后问赚的 2 块是哪一步来的。
        - 例子：3 罐先看 1 罐，3 减 1 等于还剩 2 罐。
    - 做看疑 ===
        - 正式定义：每一步都走同一条链。做是把想法写成能检验的规则，看是读做完的结果，疑是检查结果是不是碰巧。
        - 白话类比：小明先写下「晴天才摆摊」，再看这周赚了 6 块，再问是不是只碰上了连续晴天。
        - 例子：这周赚 6 块，大太阳带来 4 块，6 减 4 等于还剩 2 块。下雨的一周只赚 1 块。
    - 刚学完 [[回测]] 和 [[做看疑]]：用上周的天气本重走，赚了 8 块。就能说这个规矩以后也会赚吗？ ===
        - S — Situation：背景
        - 小明用上周的天气本重走「晴天才摆摊」。晴了 4 天，每天 2 块，一共 8 块。容易看错的地方，是看见这 8 块就说规矩以后也会赚。
        - ​
        - T — Task：任务／目标
        - 你要学会：用旧记录重走出 8 块以后，还要问是不是碰巧，别把这一周的 8 块当成以后都会赚。
        - ​
        - A — Action：行动
        - 晴天 4 天，每天 2 块，4 乘 2 等于 8。下雨的 4 天，每天 1 块，4 乘 1 等于 4。8 减 4 等于 4，少掉的 4 块是天气带来的。点开 [[回测]] 看用旧记录重走，点开 [[做看疑]] 看最后那个「疑」，点开 [[四个阶段]] 看还要问赚的钱是哪一步来的。
        - ​
        - R — Result：结果
        - 口诀：赚到 8 块，还要再疑一次。
```
