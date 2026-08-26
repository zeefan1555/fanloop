---
status: accepted
date: 2026-08-18
amends: ADR-0035, ADR-0038, ADR-0039
---

# 用权威 Trace 绑定门禁 TechDesign Bootstrap 与 Panorama

生产 Workflow 11.0.0 将原本分别负责仓库范围确认和环境准备的两个 required Skill
直接合并为 `techdesign-bootstrap`。第一 Step `bootstrap_techdesign` 同时产出已确认且
准备完成的 `repository_scope_path`，以及完成时的 `trace_document_url` 审计快照；
正常 Route 只有同时收到 `repository_workspace_prepared` 与
`trace_document_bound` 才能前进。

普通 `url` Output 只能证明字符串格式，不能证明 Requirement 已绑定该文档。因此
Condition OutputDefinition 增加可选 `source` 约束。本版只允许
`integration.trace.document_url`，并且只用于 `url` Output。Status 把 source 作为
OutputSpec 的一部分返回；Result evaluator 在接受 ConditionResult 前将提交值与当前
State 的 Trace Integration 精确比较。未绑定或不一致统一返回 `OUTPUT_INVALID`，
不提交 State、Event、Output 或 Card。

`source` 是 CLI 可从已提交本地事实判定的声明式约束，不是业务判断器，也不是表达式
语言。Go Runtime 只按 source 枚举解析当前事实，不识别生产 Workflow、Step、Skill
或 Condition ID。该约束修订 ADR-0038 中“CLI 只验证 Output 类型与静态约束”的
边界；Agent 仍负责创建文档、确认仓库范围、准备环境和判断何时上报完成。

Trace Integration 继续是当前绑定的唯一真值。`trace_document_url` Output 只记录
第一 Step 接受时的绑定快照，不参与 `trace bind`，Card 也不从 Output 反向恢复
Integration。显式改绑后，当前 Trace 链接以 Integration 为准。

`flow init` 仍先提交本地 `flow.initialized`，但在首张 Card Projection 与发送之前
完成 Card binding 捕获和 Trace provision，并重新加载包含 Trace Integration 的
State。Trace provision 失败不回滚本地 Flow，但禁止发送缺少 Trace 的 Panorama，
并返回可操作 warning。该顺序修订 ADR-0035 的首张 Card 编排，保留 ADR-0016 的
本地事实优先与远端失败不回滚边界。

Card Projection Schema 直接切换到 2，增加 `trace_document_url` 并从中恢复只读 Trace Integration；
“当前执行证据”展示一次权威 Trace 链接。Card 继续不读取 Trace Event，renderer 与
sender 继续分离，不新增投递回执、重试队列或 Card Event。该字段修订 ADR-0039 的
Projection Schema 1 契约，其余存储隔离边界不变。

本次直接删除 `select-techdesign-psms`、`techdesign-environment-check` 和生产
Workflow 10.0.0 当前源码路径，不保留 alias、兼容包装、双 Bundle 或迁移器。运行中
旧 Requirement 继续由其已安装旧 Release 解释。真实生产 YAML 的变更前后版本、
人工确认与验收条件见 `docs/specs/techdesign-bootstrap-trace.md`。
