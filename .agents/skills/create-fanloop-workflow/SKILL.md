---
name: create-fanloop-workflow
description: 将复杂 Skill、SOP 或原始材料拆成可审核、随本地仓库维护的完整 Fanloop Workflow Skill 包；用于新增或重构 Workflow，不用于修改 Runtime、IDL 或修改通用安装机制。
---

# Create Fanloop Workflow

把输入材料转换为 second-brain-skills 配置仓库 `<repo>/fanloop-workflow/<workflow-id>/` 下的完整 Skill 包：根 SKILL.md、五份 YAML 和子 skills/<skill-id>/**。产物随仓库维护，包根不再添加 workflows/<id>/ 包装层；写入不代表已通过检查或业务审批。

## 1. 先确认需要 Workflow

读取配置仓库 `AGENTS.md`、Fanloop 代码仓库的 `idl/yaml.thrift` 与相关 `docs/adr/`，以及最接近目标的现有 Workflow。存在相近业务
Workflow 时优先复用；否则以 `examples/workflows/example/` 为可运行骨架。若一个原子 Skill 已足够完成任务，停止
新增 Workflow，并说明应复用或新增哪个 Skill。

先做一张材料映射表：每项已确认事实对应哪个 Step、Condition、Route、Output 或 Skill；无法映射的内容标为
开放问题。只提取已经确认的目标、输入、输出、责任边界、人工决定、失败处理和完成标准，不把建议写成流程事实。

## 2. 改写可运行骨架

把 `examples/workflows/example/` 复制到本地临时目录，按材料映射表替换业务语义。示例只提供结构，不提供默认业务
决定：替换 Workflow/Stage/Job/Step/Condition/Output/Prompt/Skill ID 和正文，删除不需要的分支，只保留
真实需求需要的路径。完成后用 `rg` 检查新目录没有遗留 `example`、`analyze_materials`、`draft_result`、
`confirm_result` 等示例标识。

随后按以下规则收紧执行图：

- `workflow.yaml` 只注册 Stage、Job、Step；每个 Step 只有一个清晰责任，`executor` 与真实决策者一致。
- `condition.yaml` 把可独立验证的事实建成原子 Condition，并为每项定义唯一 Output；不要用 Evidence 代替事实。
- `flow.yaml` 只写正常前进或终止 Route，`loop.yaml` 只写返回最早受影响 Step 的回流 Route。
- 每个 Step 至少声明一条 Loop Route；没有更早依赖可返回时回到自身，不能留下无法返工的 Step。
- `prompt.yaml` 只保留 Step 指令和 SkillBinding；稳定、可复用的具体方法拆到同目录原子 Skill，调用处只留指针。
- 保留明确的 Human Gate。Agent 不得代替人批准，也不得把旧回复或实现自证当作新的人工决定。
- 每个生产 Workflow 都声明输出 `panorama_snapshot_path:path` 的普通 `panorama_presented` Condition，绑定同一 Workflow 内的展示 Skill，并把它加入 `flow.yaml` 与 `loop.yaml` 每条 Route 的每个 AND 组，包括正常、终止和回流 Route。展示时机、step-start 第一条消息和完成态卡片必须写进该 Workflow 的 Prompt/Skill；通用 Runtime 与校验器不会补齐。
  展示 Skill 的 `local_agent` 使用 non-dry-run `fanloop card render --view panorama --format markdown`，原样投递 `data.content` 并保留本次 `data.snapshot_path`。本 Step 业务完成后，`panorama_presented` 必须与业务 Conditions 放进同一次 `flow report result`，不得单独上报。
  必须写明：`report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.`
  不得把写入 `artifacts/panorama-*` 写成成功路径。提到 `local_agent` 时还要写明父/子交接：`The child must not submit panorama_presented.` 子 Agent在 non-dry-run render 后停住，把 `data.content` 与 `data.snapshot_path` 交还父 Agent；父 Agent 原样发送，保留同一路径与 `panorama_delivery_receipt` Evidence，让业务工作完成后再把全景与业务 Conditions 合并成一次 Result。成功 terminal Route 后另外 render 并投递 `Done` / `100%` 完成态卡，不再提交 Condition。

不要新增兼容层、fallback、运行时代码、IDL、依赖或脚本。现有 Schema 无法表达需求时停止，并把缺口与
最小契约变更交给人另行审核。

## 3. 人审后再写隔离候选

写入候选目录前，先展示：

1. 完整 Step 表：顺序、`id`、`name`、Stage、Job、`executor`。
2. 五份 YAML 的完整结构和全部 Condition/Route 组合。
3. 原子 Skill 清单、每项职责和被哪些 Prompt 引用。
4. Human Gate、回流目标、每条 Route 的 Panorama 覆盖、非目标和 ADR impact。

新 Workflow 的变更前基线记为不存在；重构现有 Workflow 时逐项列出 Step 的新增、删除、改名、重排和
`executor` 变化。取得针对该具体草案的明确批准后，才把临时草案写入：

```text
<repo>/fanloop-workflow/<workflow-id>/
|-- SKILL.md
|-- workflow.yaml
|-- condition.yaml
|-- flow.yaml
|-- loop.yaml
|-- prompt.yaml
`-- skills/<skill-id>/SKILL.md
```

范围扩大或 Schema 变化时重新对齐。Step 变化作为配置契约事实记录并自动校验，不另设 YAML 人工门禁。源根必须是绝对仓库路径，拒绝符号链接和不完整包；子 Skills 与资源保持在同一包内。

## 4. 校验并发布候选

使用兼容的现有 CLI、隔离配置和临时 Requirement Root 加载 Bundle，解析同根 Skill，至少初始化一次并验证
一条成功 Route 和一条回流 Route；不另写校验器或为配置改动构建新 binary。

逐项核对五份 YAML 引用闭合、所有 SkillBinding 都指向同目录已有 Skill、Route 可达且不含绕过 Human Gate
或 Panorama 的组合。记录候选完整文件清单与 SHA-256，按 second-brain-skills 的 `agent-governance/manage-fanloop-workflow/SKILL.md` 直接校验、提交、推送 `main` 和本地交付，不再启动维护 Workflow。`FANLOOP_CONFIG_SOURCE` 可用于候选检查，但最终必须回读用户默认环境实际绑定的源。默认配置随下一次新 flow init 拉取生效；显式作者源的编辑随下一次调用生效，无需 CLI 重建；未发生的推送或本地更新不能写成完成事实。
