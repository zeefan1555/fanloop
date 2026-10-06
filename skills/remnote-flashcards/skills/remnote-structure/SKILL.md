---
name: remnote-structure
description: 查找、创建或修正 RemNote 的 Folder 与分类 Tag。用于维护 HOME/WIKI 层级、判断一个 Rem 应是目录还是标签、创建分类节点、排查重复或错放节点；不负责笔记正文、闪卡内容或 Wiki 页面写作。
---

# RemNote Folder 与 Tag

维护 RemNote 的组织结构，同时保留稳定 Rem ID、父子层级和已有引用。核心判断是节点的真实职责，不是它位于第几层或是否有子节点。

## 类型模型

- **Folder**：只负责容纳和组织下一层节点，本身不作为内容分类标签。
- **Concept Tag**：实际用于给笔记、文章或闪卡打 Tag 的 Rem。Concept 可以有子标签，不能因有子节点就改成 Folder。
- **普通 Rem**：正文、Wiki 知识页面、说明和其他内容，不因位于目录下就改类型。
- **Document**：真实文档页面。Folder 在 SDK 中可能同时返回 `isDocument=true`；以 `isFolder=true` 和 CLI 的 `remType=folder` 为准，不再清除 Folder 的 Document 能力。

项目约定：`HOME`、`WIKI` 和纯分组目录使用 Folder；HOME 中实际分类 Tag 使用 Concept；新闪卡实体写入今日笔记，并挂 HOME 分类 Tag。WIKI 的知识页面保持普通 Rem，`SCHEMA.md`、`index.md`、`log.md` 保持 Document，整个 WIKI 不生成闪卡。

## 开始前

1. 读取当前 Step Prompt 和本 Skill；组织类型以本文类型模型为准。
2. 写操作前单独运行 `remnote-cli status`，要求三端版本兼容、Bridge 已连接且写入开启。
3. 首次调用命令时查看对应 `--help`。每次 shell 执行只运行一个 `remnote-cli` 命令。

## 查找

1. 用 `search` 找到唯一顶层 `HOME` 或 `WIKI`，以没有父级且标题精确匹配为准。
2. 从根开始用 `read --content-mode structured --view full` 浅读层级；已知父级时用 `search --parent-id` 限定范围。
3. 同名候选按父级路径、Rem ID、`remType`、aliases 和用途消歧，不能只按标题选择。
4. 判断候选是否是真实 Tag 时，运行：

```bash
remnote-cli search-by-tag --tag-id <REM_ID> --result-mode tagged --limit 1 --content-mode none --view compact
```

5. 结合以下证据判型：
   - 有 tagged Rem、查询 Portal 或明确标签职责，例如 TODO、DONE、文章：Concept。
   - 只承载下一层分类，没有直接标签用途：Folder。
   - 现有明确类型和用户命名意图优先；没有命中不等于一定不是 Tag，未使用的新标签仍可为 Concept。

完成条件：候选的唯一 Rem ID、父级路径和 Folder／Tag 职责都已确定；仍有同名歧义时不写入。

## 位置路由

- HOME：保存 Folder 与 Concept Tag；分类关系通过打 Tag 建立。
- WIKI：只保存固定目录、元文档、原始来源和普通知识页面。闪卡可以引用 Wiki 页面，但不得成为 WIKI 子树的一部分。
- 每日笔记：新闪卡、主题、普通笔记和文章的实体写入今日笔记。单卡给自身打 HOME Tag。多卡放在主题下时只给主题打 Tag，子卡通常不再打。学习批次使用领域分组下最细的 Concept Tag，例如 `HOME / 03-RESOURCES / 金融知识 / 估值`，不要停在更粗的 Folder。持续课程的这个 Concept 用教程名，例如 `人人都是量化交易员`，不用作者名 `刑无刀`。
- 创建前分别确定今日笔记父级与 HOME Tag ID；位置和标签是两个独立检查项，创建后调用 `remnote-verify`。
- 审计 WIKI 的卡片状态调用 `remnote-verify`：知识页面误开卡片时原位清除方向；真正错放的学习卡移入每日笔记并打 HOME Tag，保留原 ID。

## 创建或修正 Folder

1. 在目标父级内查重；存在同职责节点时直接复用。
2. 缺失时用 `create --parent-id <PARENT_ID>` 创建普通 Rem，并记录返回 ID。
3. 新建普通 Rem 直接设置 Folder；把现有 Concept 改回纯分组节点时，先恢复默认 Rem 类型，再设置 Folder：

```bash
remnote-cli sdk rem object-set-type --target-id <REM_ID> --args-json '["DEFAULT_TYPE"]'
remnote-cli sdk rem object-set-is-folder --target-id <REM_ID> --args-json '[true]'
```

不要在 `setIsFolder(true)` 后调用 `setIsDocument(false)`；RemNote 可能连带清除 Folder。不要删除重建节点。

完成条件：`read` 返回 `remType=folder`，父级正确，原子回读 `object-is-folder` 为 `true`，已有子树和引用仍在原 Rem ID 上。

## 创建或修正 Concept Tag

1. 在预期父级和 HOME 根范围内分别按规范名、别名查重；已有同义 Tag 时复用，不创建同名副本。
2. 缺失时用 `create --parent-id <PARENT_ID>` 创建普通 Rem，再设置 Concept：

```bash
remnote-cli sdk rem object-set-type --target-id <REM_ID> --args-json '[1]'
```

3. 把误设为 Folder／Document 的现有节点改为 Concept 时，按顺序原位转换：

```bash
remnote-cli sdk rem object-set-is-folder --target-id <REM_ID> --args-json '[false]'
remnote-cli sdk rem object-set-is-document --target-id <REM_ID> --args-json '[false]'
remnote-cli sdk rem object-set-type --target-id <REM_ID> --args-json '[1]'
```

完成条件：`read` 返回 `remType=concept`，`object-is-folder=false`、`object-is-document=false`，父级和 Rem ID 不变；需要实际打 Tag 时，再由调用方用精确 Rem ID 建立标签关系并通过 `search-by-tag` 回读。

## 批量整理

- 先生成“保留／改 Folder／改 Concept”清单，再逐项执行；父子关系与类型是两个维度，不移动节点来代替改类型。
- 每个节点独立取证。禁止用“所有叶子都是 Tag”“某层以下全是 Tag”“有子节点就是 Folder”等规则批量推断。
- 已有 tagged Rem 的节点不得改成 Folder，除非用户明确要求同时迁移所有标签关系。
- WIKI 目录按 `remnote-wiki` 固定结构判断；WIKI 页面不因为名称是术语就转成 Concept。
- WIKI 中不得创建或保留闪卡；HOME 中的分类路径由 Folder 和 Concept 共同组成，不能用 Wiki 页面代替分类 Tag。
- 写后重新读取根节点到足够深度，确认没有平铺、重复、错父级或类型漂移。

## 调用关系

- `remnote-flashcards` 选择或创建 HOME 分类时调用本 Skill。
- `remnote-wiki` 创建或修复 WIKI 目录时调用本 Skill；页面内容与索引仍由 `remnote-wiki` 负责。
- 底层命令发现、连接和故障处理遵循 `remnote-cli`；需要新增 CLI 能力时报告 blocked，工具链改造不属于本 Workflow。
