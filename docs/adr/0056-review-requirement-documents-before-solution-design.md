---
status: accepted
date: 2026-08-19
---

# 在现有需求澄清 Step 内门禁需求文档审核

生产 Workflow 13.0.0 保持 ADR-0043 确立的 12-Step 主链，TechDesign 继续只有
`bootstrap_techdesign`、`clarify_requirements`、`design_technical_solution`、
`confirm_technical_solution` 四个 Step。相对 12.0.0，Step `id`、`name`、顺序和
`executor` 全部不变，不新增独立需求审批 Step。

`clarify_requirements` 继续由 Agent 生成并发布需求文档。其唯一正常 Route 改为同轮要求
`requirements_clarified`、`requirements_approved`、`requirements_approval_recorded`，使文档
URL、批准结论和审批消息 ID 缺一不可；完整组合命中后直接进入方案设计，不增加第二次
启动确认。文档发布后、人工结论到达前不提交终态 Result，因此当前 Step 保持不变。

人类拒绝或要求修改时，Agent 同轮提交 `requirements_clarified`、
`requirements_rejected`、`requirements_approval_recorded`，Route self-loop 到
`clarify_requirements`。Runtime 继续按回流目标和 producer Step 通用失效 Output，因此本轮
`requirement_document_url`、`requirements_decision`、`requirements_approval_message_id` 以及
下游产物不会继续作为当前事实。批准与拒绝使用同一互斥组，避免同一轮同时命中。

本决策提供路由层强制门禁，不提供批准来源的密码学或身份认证。它保留 ADR-0037 的信任
边界：Condition 是 Agent 上报的事实，CLI 校验结构、值、组合与 Route，但不认证审批人
身份或消息真实性；独立 Human Step 也使用同一 seam。若未来需要防止恶意 Agent 伪造，
必须新增独立可信审批能力、公开命令边界与 ADR，不能通过隐藏 flag 或 Prompt 假装实现。

本决策保留 ADR-0038 的五文件 Bundle、原子 Condition、OR-of-AND、显式 RouteSelection 和
通用 Runtime，保留 ADR-0043 的固定 12-Step 拓扑，保留 ADR-0052 的 Thrift-first YAML
authoring 边界，保留 ADR-0055 的 12-Step Requirement E2E 断言。不修改 IDL、Workflow
Schema、Runtime、Loader、Validator、State/Event Schema 或公共 CLI Contract。

`fanloop@12.0.0` 保持字节与 digest 不变，新 Requirement 通过唯一默认版本使用
`13.0.0`；运行中旧 Requirement 继续由其绑定版本解释，不提供 alias、迁移器、fallback
或双 Route。真实生产 YAML 前后示例、Step 逐项对比、人工审核记录和验收条件见
`docs/specs/requirement-document-human-review.md`。
