---
name: fanloop-dev-workflow
description: 维护 zeefan1555/fanloop 自身的入口；纯 live Skill 配置变更直接交付，其他变更沿四步 Maintainer Workflow 推进。
---

# Fanloop Dev Workflow

尚未初始化 Requirement 时，若预期 diff 全部位于 `skills/**` 或 `exemplars/**`，且不影响 Workflow、
CLI/Runtime、IDL、测试基础设施、构建、安装或发布，则直接完成最小修改和聚焦验证。其他变更使用
`fanloop-maintainer`。已初始化的 Requirement 继续按当前 Bundle 推进。

始终执行：**读取 Status → 执行当前 Prompt/Skills → 上报 Progress 或 Result → 重新读取 Status**。
`.fanloop` 只由 CLI 管理，命令输入以最新 Status 和叶子 `--help` 为准。

## 控制器

已有 Requirement 优先使用 `bound-release-home/current/bin/fanloop` 及其隔离 data/skill roots；不存在固定
控制器时才使用全局 current。候选认证只安装到隔离环境。最终 Deliver 固定 Requirement 控制器后，才把
已合并的精确 merge commit 安装为全局 current。

## 四步职责

1. `define_verification_contract`：准备工作区，在 `requirements.md` 冻结目标、边界、Acceptance Set、
   Feature Impact Set 和主 Agent需求决定。批准前不得修改受管代码。
2. `build_until_verified`：执行子 Agent直接完成方案、实现、聚焦测试、`./tests/run-unit`、验证资产维护、
   真实 Feature 验证和自主修复，提交 clean 候选并记录 HEAD/tree/binary 身份。
3. `certify_candidate`：冻结候选，并行派发不继承实现上下文的独立 Reviewer 与黑盒 Verifier；两者通过后
   向主 Agent申请最终决定。任何候选变化都回 Build。
4. `merge_and_update_local`：只交付已认证候选。main 前进时不修改代码，记录新 base 后回 Build 和
   Certify；main 未变时等待 required checks、自动 squash merge、校验 tree、更新本 Requirement worktree
   和全局 current，并运行 Doctor 与 post-merge smoke。

沿用通用 `fanloop-workflow` 的 Panorama-first 契约：进入每个 Step 后，Panorama 必须作为第一条用户可见消息展示一次，
真正的人工问题只能随后发送；最终回复不重新 render、重复或改写 Panorama，需要人工输入时只在 Panorama 下方展示当前完整问题。
执行子 Agent持有 Requirement、Route 和受管实现，只向主 Agent回报；主 Agent负责需求批准和最终候选决定。
基础设施不可达保持 blocked，不把它伪造成产品失败。飞书文档只是可选阅读投影，不是 Route 必需事实。
