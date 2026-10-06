# 卡片集群

官方说明：[Card Clusters](https://help.remnote.com/en/articles/10104223-card-clusters)。

普通学习卡何时添加 `cc`，只按[学习卡契约](learning-card-contract.md)。本文只保留回读命令。

## 规则

- 用户没有明确要求时，计划省略 `cardCluster` 或写 `false`。两种都不添加 `cc`。
- 用户明确要求，且计划有主题标题、新卡至少 2 张时，计划写 `cardCluster: true`。只有这时才添加 `cc`。
- 主题 `title` 和 2–7 张卡都不是开关。不要调用 `object-add-powerup` 补 `cc`。
- 多张卡可以放在一个普通 `text` 主题 Rem 下，这只是为了阅读。主题 Rem 本身不是 Card Cluster。
- 单张卡不添加 Card Cluster。
- Card Cluster 是内置 Power-up，不是名为“卡片集群”或 `Card Cluster` 的普通自建标签；不得创建同名 Tag 代替它。
- 顶层主题必须保持普通 `text` Rem。不得把主题转成 Document、固定或加入侧边栏。
- 本规则只用于用户本次明确要求修改的主题，不扫描或批量给历史卡片补 `cc`。

## 回读

内置 Power-up code 是 `cc`。普通 `tags` 或标签搜索不能判断它是否存在。

写后运行：

```bash
remnote-cli sdk rem object-has-powerup --target-id <主题Rem_ID> --args-json '["cc"]'
```

把结果写入 `flashcards_written.cluster_powerup`。用户没要求时期望 `false`。用户要求且本批满足标题加至少 2 张新卡时期望 `true`。与期望不符，本批失败。意外的 `true` 是 remnote-direct 回归，不要在本 Workflow 里补写或删改来掩盖。

再按同一 ID 回读主题及子卡，确认标题、父级、普通 `text` 类型、HOME Tag 和卡片集合保持正确。子卡不重复添加 Tag 或 `cc`。

整个检查使用 CLI。命令失败或返回不完整时报告具体阻塞；重试先回读。只有用户明确要求检查复习界面时，才使用 `control-remnote` 查看队列；不揭示答案、不评分。

## 历史

2026-10-06 有一次计划省略 `cardCluster` 仍得到 `cc=true`。那不是当前 Direct 的行为。见契约的历史节和 ADR-0129。
