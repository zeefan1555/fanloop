# 干跑示例：市盈率 PE / Tag 估值

这是给 Agent 对照的导入草稿，不是一次已经写入 RemNote 的回执。没有真实写入回执时不要声称已经写卡。卡型规则见[学习卡契约](learning-card-contract.md)。

## 本批规则

- 领域分组：`HOME / 03-RESOURCES / 金融知识`
- 打在批次上的最细 Concept Tag：`估值`。不是作者名，不是粗 Folder，也不是 Wiki 页。
- 实体放在当日每日笔记。多张卡放在一个普通 `text` 主题 Rem 下，便于阅读。
- 这批不是单篇文章，`source_h1` 为 `n/a`。主题用 `resource` 加概念名。单篇文章或单章见 [前言文章干跑示例](example-book-preface.md)。
- 只给主题打 `估值`。下面五张子卡不再打这个 Tag。
- 用户没有要求 Card Cluster。计划省略 `cardCluster`（写 `false` 也可以）。当前 Direct 这两种都不会添加 `cc`。写后 `object-has-powerup ["cc"]` 必须为 false，并记入 `cluster_powerup`。若为 true，记为 remnote-direct 回归。
- 不写 WIKI，也不在卡片里放 Wiki 链接。用户没有要求。
- 顺序、短标题、别名、STAR 原文、箭头和儿童口吻都按契约。本批 5 张，落在偏好的 3–7 张里。原子卡标题是 `PE`、`EPS`、`PB`，不是带 `？` 或「是什么」的长句。

## 干跑 preflight

用户没要求集群，也没有单篇 H1。写入前对象应类似：

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
  source_h1: n/a
  theme_title: 2026-10-06 resource 市盈率 PE
  tag_ancestor_path: [HOME, 03-RESOURCES, 金融知识, 估值]
```

审卡回执对每张卡五问都是 yes。组合卡的 `star_headings` 逐字是契约里的四条标题。下面只示范第一张原子卡和第一张组合卡，其余卡同一形状。

```yaml
kid_clarity:
  - {card_key: pe, one_goal: yes, one_pass: yes, concrete: yes, plain_words: yes, no_arrow: yes}
star_headings:
  - card_key: pe-gift
    headings:
      - S — Situation：背景
      - T — Task：任务／目标
      - A — Action：行动
      - R — Result：结果
```

别名在标题卡创建之后再加上，组合卡只引用标题卡。下面草稿里的 `[[PE]]` 是标题为 `PE` 的显示结果。草稿里还没有真实 Rem ID。

- `PE` 的别名：`市盈率（PE）是什么？`
- `EPS` 的别名：`每股收益（EPS）是什么？`
- `PB` 的别名：`市净率（PB）是什么？`

## 可导入结构

段间看起来空的 Rem 含一个 `U+200B`，用来在组合卡各段之间留出空行。原子术语卡没有这些空行，也没有 S/T/A/R 标题。

```markdown
- 2026-10-06 resource 市盈率 PE
    - PE ===
        - 正式定义：市盈率 = 股价 / 每股收益（EPS），表示买下公司时，付的钱是它一年赚的钱的多少倍。
        - 白话类比：柠檬水摊一年赚 10 块，你花 50 块把它买下来，等于一次付了它 5 年赚的钱。
        - 例子：一年赚 10 块、卖价 50 块，50 除以 10 等于 5。
    - EPS ===
        - 正式定义：EPS = 归属于普通股股东的净利润 / 发行在外的普通股股数。
        - 白话类比：四个小朋友一起卖柠檬水，一年一共赚 8 块，每人分到 2 块。
        - 例子：一共赚 8 块、4 个小朋友，8 除以 4 等于每人 2 块。
    - PB ===
        - 正式定义：市净率 = 股价 / 每股净资产，表示买下公司时，付的钱是它账上家当的多少倍。
        - 白话类比：柠檬水摊的杯子、桌子和罐子里的钱一共值 20 块，你花 40 块把它买下来，等于付了两份家当的钱。
        - 例子：家当值 20 块、卖价 40 块，40 除以 20 等于 2。
    - 刚学完 [[PE]]：小明的摊多了一笔生日礼，两个摊卖价都是 20 块。小明的摊是真的更便宜吗？ ===
        - S — Situation：背景
        - 小明和小红各有一个柠檬水摊，卖价都是 20 块。小红卖柠檬水赚了 4 块。小明也算出 4 块，可其中 2 块是奶奶给的生日礼，明年没有。两个摊看起来一样便宜，这就是容易看错的地方。
        - ​
        - T — Task：任务／目标
        - 你要学会：先问今年这 4 块明年还在不在，别把只来一次的生日礼当成每年都会赚的钱。
        - ​
        - A — Action：行动
        - 小红明年还会赚 4 块。卖价 20 块，20 除以 4 等于 5，等于一次付了 5 年赚的钱。小明今年也算出 4 块，所以也是 5，看起来一样便宜。明年生日礼没了，只剩卖柠檬水的 2 块。20 除以 2 等于 10，等于要付 10 年赚的钱。点开 [[PE]] 看付了几年的钱，点开 [[EPS]] 看钱怎么分到每一份。
        - ​
        - R — Result：结果
        - 口诀：只来一次的钱，别当成每年都有。
    - 刚学完 [[PE]] 和 [[PB]]：柠檬水摊今年倒贴了 2 块。还看赚的钱，还是改看杯子和桌子？ ===
        - S — Situation：背景
        - 小明的柠檬水摊卖价 20 块。今年没赚到，还倒贴了 2 块。杯子、桌子和罐子里的钱一共值 10 块。容易看错的地方，是摊在倒贴，还有人拿赚的钱去比谁便宜。
        - ​
        - T — Task：任务／目标
        - 你要学会：摊还在赚钱时看 [[PE]]；今年倒贴了，就改看 [[PB]]，别拿倒贴的数去比谁便宜。
        - ​
        - A — Action：行动
        - 卖价 20 块，今年倒贴 2 块。倒贴的钱不能拿来除，除完也不表示更便宜。杯子、桌子和罐子里的钱一共 10 块。20 除以 10 等于 2，等于付了两份家当的钱。点开 [[PE]] 看赚的钱，点开 [[PB]] 看家当。
        - ​
        - R — Result：结果
        - 口诀：倒贴的时候，别问赚了多少，改问家当值几份。
```
