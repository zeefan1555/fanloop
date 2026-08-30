# Lark 真实 Agent E2E 契约

## 固定身份与群

| 角色 | 显示名 | Lark app ID |
| --- | --- | --- |
| driver | 使用 Fanloop 机器人 | `cli_aafadbc67e799cdc` |
| target | FanLoop 机器人 | `cli_a9245f0fddf8dbc8` |

唯一允许的群是 `oc_d532c3a5eda84c60728ab174b0ef671a`。open ID 具有 app/receiver 视角，
不得写死；每轮都从本次 driver source session 重新解析并记录。

## Preflight

1. 找到在线 source session 后执行 `botmux bots list --session-id <driver-session>`，回读并同时确认：
   session 的 chat ID 精确匹配固定群、`isSelf=true` 的 app 是 driver app、target app 在同群可见。
2. 若不存在同时满足三项的在线 session，返回 `infra_blocked`。不得切换群、app 或用户身份补做。
3. 从最终工作树执行 `npm run install:local`，再保存 `fanloop version` 和 `fanloop doctor`；版本中的 commit
   必须等于 candidate HEAD。
4. 为本轮 Case 准备简报文件，写明固定 Case、全新 Requirement 要求、只使用公开命令、证据字段和
   “到 Human Step 停止”。简报不得要求机器人替人审批、合并或发布。

## Dispatch

只用 driver session 直接派发一次：

```bash
botmux dispatch \
  --session-id <driver-session> \
  --chat-id oc_d532c3a5eda84c60728ab174b0ef671a \
  --title <包含-case-id-与-candidate-commit-的全新标题> \
  --bot-app cli_a9245f0fddf8dbc8 \
  --brief-file <case-brief-path>
```

`--bot-app` 负责建立并回读双方 talk-only exact chatGrant、新建顶层话题并等待 target 接单。不要传
`--bot`、`--repo`，不要发送 `/repo`、`/restart` 等 operate 指令，也不要使用或回退用户 token。

## Readback 与停止

1. 保存 dispatch JSON 中的 `seedMessageId`、`threadRootId` 和接单结果。
2. 用同一 driver session 执行 `botmux history --session-id <driver-session>`，定位新根消息及 target 回复；
   对引用消息执行 `botmux quoted <message-id> --session-id <driver-session>`，不得用截图代替可读回执。
3. 从 target 返回内容定位全新 Requirement Root；读取其 `fanloop flow status`、State、Events、Card 和
   CLI 执行证据，确认绑定的 Release/commit。
4. 到达 Human Step、明确失败或超出 Case 边界即停止。机器人不得提交人工 Condition。
5. Eval Report 至少保存：群、driver/target app ID、本轮双方视角 open ID、source session、根消息、
   thread、Requirement Root、candidate commit、命令/卡片/响应、停止原因、Rubric 明细与结论。
