---
name: example-panorama
description: 在 example Workflow 的 panorama_presented Condition 要求时，按当前 Agent 宿主展示本次完整 Panorama。
---

# Example Panorama

只依据系统或开发者上下文中已经声明的 Agent 人设选择一个分支：Botmux 使用 `botmux`，AIME 使用
`aime`，Aiden 使用 `aiden`，Codex、Claude Code 和 Trae 等本地 Agent 使用 `local_agent`。人设不明确时
保持 blocked。

先读取最新 `flow status`。每次进入新 Step 只展示一次，并严格按 Render、Deliver、Verify、Report 执行。
只有展示成功后才能返回 `panorama_presented`；快照存在不等于已展示。没有用户可见消息通道的 headless
worker 报告 blocked。不得把写入 artifacts/panorama-* 或任何仅保存文件当作 panorama_presented。

## botmux

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format lark-json
botmux send --card-file <ABSOLUTE_SNAPSHOT_PATH> --no-mention --session-id <BOUND_SESSION_ID>
```

`BOUND_SESSION_ID` 只取 `.fanloop/card/config.json`。发送成功后保留本次响应的精确 `snapshot_path`。

## local_agent

无用户可见消息通道的 headless worker 在 `panorama_presented` 上 blocked。report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.

能向用户聊天发消息的父 Agent 把本 Step 交给 headless 子 Agent 时，只做下面的交接。`fanloop card render` 的成功响应不证明用户看见了消息。

The child must not submit panorama_presented.
After non-dry-run fanloop card render --view panorama --format markdown, the child stops and returns data.content and data.snapshot_path to the parent; report blocked if render fails.
The parent sends the exact data.content as the first user-visible message for that Step, retains that same snapshot_path and a panorama_delivery_receipt Evidence item, lets the Step business work finish, then submits panorama_presented together with the business Conditions in one flow report result.
Never run a panorama-gated Workflow fully headless in a child that cannot post user-visible messages.

当前 Agent 自己持有用户可见消息通道，且没有把本 Step 交给 headless 子 Agent 时，运行一次非 dry-run Markdown render，把该次响应的完整 `data.content` 原样作为进入 Step 后第一条用户可见
消息发送，不加摘要或前缀。`value` 使用该次 render 的 `data.snapshot_path`：

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown
```

下列情况报告 blocked，不提交该 Condition：不得把 Markdown 写入 artifacts/panorama-* 或任何文件后声称已展示；进度、evidence 和 Result 不得把 content saved artifacts/panorama- 或 saved to artifacts 当作投递证明。

## aime

运行一次非 dry-run `lark-json` render，使用当前话题根消息 ID 和本次精确快照：

```bash
lark-cli im +messages-reply \
  --message-id <CURRENT_THREAD_ROOT_MESSAGE_ID> \
  --msg-type interactive \
  --content "$(cat -- <ABSOLUTE_SNAPSHOT_PATH>)" \
  --reply-in-thread \
  --as bot \
  --format json
```

## aiden

运行一次非 dry-run `lark-json` render，把本次精确快照复制到唯一临时目录，校验字节一致后发送，并在
退出时清理：

```bash
tmp_dir="$(mktemp -d /tmp/fanloop-panorama.XXXXXX)" || exit 1
tmp_card="$tmp_dir/card.json"
trap 'unlink "$tmp_card" 2>/dev/null || true; rmdir "$tmp_dir" 2>/dev/null || true' EXIT
cp -- <ABSOLUTE_SNAPSHOT_PATH> "$tmp_card" || exit 1
cmp -s -- <ABSOLUTE_SNAPSHOT_PATH> "$tmp_card" || exit 1
aiden-bot-cli send-card --card-file "$tmp_card"
```

## 完成后

成功的终止 Route 被接受、Workflow 为 completed 之后，进入 Step 时的全景图不是完成投递。local_agent 以及任何必须把 `data.content` 发给用户的父宿主再做一次非 dry-run `fanloop card render --view panorama --format markdown`，把该次精确 `data.content` 原样送出。那一次内容含 `Done` 与 `100%`。没有用户可见通道则 blocked。不得把写入文件当成这次投递。公开 CLI 不证明用户看见了消息。

after a successful terminal Route, render panorama once more and deliver exact data.content; the in-step retrospect panorama is not the Done 100% delivery.

只有已经把本次 `data.content` 原样发到用户可见通道的当前宿主，在成功展示后，才返回下面的 ConditionResult。headless 子 Agent 不提交 `panorama_presented`。

成功展示后返回：

```json
{"condition_id":"panorama_presented","output":{"type":"path","value":"<data.snapshot_path>"}}
```

保留该 ConditionResult，等本 Step 业务完成后，与业务 Conditions 放进一次 `flow report result`；同一次 Result 的 Evidence 写入 `ref=panorama_delivery_receipt`、`delivered=true` 和相同 `snapshot_path`。不得先单独上报 `panorama_presented`。

渲染、发送或结果确认失败时保持 blocked，不扫描旧快照、不双发、不跨宿主回退。
