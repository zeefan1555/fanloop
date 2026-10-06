# 卡片集群

官方说明：[Card Clusters](https://help.remnote.com/en/articles/10104223-card-clusters)。

Card Cluster 是可选项。默认学习路径不添加它。同一主题下有 2–7 张卡，也不是添加它的理由。

## 何时使用

- 只有用户明确要求 Card Cluster、卡片集群或内置 `cc` 时，才给主题 Rem 添加这个 Power-up。`flashcards apply` 只有在计划写了 `cardCluster: true` 时才应添加 `cc`；主题 `title` 不是这个开关。`--help` 还不接受 `cardCluster` 时，不要写这个未知字段。
- 用户没要求时不要添加。写后仍要查询。对主题运行下面的 `object-has-powerup`，结果必须为 false。返回 true 算本批失败，不能当成「缺少也不算失败」。已发布的 `flashcards apply` 会在多张新卡时自行加上 `cc`，计划省略 `cardCluster` 拦不住。根因和未推送修复见 [remnote-cli](../../remnote-cli/SKILL.md)。
- 多张卡可以放在一个普通 `text` 主题 Rem 下，这只是为了阅读。主题 Rem 本身不是 Card Cluster。放置和打 Tag 规则在[制卡 Skill](../SKILL.md)的「闪卡」一节，不在本文。
- 单张卡不添加 Card Cluster。
- Card Cluster 是内置 Power-up，不是名为“卡片集群”或 `Card Cluster` 的普通自建标签；不得创建同名 Tag 代替它。
- 顶层主题必须保持普通 `text` Rem。Card Cluster 不要求 Document 状态，不得把主题转成 Document、固定或加入侧边栏。
- 本规则只用于用户本次明确要求修改的主题，不扫描或批量给历史卡片补 `cc`。

## 添加与回读

用户明确要求之后才执行本节。Card Cluster 的内置 Power-up code 是 `cc`；没有独立的标签 Rem ID，普通 `tags` 或标签搜索不用于判断它是否存在。

1. 遵循 `remnote-cli` 的写前 status 与命令帮助要求；回读本批次主题 Rem，确认真实 ID、标题、父级和 `remType=text`。
2. 先查询集群状态；已经为 `true` 就复用，返回 `false` 才添加：

```bash
remnote-cli sdk rem object-has-powerup --target-id <主题Rem_ID> --args-json '["cc"]'
remnote-cli sdk rem object-add-powerup --target-id <主题Rem_ID> --args-json '["cc"]'
```

3. 添加后运行只读命令，必须确认返回 `true`：

```bash
remnote-cli sdk rem object-has-powerup --target-id <主题Rem_ID> --args-json '["cc"]'
```

4. 按同一 ID 回读主题及子卡，确认标题、父级、普通 `text` 类型、HOME Tag 和卡片集合保持正确。仅给主题添加 HOME Tag；用户要求集群时才同时添加 `cc`。子卡不重复添加 Tag 或 `cc`。

整个添加与结构验证使用 CLI；无需电脑操作、查找 Power-up Rem ID 或创建同名 Tag。命令失败或返回不完整时报告具体阻塞；重试先回读，不以打开界面代替真实成功回执。

只有用户明确要求检查复习界面体验时，才使用 `control-remnote` 查看队列；不揭示答案、不评分。
