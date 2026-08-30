---
status: accepted
date: 2026-08-21
amends: ADR-0044, ADR-0047, ADR-0059, ADR-0063
amended_by: ADR-0068, ADR-0069, ADR-0070, ADR-0071, ADR-0072, ADR-0090
---

# 收敛维护者 Workflow 为本地优先的八步交付链

`fanloop-maintainer` 从 3 Stage / 12 Step 收敛为 3 Stage / 8 Step：需求定义包含
`工作区准备 → 需求澄清 → 需求确认`，需求实现包含
`方案设计 → 代码实现 → 本地验证`，变更交付包含 `代码审查 → MR 交接`。
`需求确认` 是唯一 Human Step；确认无需实现时可直接终止，确认需要实现后全部自动。
MR 交接成功是实现型请求的 terminal，不等待 CI、CR、合并或发布。

进入 `需求确认` 时，先等该 Human Step 的 Panorama Card 投递完成，再发送下文定义的完整改造计划
确认卡，并以确认卡发送成功为 turn boundary。需要实现时，Agent 只接受此后到达、由当前
`fanloop-dev` 应用 `cli_aaf6cd8160b89bda` 作用域下张菲帆
`ou_3b0b9cf8364168c5eb999bd6c5a33b95` 发送且 `senderType=user`、去除首尾空白后精确等于
“批准进入 需求实现”的全新用户消息；不得只校验显示名、复用其他应用作用域的 OpenID 或复用进入
Human Step 前的消息，也不得把错误 sender、笼统同意、需求调整或混合了批准与修改的消息解释为开发授权。
有效授权被 CLI 接受并进入需求实现前，不得修改源码、提交、推送或更新 MR。该规则只收紧
Release-bound Agent 的行为，保留 ADR-0037 的信任边界：CLI 不读取聊天或认证消息；若未来要求
不可绕过的可信审批，必须另行设计 Host approval seam 和 ADR。

需求澄清采用 Release-bound 的 grill-with-docs 组合，并在 Workflow 中显式绑定编排 Skill、
grilling 和 domain-modeling 三者。Agent 自查环境事实，按 design tree 的当前 frontier 分轮批量
询问；每轮人工选择立即写入 Issue Workspace 的 `requirements.md`。该文件是需求确认的
唯一主审材料。确认前 CONTEXT/ADR 只作为本地候选，确认后才同步仓库真值。

`requirements.md` 在进入 `需求确认` 前还必须包含可直接展示的完整改造计划：目标、现状问题、
逐项改造、影响文件/契约、保持不变与非目标、验证计划、交付边界。Human Step 的 Panorama Card
投递完成后，Agent 必须从最新 `requirements.md` 生成并通过 Botmux 发送一张自包含 Markdown
确认卡；不得只给摘要、文件链接或要求审批人回翻上下文。只有该计划卡发送成功后到达的全新消息
才可作为实现授权。该约束复用现有 `requirements_grilled`、`requirements_approval_recorded` 与
审批 Route，不新增 Step、Condition、Output、Runtime 或持久化字段。
该补充计划及生产 YAML 前后示例由张菲帆在消息 `om_x100b678964a1a4bcb25236829136cf2`
确认，进入实现授权为 `om_x100b678961cefcbcb306ab63ae0d4a3`。

方案设计顺序执行 To Spec 与 To Tickets。代码实现使用 Implement + TDD，只形成本地提交，
不 push、不创建 MR。本地验证继续采用 ADR-0063 的 `targeted|e2e` 风险分档：纯文档、
self-iteration Skill、未改变 Step/Route/Condition/executor 与 CLI/IDL/Storage/发布/测试入口的
Prompt/SkillBinding，以及具有可靠聚焦 seam 的局部行为可用 targeted；推进语义、代码、IDL、
发布、测试入口或影响不确定时必须用 e2e。e2e 在聚焦命令之外从同一工作树运行
`./tests/run-unit` 与 `./tests/run-e2e`。

本 ADR 修订 ADR-0063 的远端部分：maintainer 的验证事实全部来自当前本地工作树，Workflow
不读取、不等待 Codebase `run-unit` 或其他 MR checks，也不要求先有 MR。MR 创建后 CI 可
异步运行，但不参与 Route。ADR-0055 的两个测试入口及各自职责不变，本 ADR 不新增第三入口。

代码审查发生在本地验证之后，审查 `origin/master...HEAD` 并记录 reviewed HEAD。阻断项
回到实现，修复后重新经过本地验证与代码审查。这修订 ADR-0044/0047 在 maintainer 中“先有
MR、发布飞书评审文档”的要求：本地 review report 是必填 Output，但不发布外部文档；普通
`fanloop` Workflow 的三态 MR Review 和飞书文档契约不变。

`fanloop-dev-mr-handoff` 在最终 reviewed HEAD 上幂等 push、按分支创建或更新唯一 MR，
写详细 MR 描述，并向固定群真实 mention CR。MR 描述包含背景、问题、解决方案、改动/影响、
本地测试、ADR impact、风险/非目标；飞书交接只包含背景、问题、解决方案、1–3 条影响和 MR
URL。同一 Requirement + MR 复用原话题，本地记录保存 MR URL、rootMessageId 与 messageId。

`fanloop-cr-review` 与 verdict 模板进入 self-iteration Release。ADR-0058 的边界保持不变：
self-iteration Skills 随 Release 校验和安装，但不创建全局 Skill 链接；CR 角色在 Release 发布后
从 `~/.fanloop/current/skills/self-iteration/fanloop-cr-review/SKILL.md` 读取。

ADR-0038 的五份 YAML、原子 Condition、OR-of-AND、显式 Route、Output 失效和通用 Runtime
不变；ADR-0062 的当前 Bundle 与 digest 机制不变。不修改 Thrift IDL、durable storage、公开
CLI 或其他 Workflow。精确 12→8 Step 对比、25 Conditions、Route/SkillBinding 和人工终批
记录保存在 Issue Workspace 的 `tech-design.md` 与 `requirements.md`；最终授权消息为
`om_x100b6743b53170a0b3100eaf256758a`。

2026-08-24 修订：`confirm_requirements` 的两条成功 Route 都新增
`requirements_evidence_written`，输出 Root 内的 `requirements_evidence_path:path`。Agent 在上报
成功组合前，先把确认卡与批准消息回执写入 `requirements.md`，再把该文件路径作为
Condition Output 提交。拒绝与需求变更 Loop 不增加该 Condition；CLI 继续只验证类型、
相对路径、Condition 组合与 Route，不读取文件内容或认证消息。当前 Condition 总数由 25 变为 26。
三项合并计划由张菲帆通过
`om_x100b678abaa41ca0b3018e94e33bb91` 批准进入实现。
