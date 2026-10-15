---
name: fanloop-panorama-delivery
description: Execute the one Fanloop CLI system action for Panorama delivery, using the declared host channel and acknowledging the exact action only after successful presentation.
---

# Fanloop Panorama Delivery

This is a CLI-bundled system Skill. Business Workflows do not declare, bind, copy, or specialize it.
`panorama_presented` and `panorama_card_published` are reserved for the Runtime Panorama action.
Do not submit either Condition.

## Select The Pending Action

Read the latest `flow status`. Execute this Skill only when `data.state.pending_actions` contains exactly one item whose
`type` is `panorama_delivery_required` and whose `skill.id` is `fanloop-panorama-delivery`. Keep its exact `id`,
`trigger_event_id`, optional `step_id`, and Skill path. Do not infer an action from a Workflow Condition, old snapshot,
Step name, or previous response.

The pending action must be handled before the current business Prompt, progress, result, clarification, confirmation, or
approval question. Within one action, use Render, Deliver, Verify, Acknowledge in that order.

## Select One Host

Use only the Agent identity already declared by system or developer context:

- Botmux Agent: `botmux`
- AIME Agent: `aime`
- Aiden Agent: `aiden`
- Local Agent, including Codex, Claude Code, and Trae: `local_agent`

Do not inspect environment variables, configuration, executables, or the filesystem to guess identity. If identity is not
declared, stop with the action still pending.

## Render And Deliver

Every branch performs exactly one non-dry-run Panorama render and uses only that response's exact `data.snapshot_path`.
Never scan `.fanloop/card`, reuse an old snapshot, summarize or rebuild content, switch channels, double-send, or retry an
unknown send result.

report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.

### `botmux`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format lark-json
botmux send --card-file <ABSOLUTE_SNAPSHOT_PATH> --no-mention --session-id <BOUND_SESSION_ID>
```

Read `BOUND_SESSION_ID` only from the Requirement's `.fanloop/card/config.json`. Continue only after the send command returns
a definite success receipt.

### `local_agent`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown
```

Send the response's complete `data.content` unchanged as the first user-visible message after entry. Do not add a prefix,
summary, JSON envelope, or manually reconstructed content. Use a host channel that allows the same Agent turn to continue.
Tool output, hidden context, logs, or a promise to send later do not count as delivery.

The child must not submit panorama_presented.
After non-dry-run fanloop card render --view panorama --format markdown, the child stops and returns data.content and data.snapshot_path to the parent; report blocked if render fails.
The parent sends the exact data.content as the first user-visible message for that Step, then reports panorama_presented with that same snapshot_path, then continues the Step business Prompt.
Never run a panorama-gated Workflow fully headless in a child that cannot post user-visible messages.

The child must not acknowledge the action. The parent sends that exact `data.content`, then acknowledges with the same
`data.snapshot_path`. The runtime close is `fanloop flow action acknowledge`, not a Workflow Condition.

after a successful terminal Route, render panorama once more and deliver exact data.content; the in-step retrospect panorama is not the Done 100% delivery.

### `aime`

Render once with `--format lark-json`, resolve that response's relative snapshot path below the Requirement Root, then use
the host-provided current thread root message ID:

```bash
lark-cli im +messages-reply \
  --message-id <CURRENT_THREAD_ROOT_MESSAGE_ID> \
  --msg-type interactive \
  --content "$(cat -- "$card_file")" \
  --reply-in-thread \
  --as bot \
  --format json
```

Continue only after the response proves success for the intended thread.

### `aiden`

Render once with `--format lark-json`, resolve the exact snapshot, copy it byte-for-byte into one temporary directory, and
send only after equality verification:

```bash
tmp_dir="$(mktemp -d /tmp/fanloop-panorama.XXXXXX)" || exit 1
tmp_card="$tmp_dir/card.json"
trap 'unlink "$tmp_card" 2>/dev/null || true; rmdir "$tmp_dir" 2>/dev/null || true' EXIT
cp -- "$card_file" "$tmp_card" || exit 1
cmp -s -- "$card_file" "$tmp_card" || exit 1
aiden-bot-cli send-card --card-file "$tmp_card"
```

Do not alter or re-render the JSON. Continue only after definite send success.

## Acknowledge

After and only after successful delivery, acknowledge the exact action with the exact snapshot returned by this render:

```bash
fanloop flow action acknowledge \
  --root <ABSOLUTE_REQUIREMENT_ROOT> \
  --action-id <PENDING_ACTION_ID> \
  --panorama-snapshot-path <EXACT_DATA_SNAPSHOT_PATH>
```

Read `flow status` again. Success requires that this action ID is absent from `pending_actions`. Then continue the current
business Prompt in the same turn; any real human question comes after the Panorama. The final ordinary reply does not
repeat the Panorama.

If render, delivery, verification, or acknowledgment fails or has an unknown result, stop and leave the action pending.
Do not submit business progress/result to simulate `blocked`; the Runtime intentionally rejects those reports while the
delivery action is pending.
