---
name: fanloop-dev-panorama
description: 在 fanloop-maintainer 的 panorama_presented Condition 要求时按当前 Agent 人设选择唯一通道，展示 renderer 生成的 Panorama 并返回本次精确 snapshot_path。
---

# Fanloop Panorama 投递

本 Skill 只负责识别人设、选择唯一展示方式并满足 `panorama_presented`。外层 Workflow
决定何时执行 Condition。

## 识别人设

只依据系统或开发者上下文中已经声明的当前 Agent 人设选择：

- Botmux Agent：`botmux`
- AIME Agent：`aime`
- Aiden Agent：`aiden`
- 本地 Agent，包括 Codex、Claude Code 和 Trae：`local_agent`

不得运行 shell 命令、读取环境变量、扫描配置或探测可执行文件来猜人设。人设不明确时报告
blocked，不渲染、不发送。

## 展示 Panorama

仅当最新 `flow status` 的 `data.state.current.conditions[]` 包含 `panorama_presented` 时执行。
每次进入一个新 Step 只展示一次；同一 Step 内的 progress 和人工交互不重复展示。

先读取最新 `flow status`，再只执行对应的一个分支。所有分支都必须执行一次非 dry-run render，只使用
本次成功响应的精确 `data.snapshot_path`；不得扫描 `.fanloop/card` 猜最新文件。

### 统一事实门

所有宿主严格按 Render、Deliver、Verify、Report 的顺序执行。Panorama 必须是进入新 Step 后第一条
用户可见消息，不得先发进度前缀、澄清问题或确认问题。只有本次 render 的完整内容已经在当前宿主通道
展示成功，才返回 `panorama_presented`；生成快照、取得路径或准备好发送动作都不能单独满足该 Condition。

### `botmux`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format lark-json
botmux send --card-file <ABSOLUTE_SNAPSHOT_PATH> --no-mention --session-id <BOUND_SESSION_ID>
```

`BOUND_SESSION_ID` 只取 Requirement 的 `.fanloop/card/config.json` 中绑定的 `session_id`。

### `local_agent`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown
```

成功后保留响应的 `data.content`，并把它原样作为进入该 Step 后的第一条用户可见消息单独发送；不加
前缀或摘要，不展示 JSON envelope，不自行拼装内容。使用宿主可继续执行的用户可见消息通道，例如
Codex commentary。消息实际发出后即完成 Deliver/Verify，保留 ConditionResult 并在同一轮继续执行
Prompt、提交 Result 或进入后续 Agent Step；真正的人工问题只能随后发送，最终回复不得重复 Panorama。

宿主没有可继续执行的用户可见消息通道时报告 blocked。工具输出、日志、只保存在上下文、回复“稍后
发送”或只展示摘要都不满足 `panorama_presented`。

### `aime`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format lark-json
```

只使用本次成功响应的 `data.snapshot_path`，在 Requirement Root 下解析为绝对 `card_file`；不得扫描
`.fanloop/card` 猜最新文件。使用 AIME 宿主提供的当前话题根消息 ID 发送：

```bash
lark-cli im +messages-reply \
  --message-id <CURRENT_THREAD_ROOT_MESSAGE_ID> \
  --msg-type interactive \
  --content "$(cat -- "$card_file")" \
  --reply-in-thread \
  --as bot \
  --format json
```

### `aiden`

先运行一次非 dry-run `lark-json` render，并只使用本次返回的 `data.snapshot_path`
在 Requirement Root 下解析出精确绝对 `card_file`。将快照原样
暂存到唯一的 `/tmp` 目录，校验字节一致后发送，并在命令退出时清理：

```bash
tmp_dir="$(mktemp -d /tmp/fanloop-panorama.XXXXXX)" || exit 1
tmp_card="$tmp_dir/card.json"
trap 'unlink "$tmp_card" 2>/dev/null || true; rmdir "$tmp_dir" 2>/dev/null || true' EXIT
cp -- "$card_file" "$tmp_card" || exit 1
cmp -s -- "$card_file" "$tmp_card" || exit 1
aiden-bot-cli send-card --card-file "$tmp_card"
```

不得重新渲染或修改 JSON。暂存、校验或发送任一步失败时，清理临时文件并停止。

## 返回 ConditionResult

只有当前分支已成功展示或发送后，才返回：

```json
{"condition_id":"panorama_presented","output":{"type":"path","value":"<data.snapshot_path>"}}
```

`value` 必须是本次 render 原样返回的 Requirement Root 相对路径。渲染或发送失败、结果无法确认或宿主能力不可用时，
上报 blocked，不提交 Result。不得跨模式 fallback、双发、扫描旧快照或在结果未知时自动重试。
