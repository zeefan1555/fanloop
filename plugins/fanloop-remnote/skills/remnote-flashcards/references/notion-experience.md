# remnote workflow 经验表

`retrospect_run` 把一条可复用的复盘发现写进这张已经存在的 Notion 多维表。不新建第二套维护清单，不在复盘里改 Workflow、验收条件或 CLI。

触发：`retrospective_recorded` 的 `findings` 至少有一条，且该条具备契约要求的证据、机制、建议改动、验证方法。`findings` 仍是 0..1。零发现不调用 `ntn`。

## 默认目标

| 项 | 值 |
| --- | --- |
| 标题 | remnote workflow 经验表 |
| Database ID | `00000000-0000-4000-8000-000000000001` |
| Data source ID | `00000000-0000-4000-8000-000000000002` |
| URL | https://www.example.invalid/notion/remnote-workflow-experience |

上表是占位符，不是真实库。实写前设置环境变量；未设置时不要把占位符当成线上库。环境变量覆盖 ID，不覆盖属性名：

- `FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID`：本次 POST 的 `parent.data_source_id`
- `FANLOOP_REMN_EXPERIENCE_DATABASE_ID`：只用于写前核对 `ntn api v1/databases/<id>` 的 `data_sources[].id` 含有上面的数据源

未设置时仍读取上表占位符，因此不会写到真实库。Workflow YAML 不增加配置字段。

FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID overrides the data source id; FANLOOP_REMN_EXPERIENCE_DATABASE_ID overrides the database id. Defaults live in references/notion-experience.md.

## 认证

操作者机器执行 `ntn login`（浏览器 OAuth）。没有可用钥匙串的机器先设置 `NOTION_KEYRING=0`，再执行 `ntn login`。不要把 Integration API key 当作主路径，也不要为了这次写入安装 Node SDK。

Auth is ntn login; headless hosts set NOTION_KEYRING=0. Do not require an integration API key.

## 顺序

1. 确认本步还没有为这条 finding 记下 Notion page id。已经记下则不要再 POST。
2. 读取本文件的默认 ID，再应用上面两个环境变量。
3. 只有 `FANLOOP_REMN_EXPERIENCE_DATABASE_ID` 被设置时，才执行 `ntn api v1/databases/<database id>`，确认 `data_sources[].id` 包含本次数据源。对不上则 `notion_sync.status=failed`，不要改写到别的库。未设置该变量时，用文档默认数据源直接 POST，不要为了核对再发一次 GET。
4. `findings` 为空：不调用 `ntn`，`notion_sync.status=skipped`。
5. 有一条 finding：按下面的正文 POST 一行。成功响应里的 `id` 写入 `notion_sync.page_ids`，`notion_sync.status=synced`。
6. `ntn` 不在 PATH、未登录、或 POST 失败：`notion_sync.status=failed`，`reason` 写退出码和简短 stderr。仍提交 `retrospective_recorded`，不让复盘失败挡住已经通过的制卡交付。摘要照旧追加到当前 progress Evidence。

When findings is non-empty, POST one page per finding with ntn api v1/pages to data source 「remnote workflow 经验表」; zero findings skip the write; a failed ntn call records notion_sync and still allows retrospective_recorded.

## 字段

Map 经验 to the one-line lesson, 日期 to the ISO run date, 主题 to the RemNote theme title, 类别 to 文案清晰度|工具链路|流程门禁|其他, 详情 to evidence + mechanism + suggested change + verification, and 编码状态 to 待编码 unless the finding already points at a merged fix (已编码) or is a non-code lesson (不适用).

| Notion 属性 | 来源 |
| --- | --- |
| 经验 (title) | 这条发现的一句话教训 |
| 日期 (date) | 本次运行日期，`YYYY-MM-DD`。复盘当时的日历日；主题标题里的日期不能代替它 |
| 主题 (rich_text) | `preparation_receipt.theme_title` 或已回读的主题 Rem 标题。未知写 `unknown` |
| 类别 (select) | 只能是 `文案清晰度`、`工具链路`、`流程门禁`、`其他`。文案与标题问题用文案清晰度；CLI、apply、`cc`、插件用工具链路；Route、预检、全景图门禁用流程门禁。对不上用 `其他` |
| 详情 (rich_text) | 纯文本四段：`证据:`、`机制:`、`建议改动:`、`验证:` |
| 编码状态 (select) | 默认 `待编码`。发现已经指向已合并的修复时用 `已编码`。不是代码改动的教训用 `不适用` |

不要新建 select 选项。用户没要求集群而 `cluster_powerup` 仍为 true 的那条发现，类别是 `工具链路`，编码状态是 `待编码`，除非它已经指向合并进 remnote-direct 的修复。

## POST

把 `<data_source_id>` 换成第 2 步得到的 ID。正文用 JSON，避免中文属性名在 shell 括号里被拆开。官方 CLI 用 `-X POST`。若本机 `ntn` 拒绝 `-X`，改用已经在这张表上成功过的 `ntn api --method POST v1/pages`，body 不变。

```bash
ntn api v1/pages -X POST --data '{
  "parent": {"type": "data_source_id", "data_source_id": "<data_source_id>"},
  "properties": {
    "经验": {"title": [{"text": {"content": "一句话教训"}}]},
    "日期": {"date": {"start": "YYYY-MM-DD"}},
    "主题": {"rich_text": [{"text": {"content": "主题标题"}}]},
    "类别": {"select": {"name": "其他"}},
    "详情": {"rich_text": [{"text": {"content": "证据:\n机制:\n建议改动:\n验证:"}}]},
    "编码状态": {"select": {"name": "待编码"}}
  }
}'
```

`notion_sync` 记 `status`、`reason`、`page_ids` 和实际使用的 `data_source_id`。不要把 token 写进 Evidence。
