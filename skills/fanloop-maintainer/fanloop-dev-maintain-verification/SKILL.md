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

按 `fanloop-dev-verify` 的 Launch 模型建立一个隔离候选。Fanloop 是短生命周期 CLI：每个 Feature 使用
全新 Requirement Root，首次驱动及失败后运行 Doctor，逐项完成真实公开 CLI 路径。每轮清理临时状态，
同时确认已采集证据仍存在；最后执行 multi-surface journey。

## Triage

- 用户行为说明错误或缺失：doc drift，修正 Feature Map。
- 产品可用但验证入口无法稳定驱动：harness gap，记录所需最小控制面变更。
- 产品真实行为损坏：product gap，只报告证据，不改地图伪装成功。
- 公共 CLI、IDL、Workflow YAML 或推进语义需要变化：blocked，提交精确人审材料，不越过契约门禁。

## Outcome

`clean` 要求每个 Feature 都有 source 与 live 覆盖且无需修改。`changed` 只包含经过重新驱动证明的验证
Skill/Feature Map 修正，一次最多一个 PR。覆盖未完成、环境不可达或修正不能安全交付时为 `blocked`，
明确列出未覆盖 Feature、已尝试入口和所需外部变化。
