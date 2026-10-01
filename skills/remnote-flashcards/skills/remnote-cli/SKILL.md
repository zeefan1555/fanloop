---
name: remnote-cli
description: 在本项目中使用本地 remnote-cli 查询、写入、排障或调用 RemNote Plugin SDK。TODO、Wiki、制卡和界面操作仍由对应专项技能负责。
---

# RemNote CLI

这是本项目自维护的 `remnote-cli` 操作手册。以本机已安装命令的 `--help` 和真实返回为准，不加载或转交给工具链仓库中的其他 Skill。

## 开始前

1. 写操作前运行 `remnote-cli status`，确认 Bridge、Server、CLI 版本兼容、连接正常且写入已开启。
2. 首次使用某个命令时先运行对应 `--help`，不要凭记忆猜参数。
3. 导航和多步处理默认使用 JSON，保留 Rem ID；只做单次人类阅读时才用 `--text`。

用户当前请求已经明确表达创建、修改、移动、删除或迁移意图时，可以直接执行对应写操作，不要求用户再输入固定确认口令。请求只是查询、分析、审阅或方案讨论时保持只读。删除、整体替换和批量迁移必须先核对目标 Rem ID、影响范围和引用关系；语义不明确时再向用户确认具体动作。

## 命令选择

- 优先使用稳定的任务命令：`status`、`search`、`read`、`search-by-tag`、`review-stats`、`outline`、`todo`、`create`、`update`、`insert-children`、`replace-children`、`update-tags`、`set-document-status` 和 `journal`。
- 一级命令没有覆盖时，查看 `remnote-cli sdk --help`、命名空间帮助和具体方法帮助，再调用公开 SDK 能力。
- 不调用私有 `_call`、任意对象路径或 JavaScript 求值；只使用 `sdk capabilities` 中公开且受支持的方法。
- 每次 shell 执行只运行一个 `remnote-cli` 命令，避免命令链让写入边界和错误来源变得不清楚。

## 渐进式读取

- 先用 `search` 定位，再按精确 Rem ID `read`。
- 树形内容从 `--depth 1 --child-limit 50 --content-mode structured` 开始；只有相关内容缺失或返回明确截断时才增加深度。
- 在已知根节点下查询时使用 `--parent-id <REM_ID>`，不要为了找一个概念深读整个知识库。
- 需要继续导航时保留 `remId`、`parentRemId`、aliases、引用和分页游标；不要只保存渲染后的 Markdown。

## 常用读取

```bash
remnote-cli status
remnote-cli search "<query>" --view full --ancestor-depth 3 --depth 1
remnote-cli search "<query>" --parent-id <ROOT_ID> --view full --depth 1
remnote-cli read <REM_ID> --content-mode structured --view full --depth 1 --child-limit 50
remnote-cli search-by-tag --tag-id <TAG_ID> --result-mode tagged --view full
remnote-cli review-stats <REM_ID>
remnote-cli review-stats --today
remnote-cli sdk capabilities --status supported --text
```

## 常用写入

- 新建：`create`；需要正文时优先使用 `--content-file`。
- 修改标题或 aliases：`update`。
- 追加子 Rem：`insert-children`。
- 整体替换直接子 Rem：`replace-children`。只在用户明确要求替换，且 `status` 显示 `acceptReplaceOperation=true` 时使用。
- 设置 Document：先运行 `set-document-status` 预览，再使用 `--apply`；尽量传 `--expected-old-rem-type`。
- 标签增删：`update-tags`，只传精确 Tag Rem ID。
- 移动、删除或其他一级命令未覆盖的操作：先查看对应 `sdk` 方法帮助，再按精确目标 ID 调用。
- 支持 dry-run 的命令先预览；写入后立即按原 Rem ID 回读，不能只看命令返回成功。
- `create`、`insert-children`、`replace-children` 和 `journal` 创建的非叶子 Rem 会由 Bridge 在最近的 Document、每日笔记或 Portal 上下文中自动折叠；`outline collapse` 只用于既有内容修复或显式批量调整。

Markdown 写入中的精确原生引用使用 `[[id:<REM_ID>]]`。多行内容使用临时 UTF-8 文件和 `--content-file`，避免 shell 转义破坏结构。

## 写入不变量

- 更新已有内容时优先保留原 Rem ID、位置、aliases、引用和卡片复习历史。
- 重试写入前先回读，确认上一次是否已经部分成功，避免重复创建。
- 移动页面前记录原父级；移动后回读新父级和外部引用。
- 删除前检查引用方和替代目标。需要保留历史上下文时优先归档，不直接删除。
- 不直接读取或修改 RemNote 应用数据目录 数据目录。

## 专项路由

- TODO：使用 `remnote-todo`。
- Folder、HOME 分类 Tag 与层级：使用 `remnote-structure`。
- Wiki：使用项目 `remnote-wiki`。
- 笔记与闪卡：使用 `remnote-flashcards`；其中可复用概念由它调用 `remnote-wiki`。
- CLI 无法完成的原生界面状态：使用 `control-remnote`。

专项 Skill 决定内容结构和业务步骤；本 Skill 只规定 CLI 的发现、调用、写入边界和回读方式。

## 故障处理

1. 先运行 `remnote-cli status`，区分 Server 不可达、Bridge 未连接、版本不兼容和写入开关关闭。
2. Bridge、Server、CLI 应处于相同 `0.x` minor，优先使用完全一致版本。
3. Bridge 未连接时，确认 RemNote 已打开并等待自动重连；仍失败时再按 `control-remnote` 检查插件面板。
4. SDK 调用失败时，重新查看 `sdk capabilities` 和具体方法 `--help`，报告真实错误，不伪造降级结果。
5. 只有连接和版本恢复后才重试原操作；重试前先回读目标，防止重复写入。
