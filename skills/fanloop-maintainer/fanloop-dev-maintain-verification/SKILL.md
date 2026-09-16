---
name: fanloop-dev-maintain-verification
description: 审计并真实驱动 Fanloop 的 Verification Skill 与 Feature Map，修正验证资产漂移而不掩盖产品问题。
---

# Maintain Fanloop Verification

目标固定为 `../fanloop-dev-verify/`。覆盖单位是 Feature，不是句子；每个 Feature 都必须同时获得源码审计
和真实 CLI 覆盖。运行结论只能是 `clean`、`changed` 或 `blocked`。

## Index hygiene

读取 Feature Map README 与同目录 Markdown 文件。修正缺失、重复、额外或失效链接；不生成另一份根级
地图。索引中的每个 Feature 都必须使用规定的四个 H2，且能从用户视角说明入口与证明标准。

## Source wave

可用协作 Agent 时，每个 Feature 派发一个并行只读任务。每个任务返回：用户行为摘要、权威源码入口、
疑似漂移或 none、一个真实驱动配方。子任务不启动产品、不编辑文件。协调者逐一收齐结果，并检查近期
变更是否引入 Feature Map 未覆盖的用户表面。

## Live pass

按 `fanloop-dev-verify` 的 Launch 模型建立一个隔离候选，先运行 `fanloop verify doctor`。Fanloop 是短生命周期
CLI：每个 Feature 使用全新 Requirement Root，首次驱动及失败后运行 Doctor，逐项完成真实公开 CLI 路径，
用 `fanloop verify snapshot` 保存证据。每轮以 `fanloop verify cleanup --dry-run` 预览后清理临时状态，同时确认
已采集证据仍存在；最后运行 `fanloop verify smoke` 完成 multi-surface journey。

并行 live drive 优先把每个 Feature 放到独立远程环境；每个环境必须拥有独立候选安装、`FANLOOP_DATA_HOME`、
Requirement Root 和 evidence 目录。远程环境不可用时才回退本机，并保持同样的写状态隔离；不得并行共享
current、Requirement 或 evidence。

## Triage

- 用户行为说明错误或缺失：doc drift，修正 Feature Map。
- 产品可用但验证入口无法稳定驱动：harness gap，记录所需最小控制面变更。
- 产品真实行为损坏：product gap，只报告证据，不改地图伪装成功。
- 公共 CLI、IDL、Workflow YAML 或推进语义需要变化：blocked，提交精确人审材料，不越过契约门禁。

## Outcome

`clean` 要求每个 Feature 都有 source 与 live 覆盖且无需修改。`changed` 只包含经过重新驱动证明的验证
Skill/Feature Map 修正，一次最多一个 PR。覆盖未完成、环境不可达或修正不能安全交付时为 `blocked`，
明确列出未覆盖 Feature、已尝试入口和所需外部变化。

## Scheduled maintenance

每日任务执行 Index hygiene、Source wave 和 Live pass。`clean` 时静默结束；`changed` 时最多创建一个仅含
验证资产的 PR；`blocked` 或 `product gap` 时通知并附 run ID、候选身份、最早失败命令和 evidence 路径。
定时任务不修改产品代码，不合并 PR，也不使用真实用户凭据或 Botmux 目标。
