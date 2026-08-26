---
status: accepted
date: 2026-08-25
amends: ADR-0039, ADR-0048, ADR-0064, ADR-0067, ADR-0073, ADR-0074
---

# 发布维护者产物与完整 CLI 日志链接

`fanloop-maintainer` 的 production Trace Integration 直接扩展为一次性绑定两份不同文档：
Trace 文档和 CLI 日志文档。首次 `trace bind` 必须同时提供两者；相同文档 identity、相同
Registry 与相同日志文档 identity 的重复绑定幂等，任一项变化或复用 Requirement 来源、技术方案、
Trace 文档都拒绝。普通 Workflow production 与所有显式 test 仍只绑定 Trace 文档。该决定扩展
ADR-0064 的一次性绑定，并保留 ADR-0048 的 Integration 权威与本地事实优先边界。

`trace sync` 对维护者 production 固定投影 `trace_document`、`cli_log_document`、`registry`
三个独立 target。CLI 日志 target 在共享锁下读取 `.fanloop/log/cli.jsonl` 的原始字节快照，
不解析、脱敏、删减、截断或摘要，再以不会被正文反引号提前闭合的 Markdown fence 覆盖同一飞书
文档。当前 `trace sync` 调用仍在命令返回后才追加，因此远端最多落后这一条调用；平台容量或更新
失败形成 `TRACE_UPDATE_FAILED` 的 partial 结果，不回滚本地事实和其他成功 target。日志仍不参与
恢复、路由或 Doctor 健康判断。该决定修订 ADR-0073 的“不参与远端投影”和 ADR-0074 的“无上传”
边界；用户已明确接受完整 transcript 可能包含密码、token、URL、Evidence 和用户标识的风险。

自迭代 production Registry 在既有字段外增加 `需求澄清`、`技术方案`、`MR`、`CLI 日志` 四个
URL 字段。需求澄清、技术方案和首个 MR URL 来自当前有效 Output，CLI 日志来自 Trace Binding；
回流失效后下一次同步清空对应字段。`PRD` 保留真实 PRD 语义，没有真实 PRD 时写 `null`，不再把
需求澄清文档冒充 PRD。普通 production 与 test Registry payload 不变，不扫描或批量回填历史行。
这局部修订 ADR-0067 中维护者需求澄清可投影为 PRD 的旧语义。

Trace 文档头部把 PRD、需求澄清、技术方案、MR 与 CLI 日志分开展示。Card Projection 保存日志
文档 URL，Panorama 在全局执行证据区把它显示在 Trace 链接之后；日志不成为 Workflow Output，
不进入阶段列，也不随单个 producer Step 回流失效。该决定扩展 ADR-0039/0048 的独立 Card 投影。

Storage Thrift 直接切换 State/Event Schema 12、Card Projection Schema 5、Trace Config Schema 2；
公开 Trace Thrift 增加日志文档绑定、状态字段和 target 枚举。生成物只由现有生成链产生。旧 Schema
不双读、不迁移、不加兼容层；运行中的旧 Requirement 继续由其绑定 Release 解释。ADR-0051 的
Thrift 真值边界保持不变。

本决策不修改五份生产 Workflow YAML、Step、Condition、Route、executor 或 Prompt/SkillBinding，
不增加依赖、日志轮转、增量上传、清理、历史回填、自动合并或发布。真实 Base 的四列与默认视图只在
交付验证阶段一次性创建并回读，不由每次 Runtime sync 自动修改 schema。

需求范围由张菲帆在 `om_x100b67fae5f388a0b1a7dd47a659e9e` 批准进入实现；上述两份 Thrift 的
精确 field ID、类型、可选性、枚举与消费者影响由 `om_x100b67fabd4fe8acb1f3d6d0141392f` 单独批准。
