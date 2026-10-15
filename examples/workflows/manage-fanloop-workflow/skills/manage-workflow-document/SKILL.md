---
name: manage-workflow-document
description: 为 manage-fanloop-workflow 创建并持续维护唯一 L2 飞书文档，把分析、候选、审核、发布证据和最终 HTML 可视化集中在同一文档。
---

# 维护唯一飞书管理文档

每个 Requirement 只使用一份飞书 Docx。先读取已有 Outputs；存在 `workflow_document_url` 时必须复用，不得
新建第二份文档。没有时使用 `lark-cli docs +create --as user` 创建，标题采用
`Fanloop Workflow 管理 · <workflow-id> · <create|update>`。

初始文档使用 DocxXML，只保留以下短索引，正文总说明不超过 12 行：

1. 目标与基线
2. 变更审核 HTML
3. 人工决定

Step、Condition/Route、Output、Skill、完整 YAML、Diff、验证和发布证据全部放进唯一主 HTML5 Block，不在
飞书正文重复铺开。飞书正文只显示标题、当前状态、主 HTML Block 和必要的人审决定原文。

创建后立即查询当前用户可用密级：

```bash
lark-cli drive +secure-label-list --as user --page-size 10 --lang zh --json
```

按名称精确选择 `L2-内部` 返回的实时 label ID，再执行：

```bash
lark-cli drive +secure-label-update --as user --token <DOCUMENT_URL> --label-id <LABEL_ID> --json
```

不得硬编码 label ID。随后使用 `lark-cli drive +permission-get-setting --as user --token <DOCUMENT_URL>` 回读
权限。L2 是密级标签，不会自动授予阅读权限；只能报告真实 sharing/permission 设置。当前用户或本次明确
的审核人无法访问时保持 blocked，提示文档 owner 补充权限，不得宣称“所有人都能看到”。

每次更新前用 `lark-cli docs +fetch --doc <DOCUMENT_URL> --detail with-ids` 回读最新 revision 和目标章节 block。
使用 `docs +update` 的定点 block 操作或 append 更新，不使用 overwrite，不破坏已有章节、评论和 HTML Block。
写入后重新 fetch，保存真实 `block_id`、`revision_id` 和文档 URL。所有 Condition 的文档引用对象至少包含：

```json
{"document_url":"...","block_id":"...","revision_id":1,"summary":"..."}
```

候选 YAML、绑定 Skill、diff、人工回复原文、Fornax 回执、Doctor 结果和用户更新提示都必须进入该文档的
HTML Block 或极短决定区。
本地文件仅用于临时打包或作为 `@file` 写入源，不得作为最终交付链接。

## HTML5 Block

HTML 的解释方式、视觉设计和内容层次全部由 `explain-workflow-design` 定义。本 Skill 只负责在分析、构建、
审核和发布阶段更新同一个主 HTML Block，不再创建设计图、候选图和交付图三个平行产物。

通过 DocxXML 插入：

```xml
<html5-block path="@./delivery-report.html"/>
```

HTML `<head>` 必须包含 UTF-8、viewport、`use-iframe=true`、`html-box-height-mode=auto` 和准确 description。
写入后重新 fetch，确认 `html5-block` 的 reference data 存在且摘要正确。三个 Condition 均引用当前主 Block：
分析阶段对应 `workflow_design_visual_ready`，候选审核阶段对应 `workflow_review_visual_ready`，发布阶段对应
`delivery_visualization_ready`。

人工明确批准时，把回复原文、候选身份、当前 `block_id`、`revision_id` 和内容摘要写入决定区。发布阶段更新
HTML 时必须复核已批准部分未变化，只允许追加发布状态、配置地址、版本 Diff 地址、Doctor 结果和
`fanloop update config` 提示。`fanloop update` 只切换 CLI，不能代替配置同步。
