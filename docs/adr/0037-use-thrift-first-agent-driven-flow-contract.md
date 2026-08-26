---
status: accepted
date: 2026-08-14
amended_by: ADR-0045
---

# Flow 使用 Thrift-first 契约与 Agent 驱动的事实提交

Fanloop Flow 的公开方法、Request/Response、枚举、动态 Artifact Union 和稳定错误目录以 `idl/flow.thrift` 为唯一可编辑真值。Go Runtime 与 CLI 直接使用生成到公开 `flowidl` package 的类型和 `FlowService` 接口；不再维护手写 Flow DTO、Flow `CommandSpec` 错误目录或第二套公开字段。Workflow YAML 继续只表达 Stage、Step、executor、Prompt、Artifact Schema、正常 Flow 和允许的 Loop 目标。

公开报告直接切换为 `flow report progress`、`flow report output` 和 `flow report loop`。Agent 负责解释自然语言并选择命令：确认当前产物时提交 Output，人工反馈改变既有结论时提交 Manual Loop，只更新过程事实时提交 Progress。Output 只接受 Artifact、Summary 和 Evidence；调用方不提交 Check、Guard passed/failed、next Step、decision、move 或 forward。Loop 统一 Automatic 与 Manual 两种模式，目标必须来自最新 Status。

CLI 只校验 Thrift 结构、当前 Step、Artifact key/type/required/声明式约束和允许的 Loop mode/target，然后执行 Workflow 已声明的确定性迁移。Artifact 是 Agent 声明的 Step 产物或退出事实并参与 Guard；Evidence 是 Event/Card 的追溯数据，不参与 Guard且可以为空。CLI 不读取聊天、不判断人工意图、不验证审批人身份或消息真伪、不证明 Artifact 内容质量，也不复核 Agent 是否选择了语义最优的 Manual Loop 目标。

Human Step 使用同一 Output seam。Workflow 以普通 Artifact 表达审批事实，例如 `human_approved:boolean` 和 `approval_message_id:string`；Agent 识别人工消息后提交这些事实。`HumanApprovalVerifier`、可信 assertion、审批 ID/过期/重放绑定、专用 Host seam 和 `manual_evidence_required` 被删除。若未来需要强可信审批，必须新增独立认证能力、公开命令边界和 ADR，不能把隐藏 flag、stdin token 或 verifier 塞回普通 Report。

Status 返回当前行动所需的唯一视图：当前 Stage/Step、执行 status/summary/evidence、Prompt/Skills、当前 Artifact output contract、正常 `on_accept`、Automatic/Manual Loop 和已接受的上游 Artifact。State 仍只保存当前事实；Event 记录已接受的 Progress、Output、Loop durable effect；Trace/Panorama 从 State/Event 投影。每个已接受的 Progress、Output 和 Loop 都可 best-effort 刷新 Panorama，本地 durable commit 继续作为成功边界。

本决策在 Flow 域取代 ADR-0025 的 Go `CommandSpec` IDL 真值；修订 ADR-0035 的 Panorama 触发范围；部分取代 ADR-0036 的单一 Report、Evidence/Checks Guard、BackSelection 和可信 Human verifier。ADR-0036 的五文件不可变 Bundle、显式正常 Flow、Artifact owner/invalidation、本地事实优先等未冲突部分继续有效。

本次新发布包只携带 Workflow 6 和 State/Event Schema 7；维护者专用 `fanloop-dev` 也只携带使用同一当前 Schema 的新 Bundle。源码中的旧 Workflow Bundle 与旧 State decoder 一并删除；运行中旧 Requirement 必须继续由其已安装的旧 Release 解释或在切换前结束。这里取代 ADR-0009 中“新版本发布禁止删除已发布 Workflow `id@version`”的源码/发布包保留要求，并采用 ADR-0019 的 release-bound 支持边界。本次直接删除废弃路径，不提供 alias、双读写、迁移 adapter、Feature Flag 或 fallback。
