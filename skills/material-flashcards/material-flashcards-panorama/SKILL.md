---
name: material-flashcards-panorama
description: 在 material-flashcards 的 Human Step 原样展示 renderer-owned 隐私安全 Panorama，并返回本次精确 snapshot_path。
---

# Material Flashcards Panorama 投递

本 Skill 只负责识别人设、执行唯一 renderer 分支、原样展示并满足 `panorama_card_published`。它不读取、拼装或概括材料和卡片。

## 隐私前置门禁

只在精确卡片预览已经通过另一路径投递，且最新 `flow status` 当前 Condition 包含 `panorama_card_published` 时执行。渲染前确认 Current Evidence 为空；若仍有上游 locator，先在当前 Step 使用不带 Evidence、且 Summary 仅含 progress、card_count、已确认可展示的 Vault 相对 target_path 和 review_status 的 `flow report progress --status in_progress` 清空，再读取最新 `flow status` 确认。Requirement 标题必须通用且不敏感。

Panorama 不得包含卡片正文、个人细节、来源内容、排除细节、findings、反馈、消息 ID、sender identity、投递位置、私密 URL 或详细错误。平台固定的流程层级和 Trace/CLI 链接可以保留，但链接指向的信息也必须通过同一隐私门禁。失败时 `blocked`，不得降级为手工拼卡。

## 识别人设

只依据系统或开发者上下文中已经声明的当前 Agent 人设：

- Botmux Agent：`botmux`
- AIME Agent：`aime`
- Aiden Agent：`aiden`
- Codex、Claude Code 和 Trae：`local_agent`

不得运行 shell、读取环境变量、扫描配置或探测可执行文件来猜人设。人设不明确时停止。

## 唯一展示分支

先读取最新 `flow status`，再选择且只选择一个分支。每次进入 Human Step 执行一次非 dry-run render；只使用本次成功响应的精确 `data.snapshot_path`。不得跨模式 fallback、双发、扫描旧快照，也不得复用上次快照。

### `botmux`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format lark-json
botmux send --card-file <ABSOLUTE_SNAPSHOT_PATH> --no-mention --session-id <BOUND_SESSION_ID>
```

`BOUND_SESSION_ID` 只取 Requirement `.fanloop/card/config.json` 已绑定的 `session_id`。

### `local_agent`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown
```

保留 `data.content`；本轮最终普通回复必须完整展示同一份 Panorama。不展示 JSON envelope，不自行拼装内容。

### `aime`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format lark-json
lark-cli im +messages-reply --message-id <CURRENT_THREAD_ROOT_MESSAGE_ID> --msg-type interactive --content "$(cat -- "$card_file")" --reply-in-thread --as bot --format json
```

`card_file` 必须由本次 `data.snapshot_path` 在 Requirement Root 下解析；不得修改内容。

### `aiden`

将本次 `lark-json` render 返回的精确快照原样复制到唯一临时目录，`cmp` 校验字节一致后执行：

```bash
aiden-bot-cli send-card --card-file "$tmp_card"
```

无论成功失败都清理临时文件；不得重新渲染或修改 JSON。

## 返回

只有当前分支真实展示成功后，才返回：

```json
{"condition_id":"panorama_card_published","output":{"type":"path","value":"<data.snapshot_path>"}}
```

`value` 必须是本次 render 原样返回的 Requirement-root-relative 路径。结果不确定、渲染或发送失败、宿主能力不可用时保持 `blocked`。
