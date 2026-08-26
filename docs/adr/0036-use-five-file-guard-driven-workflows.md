---
status: accepted
date: 2026-08-13
---

# 使用五文件 Guard 驱动 Workflow

Fanloop Workflow 5 由不可拆分、不可变的 `workflow.yaml`、`flow.yaml`、`guard.yaml`、`loop.yaml` 和 `prompt.yaml` 组成。`workflow.yaml` 只注册 Stage → Job → Step 与 executor；`flow.yaml` 是正常路径唯一真值；`guard.yaml` 只声明 CLI 可判定的静态产物、证据和检查要求；`loop.yaml` 只声明失败 Prompt、同一 Guard 和允许的 `back[]`；`prompt.yaml` 保存 Agent Prompt 及逐项 Skill 使用指导。Workflow digest 覆盖五份文件的规范化语义。

`flow status` 是执行者获取当前上下文的唯一接口，一次返回 Stage、Job、Step、解析后的 Prompt/Skills、Guard、有效 Artifacts，以及 flow mode 的 next/terminal 或 loop mode 的 GuardResult/back。调用方不读取 YAML 拼装命令。

所有状态上报继续使用一个 `flow report`，但 discriminator 直接切换为 `progress|output|back`。Output 只提交当前 Step 的真实 Artifacts、Evidence 和 Checks；CLI 通过统一 Guard evaluator 计算 passed/failed。passed 使用 Flow 的显式 `next_step_id` 或 terminal；failed 保存 durable GuardResult 并保持当前 Step。Agent 随后只能从 Status 返回的 `back[]` 选择 BackSelection；CLI 执行回流并按 Artifact owner 失效目标及其下游产物。

Human Step 复用同一 Output/Guard seam，但公开 CLI 不提供任何 Human outcome 输入。只有授权宿主在进程内安装的 `HumanApprovalVerifier` 可以验证不透明 assertion；验证结果必须完整绑定当前 Requirement、Workflow、Event、Step、Output、outcome、Evidence、过期时间和唯一 ID，核心重复校验并拒绝篡改或重放后才构造 human Evidence。独立 CLI 因没有验证器而在 Human Step fail closed。Trace 和 Card 仍是已提交 State/Event 的投影；本地事实先提交，远端投影 best-effort，失败不能回滚本地状态。

这次变更直接切换 Workflow 5、Flow 2、Guard 1、Loop 2、Prompt 1、State 6 和 Event 6。旧三文件 decoder、`rules[]`、RuleID decision、`FlowRule`、`LoopRule`、`RuleView`、Step instruction/skill、按物理顺序推导 next Step 及旧 State/Event decoder 全部删除，不提供 alias、adapter、双读写、Feature Flag、fallback 或迁移工具。

本决策取代 ADR-0033 的三文件 Rule Bundle、RuleID decision 和 Rule 回流模型，也取代 ADR-0029 中按有序 Step 隐式推进以及失败目标不可由 Agent 在受限候选中选择的部分。单一 `flow report`、不可变 Workflow 绑定、Artifact owner/invalidation、统一发布边界、Card 与投递分离和本地事实优先等未冲突决策继续有效。
