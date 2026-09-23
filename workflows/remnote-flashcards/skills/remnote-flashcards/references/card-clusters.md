# 卡片集群

官方说明：[Card Clusters](https://help.remnote.com/en/articles/10104223-card-clusters)。

## 何时使用

- 同一学习链路生成 2–7 张卡片时，把卡片作为同一个顶层主题 Rem 的直接子 Rem，并给该主题添加 RemNote 内置的 Card Cluster Power-up。
- 超过 7 张卡片时，先按独立学习链路拆成多个主题；不要仅为凑数量拆分，也不要把大量弱相关卡片塞进一个集群。
- 单张卡不添加 Card Cluster。
- Card Cluster 是内置 Power-up，不是名为“卡片集群”或 `Card Cluster` 的普通自建标签；不得创建同名 Tag 代替它。
- 顶层主题必须保持普通 `text` Rem。Card Cluster 不要求 Document 状态，不得把主题转成 Document、固定或加入侧边栏。

## 添加与回读

Card Cluster 的内置 Power-up code 是 `cc`；没有独立的标签 Rem ID，普通 `tags` 或标签搜索不用于判断它是否存在。

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

4. 按同一 ID 回读主题及子卡，确认标题、父级、普通 `text` 类型、HOME Tag 和卡片集合保持正确。仅给主题添加 HOME Tag 与 `cc`，子卡不重复添加。

整个添加与结构验证使用 CLI；无需电脑操作、查找 Power-up Rem ID 或创建同名 Tag。命令失败或返回不完整时报告具体阻塞；重试先回读，不以打开界面代替真实成功回执。

只有用户明确要求检查复习界面体验时，才使用 `control-remnote` 查看队列；不揭示答案、不评分。
本规则只用于新建主题和本次明确修改的主题，不扫描或批量改造历史卡片。
