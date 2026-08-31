# Lark 两机器人真实 E2E 契约

## 固定身份

| 角色 | 显示名 | Lark app ID |
| --- | --- | --- |
| driver | 使用 Fanloop 机器人 | cli_aafadbc67e799cdc |
| target | FanLoop 机器人 | cli_a9245f0fddf8dbc8 |

唯一群为 oc_d532c3a5eda84c60728ab174b0ef671a。open ID 具有 app/receiver 视角，每轮从本次 driver
source session 解析，不得写死。

## Preflight

1. 执行 botmux bots list --session-id DRIVER_SESSION，确认 chat、isSelf driver app 与 target app。
2. 回读 lark-cli whoami --as bot，要求 bot identity ready；不得切换 --as user。
3. 在 candidate_head 的干净工作树中，使用 `env -u FANLOOP_DATA_HOME` 并同时清除四个 Skill Root
   覆盖后执行 `npm run install:local`。要求 `$HOME/.fanloop/current` 指向本机 Releases，且
   `$HOME/.fanloop/current/bin/fanloop version` 的 commit 等于 candidate_head、`fanloop doctor` 为
   healthy。禁止用临时候选 bin、私有 bin 或隔离数据目录替代，验收结束后保留该 candidate 为 current。
4. 从 `eval_playbook_path` 读取恰好两个冻结 Case，校验 Playbook 文件名摘要及每个 brief_sha256、
   rubric_sha256；固定执行顺序为清单顺序，不选择 Case。原始 brief 已包含全新目录、全新 Requirement、
   公开 CLI、证据字段、内层环境隔离和“到 merge_code 前停止”，不得再生成、复制或改写。

## 并行 Dispatch

两个 Case 分别执行一次以下命令，可并行等待：

~~~bash
botmux dispatch \
  --session-id DRIVER_SESSION \
  --chat-id oc_d532c3a5eda84c60728ab174b0ef671a \
  --title UNIQUE_CASE_TITLE \
  --bot-app cli_a9245f0fddf8dbc8 \
  --brief-file FROZEN_BRIEF_PATH
~~~

不得传 --bot、--repo，不发送 /repo、/restart，不使用用户 token。两个 dispatch 必须产生不同的根消息、
thread 和 Requirement。`FROZEN_BRIEF_PATH` 必须是 Playbook 记录并通过摘要校验的原文件；“全新”只
适用于话题、目录和 Requirement，不适用于题面或 Rubric。

## 内层 CLI 隔离

target 执行 Fanloop 时使用等价于以下的环境边界：

~~~bash
env -u BOTMUX_CHAT_ID -u BOTMUX_SESSION_ID /Users/bytedance/.fanloop/current/bin/fanloop ...
~~~

外层 Botmux 变量只用于机器人通信，绝不进入 Requirement。内层不得使用 lark-cli、用户 token 或
--as user，不运行 trace bind/trace sync，不生成 Card Binding、Trace Integration、远端 Trace Event、
用户 Trace 文档或用户 CLI 日志文档。

## Readback

1. 保存两个 dispatch 的 seedMessageId、threadRootId 和接单结果。
2. 用同一 driver session 执行 botmux history，并对引用消息执行 botmux quoted；截图不能代替回执。
3. 读取两个 Requirement 的 Status、State、Events、Card、CLI 输出与文件树；验证候选 SHA 和 Case 结果。
4. 明确断言 `.fanloop/card/config.json` 不存在，State 不含 Trace/CLI 文档 Integration，Events 不含
   trace_document_bound、trace_sync_started 或远端 trace_synced；不存在用户 Trace/CLI 文档 URL。
5. 到达 merge_code 前、确定失败或超出 Case 边界立即停止；Candidate 不执行合码。

任一隔离断言失败为 governance_failed；权限或网络不可用为 infra_blocked，不回退用户身份。
