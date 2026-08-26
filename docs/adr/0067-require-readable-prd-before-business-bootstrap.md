---
status: accepted
date: 2026-08-19
amends: ADR-0038, ADR-0048, ADR-0059, ADR-0062
---

# 业务 Workflow 在 Bootstrap 前要求可读 PRD

`Requirement.source_url` 只表示用户给出的需求来源定位符，不承诺其本身是 PRD。Meego
工作项 URL 必须由 `techdesign-bootstrap` 使用 Meegle CLI 解析，读取工作项中唯一明确的
PRD 字段，再通过飞书文档能力读取非空正文。Meego 标题、描述、评论和其他字段都不能在
PRD 缺失或不可读时作为降级正文。

Skill 在成功读取正文后生成 `.techdesign/requirement_source.json`，记录工作项身份、PRD
字段与 URL、正文摘要和验证时间，不保存凭据或完整 Meego 响应。直接 PRD URL 与只有对话
原文的既有入口也生成同类回执，保持原入口可用。

为使需求定位符不因跨轮会话或后续外部调用失败而丢失，Skill 在识别到用户提供的 Meego
URL 后、调用 Meegle CLI 前，先原子写入 `.techdesign/meego_source.json`。该文件只保存原始
Meego URL、来源类型、schema version 和捕获时间；它不是 PRD，不证明 PRD 可读，也不参与
Condition、Route 或 Result。完整工作项身份仍只在 Meegle 查询和 PRD 校验成功后进入需求
来源回执。

按 ADR-0062 的无业务版本当前 Bundle 模型，直接修订 `fanloop` 和 `douyin-game`
当前五文件 Bundle。两个 Bundle 都新增原子 Condition
`requirement_source_resolved`，输出 `requirement_source_receipt_path`；第一 Step 的唯一正常
Route 要求它与 `repository_workspace_prepared`、`trace_document_bound` 同时成立。
Bundle 的准确内容身份继续由 Workflow ID 与 normalized digest 表达，随完整 Release 发布。

两个当前 Bundle 相对 merge base 保持 12 个 Step 的 ID、名称、顺序和 executor 完全不变，
`loop.yaml` 零变化，后续业务 Route 零变化。2026-08-19 人工审核确认的真实变更是：
`condition.yaml` 新增上述 Condition，`flow.yaml` 将第一 Step 的唯一 AND Route 从
`[repository_workspace_prepared, trace_document_bound]` 修订为
`[requirement_source_resolved, repository_workspace_prepared, trace_document_bound]`，
`prompt.yaml` 只同步 Bootstrap 执行与 Condition 说明。生产 YAML 和契约测试共同保存该审核真值。

本次不扩展 Output `source` 或通用 Runtime 业务判断。CLI 仍只验证 YAML 声明的类型与
Route；需求正文读取、回执生成和回读真实性由 required Skill 负责。因此保留 ADR-0048 的
声明式 Runtime 边界和既有 `path` Output 语义，不把所有 path 全局收紧为现存普通文件，
避免改变 `fanloop-maintainer` 等目录型 Output。按 ADR-0038 用新的原子 Condition 表达
可审计的前置业务事实。

Trace 展示把 Meego 定位符和 PRD 文档分开。Meego URL 不再写入“PRD 文档”或 Registry
PRD 字段；直接 PRD URL 与需求澄清文档仍投影为 PRD。

本决策不修改 Thrift IDL、生成代码、State/Event Schema、Workflow YAML Schema 或公开
CLI 命令契约。`fanloop-maintainer` 是反馈诊断流程，不绑定 `techdesign-bootstrap`，
当前 Bundle 语义不变。
