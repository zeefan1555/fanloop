---
name: fanloop-dev-workflow
description: 维护 zeefan1555/fanloop 自身的入口；配置与代码沿同一四步 Workflow 分流；配置轻量校验后也完成 PR 合码及 live 同步。
---

# Fanloop Dev Workflow

## 四步，两条通道

仓库改造统一启动或继续 `fanloop-maintainer`。保留 Define → Build → Certify → Deliver 四个 Step，按完整实际 diff 选择通道，不因配置位于仓库而执行代码验证。

- **配置**：现有 Schema 下的 Workflow YAML、Skill 指令/参考材料及必要说明。当前 Prompt 提供轻量步骤，每次 Result 携带 `configuration_change`。
- **代码或混合改动**：可执行脚本、实现、IDL/Schema、Runtime、测试、构建、安装或发布逻辑。读取 `available_routes` 中 `code_change` Route 的原 Prompt，执行它的全部必需 Skills 与原有验证，每次 Result 携带 `code_change`。

只有用户明确只改本机 live 配置时才直接修改和回读，不创建仓库 PR。已有 Requirement 按最新 Status 推进；分类变化先回 Define，不能通过换通道跳过代码门禁。

## 配置通道

1. **Define**：用 `configuration_scope_recorded` 记录已授权目标、文件范围、实际默认分支基线、已有 live 目标/摘要与开始时间。范围明确就前进，不创建额外子 Agent、Spec、Tickets 或 Verification Contract。
2. **Build**：直接修改，清理冲突指引；用 `configuration_changes_written` 记录真实 diff。保留未授权的 Step 身份，不配套修改代码测试或 Golden。
3. **Certify**：检查相关引用、命令帮助与配置加载；需要时只验证受影响 Route。提交候选，用 `configuration_checked` 记录 HEAD/tree 和检查结果。无需 Go 全量测试、构建、Release 安装、代码 manifest 或独立代码验收。
4. **Deliver**：创建/复用唯一 PR，按已授权交付继续 squash 合并，核验 merge tree；更新本任务 worktree，从精确 merge commit 同步本次批准文件到 live，并回读摘要与 Doctor。提交所有真实交付 Condition 后 terminal；报告 PR、merge、live 路径和全程耗时，不能停在“工作区已修改”。

实际 required checks 仍须满足；未设置必需检查时不人为附加代码门禁。live 内容与修改前摘要不一致时停止覆盖，保留并发修改。已合码后的同步失败只恢复同一 merge commit；代码分支的 binary 安装不适用于配置分支。

## 代码路径

始终执行：**读取 Status → 执行当前 Prompt/Skills → 上报 Progress 或 Result → 重新读取 Status**。`.fanloop`
只由 CLI 管理；构造命令前读取目标叶子 `--help`。

## 控制器

已有 Requirement 先按通用 `fanloop-workflow` 解析固定控制器。Status、Progress、Result、Doctor 与 Panorama
始终使用同一 `<REQUIREMENT_CONTROLLER>` 及其 `skill-roots/{codex,agent,trae,claude}`；不得回退到
`$HOME/.fanloop/current` 或手改 State 绕过失败。新 Requirement 的 `flow init` 使用全局 current。
需要在全局 current 切换前保护已有 Requirement 时，使用本 Skill 的 `scripts/pin-controller-release.sh`。

## 四步职责

1. `define_verification_contract`：准备工作区，在 `requirements.md` 冻结目标、边界、Acceptance Set、
   Feature Impact Set 和主 Agent需求决定。批准前不得修改受管代码。
2. `build_until_verified`：执行子 Agent直接完成方案、实现、聚焦测试、`./tests/run-unit`、验证资产维护、
   真实 Feature 验证和自主修复，提交 clean 候选并记录 HEAD/tree/binary 身份。
3. `certify_candidate`：冻结候选，先派发不继承实现上下文的独立 Reviewer；Review 通过且候选身份未变后
   才派发黑盒 Verifier。两者通过后向主 Agent申请最终决定并记录 receipt。任何候选变化都回 Build。
4. `merge_and_update_local`：只交付已认证候选。main 前进时不修改代码，记录 `delivery_main_head` 后回
   Build 集成并重新 Certify；main 未变时等待 required checks、自动 squash merge、校验 tree、更新本 Requirement worktree
   和全局 current，并运行 Doctor 与 post-merge smoke。

沿用通用 `fanloop-workflow` 的 Panorama-first 契约：进入每个 Step 后，Panorama 必须作为第一条用户可见消息展示一次，
真正的人工问题只能随后发送；最终回复不重新 render、重复或改写 Panorama，需要人工输入时只在 Panorama 下方展示当前完整问题。
执行子 Agent持有 Requirement、Route 和受管实现，只向主 Agent回报；主 Agent负责需求批准和最终候选决定。
基础设施不可达保持 blocked，不把它伪造成产品失败。飞书文档只是可选阅读投影，不是 Route 必需事实。
