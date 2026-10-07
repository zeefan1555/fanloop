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

运行一次非 dry-run Markdown render，把该次响应的完整 `data.content` 原样作为进入 Step 后第一条用户可见
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

成功展示后返回：

```json
{"condition_id":"panorama_presented","output":{"type":"path","value":"<data.snapshot_path>"}}
```

渲染、发送或结果确认失败时保持 blocked，不扫描旧快照、不双发、不跨宿主回退。
