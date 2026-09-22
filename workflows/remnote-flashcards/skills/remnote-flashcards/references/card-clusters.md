# 卡片集群

官方说明：[Card Clusters](https://help.remnote.com/en/articles/10104223-card-clusters)。

## 何时使用

- 同一学习链路生成 2–7 张卡片时，把卡片作为同一个顶层主题 Rem 的直接子 Rem，并给该主题添加 RemNote 内置的 Card Cluster Power-up。
- 超过 7 张卡片时，先按独立学习链路拆成多个主题；不要仅为凑数量拆分，也不要把大量弱相关卡片塞进一个集群。
- 单张卡不添加 Card Cluster。
- Card Cluster 是内置 Power-up，不是名为“卡片集群”或 `Card Cluster` 的普通自建标签；不得创建同名 Tag 代替它。
- 顶层主题必须保持普通 `text` Rem。Card Cluster 不要求 Document 状态，不得把主题转成 Document、固定或加入侧边栏。

## 添加方式

当前 `remnote-cli 0.17.0` 的搜索和回读不会可靠暴露 Card Cluster Power-up 的 ID。先用 CLI 创建并核验主题及子卡，再通过 RemNote 界面添加内置 Card Cluster；若后续 CLI 明确支持 Power-up ID，则可改用该能力，但必须复用内置实体。

安全操作顺序：

1. 操作前用 CLI 回读目标，确认标题准确、`remType` 是 `text`，并记录 Rem ID。
2. 在界面中缩放进入这个主题。不要点击标题栏的星标、固定或“转换为文档”；固定普通 Rem 可能同时把它转换成 Document。
3. 聚焦主题标题，输入 `/` 打开插入菜单，再输入 `cluster`。只有菜单明确显示“高级 / 卡片集群 / off”时，才点击“卡片集群”开关。
4. 如果 `/cluster` 只是留在标题里、菜单没有出现，不要按回车；先删掉这段字面文本，再重新打开插入菜单。
5. 开关生效后，立即确认标题恢复原样、页面显示“卡片集群”、主题未出现在 `Pinned`，再用 CLI 回读并确认 `remType` 仍为 `text`。

若主题意外被固定或转成 Document，先重新回读当前状态。仅当当前 `remType` 确实是 `document` 时，先在界面取消固定，再执行 `set-document-status <TOPIC_REM_ID> --no-document --apply --expected-old-rem-type document`；若已经是 `text`，不要重复执行恢复命令。恢复后重新确认 Card Cluster 仍存在。

添加后，顶层主题同时拥有：

- 一个最具体的 HOME 分类 Tag，用于归档与按主题复习。
- Card Cluster Power-up，用于在复习单张子卡时显示同组卡片的上下文。

子卡不重复添加分类 Tag 或 Card Cluster。

## 验证

1. 在 RemNote 界面确认顶层主题显示“卡片集群”。CLI `read` 可能只返回普通分类 Tag，不能据此判定 Power-up 缺失。
2. 用 CLI 回读并确认顶层主题的 `remType` 是 `text`，标题没有残留 `/cluster`，并且主题未出现在侧边栏的 `Pinned`。缩放查看时暂时出现在 `Unpinned` 不等于被固定。
3. 只有用户明确要求验证复习体验，或 Power-up 状态异常时，才打开专属复习队列。
4. 进入队列时不显示答案、不评分；确认同组卡以灰色上下文或空白占位显示后立即退出，避免改变复习记录。

界面不可用或无法看到集群上下文时，不宣称完成；保留已创建卡片并报告 Card Cluster 验证阻塞。

本规则默认只用于新建主题和本次明确修改的主题，不主动扫描或批量改造历史卡片。
