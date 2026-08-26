---
status: accepted
date: 2026-08-19
amends: ADR-0010, ADR-0045
amended_by: ADR-0063
---

# 新 Requirement 在绑定前选择正式 Workflow

Fanloop Release 可以携带多个正式 Workflow ID。`fanloop-workflow-selector` 在新
Requirement 首次 `flow init` 前读取自身 `SKILL.md` 内固定 marker 包围的版本化 fenced
YAML 规则块，按以下顺序返回一个 Workflow selector：

1. 用户显式指定；
2. 目标 repository/component 精确规则；
3. 发送者所属部门的稳定 `open_department_id` 规则；
4. `default: fanloop`。

规则未命中使用默认值；显式选择或已命中规则指向当前 Release 不存在的 Workflow 时
失败，不静默 fallback。Release 构建必须从该内联规则块解析 default、repository 和
department 的全部目标，并与 Manifest 中的 Workflow ID 交叉校验；规则块缺失、重复、
结构非法或目标不存在都阻断发布。宿主负责提供已验证的 repository 和 sender department
上下文；Skill 不在 CLI Runtime 中读取聊天或 Lark。选择结果只传给现有
`flow init --workflow`，State 继续固定 ID、version、digest。已初始化 Requirement 在
任何 Release 更新和 selector 之前直接继续绑定 Workflow。

新增正式 `douyin-game@1.0.0`，供配置部门使用。它复用普通 12 Step 与需求文档人工
审核门禁，检查并记录真实单测终态，但有意保留非阻断语义：`unit_tests_passed` 和
`unit_tests_failed` 都进入代码评审。单测失败事实不会被隐藏，也不构成该 Workflow 的
回实现 Route。

新增正式 `fanloop-maintainer@1.0.0`，供 `fanloop_cli` repository 规则使用。它采用
普通五文件 Bundle、统一 Runtime、Manifest、Doctor 与安装切换，不恢复
`fanloop_dev` build tag、DEV Overlay、特殊安装器、特殊 Doctor 或专用 Workflow
source。它直接复用 `fanloop@13.0.0` 的 3 Stage、12 Step ID 与顺序，差异只由自己的
Condition、Prompt 和 SkillBinding 表达。

维护流程在 `confirm_technical_solution` 人工审核 AI 生成的技术方案后才编码；
`confirm_ready_for_test` 是 Agent 自动交接，不增加实现后人工审批。Test Stage 保留
`clarify_test_cases`、`check_test_environment`、`execute_test_cases` 与
`confirm_test_results` 四个既有 Step：AI 先设计并按需补齐版本化测试资产，再检查环境，
最后从仓库根运行 `./tests/run-unit` 与 `./tests/run-e2e`。测试资产一旦变化，必须先回到
MR 门禁与 AI Code Review，使 Review 覆盖最终 Diff；最后由人审核 MR、源码、技术方案、
AI Code Review 和本地测试证据。维护方案默认保持 Thrift IDL 与 durable storage 不变；
确需改变时，必须在方案中给出精确契约 diff 与全链路影响，并由该方案 Human Gate 在编码
前明确批准。self-iteration Skill 随 Release 校验但不安装到用户全局 Skill 根。

因此本决策只取代 ADR-0045 中“不提供替代 maintenance Workflow”和“只存在一个
正式 Workflow ID”的部分；其退役专用 Overlay 与运行时的决定继续有效。ADR-0010
的显式 `flow init` 绑定保持不变，只增加调用前的宿主选择。ADR-0019 的不可变运行中
绑定、ADR-0038 的 YAML Route 真值和 ADR-0043 经 ADR-0060 修订后的默认 Workflow
语义继续有效。各 Workflow 对通用或业务专用 Skill 的引用边界由 ADR-0058 定义。
`douyin-game` 保留 ADR-0056 确立的需求文档人工审核门禁。

2026-08-19 follow-up 修订取代本 ADR 初版中“douyin strict、social 无专用差异”和
9 Step maintainer 的设计记录。后续终批又删除了尚未发布的 `social-game`，将其 strict
单测语义收敛到 ADR-0060 定义的默认 `fanloop@14.0.0`，并把 selector 规则载体从独立
`routes.yaml` 改为 `SKILL.md` 内联块。selector 优先级、默认 ID、运行中固定绑定及 Skill
分组边界不变。
