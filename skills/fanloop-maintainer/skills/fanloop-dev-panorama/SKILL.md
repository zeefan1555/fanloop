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
没有用户可见消息通道的 headless worker 报告 blocked。不得把写入 artifacts/panorama-* 或任何仅保存文件当作 panorama_presented。

### `botmux`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format lark-json
botmux send --card-file <ABSOLUTE_SNAPSHOT_PATH> --no-mention --session-id <BOUND_SESSION_ID>
```

`BOUND_SESSION_ID` 只取 Requirement 的 `.fanloop/card/config.json` 中绑定的 `session_id`。

### `local_agent`

无用户可见消息通道的 headless worker 在 `panorama_presented` 上 blocked。report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.

能向用户聊天发消息的父 Agent（例如 Grok Bot SendToUser 或 Codex commentary）把本 Step 交给 headless 子 Agent（executor 或 subagent）时，只做下面的交接。`fanloop card render` 的成功响应不证明用户看见了消息。

The child must not submit panorama_presented.
After non-dry-run fanloop card render --view panorama --format markdown, the child stops and returns data.content and data.snapshot_path to the parent; report blocked if render fails.
The parent sends the exact data.content as the first user-visible message for that Step, then reports panorama_presented with that same snapshot_path, then continues the Step business Prompt.
Never run a panorama-gated Workflow fully headless in a child that cannot post user-visible messages.

子 Agent 交还这两个字段后停止，不改写 `data.content`，不写 `artifacts/panorama-*`，不继续业务 Prompt。父 Agent 先原样发送，再上报同一 `data.snapshot_path`，然后才继续业务 Prompt。

当前 Agent 自己持有可继续执行的用户可见消息通道，且没有把本 Step 交给 headless 子 Agent 时，投递只此一条，按顺序执行。先做一次非 dry-run render：

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown
```

只保留该次响应的 `data.content` 和 `data.snapshot_path`。把该次 `data.content` 原样作为进入该 Step 后的第一条用户可见消息单独发送；不加前缀或摘要，不展示 JSON envelope，不自行拼装内容。使用宿主可继续执行的用户可见消息通道，例如 Codex commentary。消息实际发出后即完成 Deliver/Verify，保留 ConditionResult 并在同一轮继续执行 Prompt、提交 Result 或进入后续 Agent Step；真正的人工问题只能随后发送，最终回复不得重复 Panorama。

`panorama_presented` 的 `value` 必须是上述同一次 render 的 `data.snapshot_path`。下列情况报告 blocked，不提交该 Condition：当前 Agent 没有可继续执行的用户可见消息通道，并且没有能代为发送的父 Agent；不得把 Markdown 写入 artifacts/panorama-* 或任何文件后声称已展示；进度、evidence 和 Result 不得把 content saved artifacts/panorama- 或 saved to artifacts 当作投递证明；工具输出、日志、只保存在上下文、回复“稍后发送”或只展示摘要也都不满足。有能发消息的父 Agent 时，子 Agent 按上面的交接交还 `data.content` 与 `data.snapshot_path`，不在这里改报 blocked。

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

只有已经把本次 render 的 `data.content` 原样发到用户可见通道的当前宿主，在展示或发送成功后，才返回下面的 ConditionResult。headless 子 Agent 不提交 `panorama_presented`。

```json
{"condition_id":"panorama_presented","output":{"type":"path","value":"<data.snapshot_path>"}}
```

`value` 必须是本次 render 原样返回的 Requirement Root 相对路径。渲染或发送失败、结果无法确认或宿主能力不可用时，
上报 blocked，不提交 Result。不得跨模式 fallback、双发、扫描旧快照或在结果未知时自动重试。
