---
name: fanloop-dev-workflow
description: 维护 zeefan1555/fanloop 自身的入口，沿需求确认、研发实现、验收交付三阶段推进到合并与本地 CLI 更新。
---

# Fanloop Dev Workflow

始终执行：**读取 Status → 执行当前 Prompt/Skills → 上报 Progress 或 Result → 重新读取 Status**。`.fanloop` 只由 CLI 管理；构造命令前读取目标叶子 `--help`。

## 控制器

已有 Requirement 先按通用 `fanloop-workflow` 解析固定控制器。存在 `bound-release-home` 后，Status、Progress、Result、Doctor 与 Panorama 都用 `bound-release-home/current/bin/fanloop` 和 `skill-roots/{codex,agent,trae,claude}`；不得回退到 `$HOME/.fanloop/current` 或手改 State 绕过 `WORKFLOW_MISMATCH`。新 Requirement 的 init 使用全局 current。

候选验收只在临时 `FANLOOP_DATA_HOME` 中安装，不改变全局 current。只有 PR 合并后的 `update_local_cli` 才先固定控制器，再把精确 merge commit 安装到全局 current。

## 当前 Step

1. 只使用最新 `data.state.current` 的 prompt、skills、conditions、available_routes 与 outputs。每个 Skill 按 Status 给出的绝对 `SKILL.md` 完整读取，不按 ID 猜路径。
2. 未完成时运行 `<REQUIREMENT_CONTROLLER> flow report progress`；形成事实后从当前 Conditions 选一个完整 `when.any_of` 组合，运行 `<REQUIREMENT_CONTROLLER> flow report result` 并明确 next/back/terminal。
3. Output key 由 Condition 定义，Evidence 不参与路由。命令失败不猜测、不换控制器；每次写后重新运行 `<REQUIREMENT_CONTROLLER> flow status`。
4. `confirm_requirements` 是 Agent Step。无新产品决策时可委托一个独立 Sub-agent 复核 requirements.md，随后以 `agent_approved` 推进；有真实决策 frontier 时保持 blocked，并使用保留的人工 Panorama Route 等用户明确决定。
5. 三个 Stage 的稳定产物分别是需求飞书文档、研发实现飞书文档、验收交付飞书文档；回流只更新同一文档，不重复创建。
6. Review 冻结 candidate_head；Agent 验收使用一个无实现上下文的 Sub-agent 与隔离候选 CLI；merge_code 负责唯一 PR、Ruleset、required checks 和 squash 合并；update_local_cli 安装精确 merge commit。

## 需求确认的人工补充路径

只有当前 Agent 或独立 Sub-agent 无法消除真实产品决策 frontier 时才进入人工路径；这不是端到端人工验收。

1. 在 `requirements.md` 记录 `expectedApprover`：当前 `fanloop-dev` 应用 `cli_aaf6cd8160b89bda` 作用域下的张菲帆 OpenID `ou_3b0b9cf8364168c5eb999bd6c5a33b95`。
2. 先按 `panorama_card_published` 绑定 Skill 展示最新 Panorama，保存本次精确 `panorama_snapshot_path`；展示失败保持 blocked。
3. 再从最新 `requirements.md` 生成一张自包含 Markdown 确认卡。卡内依次完整展示当前 Stage/Job/Step、目标、现状问题、逐项改造、影响文件/契约、保持不变与非目标、验证计划、交付边界和精确授权口令；不得只给摘要、只贴链接或要求审批人回翻上下文。使用 `botmux send --mention-back` 发送并记录 messageId，确认卡发送成功为 turn boundary。
4. 只接受 turn boundary 此后到达的全新用户消息。用 `botmux quoted <message_id> --raw` 或当前话题 history 回读元数据；必须同时满足 `senderType=user` 且 `senderId` 精确等于上述 OpenID，显示名不能代替身份校验。
5. 需要实现时，正文去除首尾空白后必须精确等于 `批准进入 需求实现`。`同意`、泛化授权、同时含需求修改的消息、机器人结论或 turn boundary 之前的消息都不构成批准。
6. 有效批准先写入审核卡 messageId、Panorama 路径、批准消息 ID、senderType、senderId、正文与实现必要性，再上报完整人工 Route。其余反馈记录真实证据并回需求澄清；不得伪造或复用旧批准。

## 最终回复

遵循通用 `fanloop-workflow` 的 renderer-owned 最终回复契约。结束一轮普通回复前紧邻执行：

~~~bash
<REQUIREMENT_CONTROLLER> flow status --root <ABSOLUTE_REQUIREMENT_ROOT>
<REQUIREMENT_CONTROLLER> card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown --dry-run
~~~

本轮最终普通回复必须完整原样展示 render 响应的 `data.content`。任一命令失败即以真实错误阻塞并停止；不得手工 fallback、复用旧 render 或快照。
