---
name: fanloop-dev-workflow
description: 维护 zeefan1555/fanloop 自身的入口；纯 live Skill 配置变更直接交付，其他变更沿 Maintainer Workflow 推进。
---

# Fanloop Dev Workflow

## 适用边界

尚未初始化 Requirement 时，先检查预期 Git diff。若全部变更位于 `skills/**` 或 `exemplars/**`，且不需要
修改 Workflow YAML、CLI/Runtime、IDL、测试基础设施、构建、安装或发布逻辑，则这是纯 live Skill 配置
变更：不创建、不初始化 `fanloop-maintainer`，直接完成最小修改和受影响的聚焦检查，再按普通 Git/PR
流程交付。任何非 live 配置文件进入预期 diff，或影响面无法确定时，都必须启动完整自迭代流程。已经
初始化的 Requirement 不因后续范围缩小而废弃，继续按绑定 Workflow 推进。

始终执行：**读取 Status → 执行当前 Prompt/Skills → 上报 Progress 或 Result → 重新读取 Status**。`.fanloop`
只由 CLI 管理；构造命令前读取目标叶子 `--help`。

## 控制器

已有 Requirement 先按通用 `fanloop-workflow` 解析固定控制器。Status、Progress、Result、Doctor 与 Panorama
始终使用同一 `<REQUIREMENT_CONTROLLER>` 及其 `skill-roots/{codex,agent,trae,claude}`；不得回退到
`$HOME/.fanloop/current` 或手改 State 绕过 `WORKFLOW_MISMATCH`。新 Requirement 的 `flow init` 使用全局 current。
需要在全局 current 切换前保护已有 Requirement 时，使用本 Skill 的 `scripts/pin-controller-release.sh`。

候选验收只安装到临时 `FANLOOP_DATA_HOME`，不改变全局 current。本 Workflow 只交付 PR，不自动合并、
不发布、不更新本地 CLI。

## 当前 Step

1. 只使用最新 `data.state.current` 的 prompt、skills、conditions、available_routes 与 outputs。每个 Skill 按 Status 给出的绝对 `SKILL.md` 完整读取。
2. 进入 Step 后先按 `panorama_presented` 展示一次 Panorama。未完成时上报 Progress；形成事实后从当前 Conditions 选完整 `when.any_of` 组合并显式选择 next/back/terminal。
3. 需求澄清只接受真实 human 决定；方案确认是 Agent 自主评审；最终验收 `confirm_human_acceptance` 是 Human Step。Developer 不得自批或将沉默当成通过。
4. 任意运行中 Step 只在 human 明确指定唯一目标时使用 `fanloop-dev-human-step-jump`。跳转只改变位置，不伪造被跨过 Step 的完成事实。
5. `review_code` 冻结 `review_base` / `reviewed_head`；Agent 验收使用一个无实现上下文的全新 Sub-agent 和隔离候选 CLI；人类验收后由 `handoff_merge_request` 创建或更新唯一 PR、等待 required checks 并交接。
6. 主分支、source HEAD、工作树或报告身份漂移时按 YAML 回流；仅 `handoff_merge_request` 中的 `origin/main` 正常前进留在本 Step 合入并重跑短门禁。

## 最终回复

遵循通用 `fanloop-workflow` 的 renderer-owned 最终回复契约。结束一轮普通回复前紧邻执行：

~~~bash
<REQUIREMENT_CONTROLLER> flow status --root <ABSOLUTE_REQUIREMENT_ROOT>
<REQUIREMENT_CONTROLLER> card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown --dry-run
~~~

本轮最终普通回复必须完整原样展示 render 响应的 `data.content`。任一命令失败即以真实错误阻塞并停止；不得手工 fallback、复用旧 render 或快照。
