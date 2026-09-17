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

候选验收只安装到临时 `FANLOOP_DATA_HOME`，不改变全局 current。human 验收后，最终 Step 自动合并
唯一 PR，把本 Requirement 的源码 worktree 更新到 merge commit，并从该提交更新全局 current。

## 当前 Step

1. 只使用最新 `data.state.current` 的 prompt、skills、conditions、available_routes 与 outputs。每个 Skill 按 Status 给出的绝对 `SKILL.md` 完整读取。
2. 进入 Step 后立即按 `panorama_presented` 把 Panorama 作为第一条用户可见消息单独展示，不得先发进度前缀或人工问题。展示成功后同一轮继续；真正的人工问题只能随后发送，最终回复不重复 Panorama。未完成时上报 Progress；形成事实后从当前 Conditions 选完整 `when.any_of` 组合并显式选择 next/back/terminal。
3. 需求澄清只接受真实 human 决定；方案确认是 Agent 自主评审；最终验收 `confirm_human_acceptance` 是 Human Step。Developer 不得自批或将沉默当成通过。
4. 任意运行中 Step 只在 human 明确指定唯一目标时使用 `fanloop-dev-human-step-jump`。跳转只改变位置，不伪造被跨过 Step 的完成事实。
5. `review_code` 冻结 `review_base` / `reviewed_head`；Agent 验收使用一个无实现上下文的全新 Sub-agent 和隔离候选 CLI；人类验收后由 `merge_and_update_local` 创建或更新唯一 PR、等待 required checks、自动 squash merge，并更新源码 worktree 与全局 current。
6. 主分支、source HEAD、工作树或报告身份漂移时按 YAML 回流；仅 `merge_and_update_local` 中的 `origin/main` 正常前进留在本 Step 合入并重跑短门禁。PR 已合并后的本地失败只重试同一 merge commit。

## 最终回复

遵循通用 `fanloop-workflow` 的 Panorama-first 契约：Panorama 已在进入 Step 时单独展示，最终回复不重新
render 或重复。需要 human 输入时，只在 Panorama 下方展示当前完整问题。
