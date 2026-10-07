---
status: accepted
date: 2026-10-07
amends: ADR-0104
---

# 终止 Route 之后再投递一次 Done 全景图

2026-10-07 的 remnote-flashcards 跑完复盘后，用户只看到进入 `retrospect_run` 时的全景图（`In Progress`，`75%`）。Workflow 已经 completed，但没有人再把 renderer 的 `Done` `100%` 内容发给用户。操作者后来要求重渲染才看到完成态。

ADR-0104 的「最终普通回复不再重新 render 或重复 Panorama」只约束同一步里不要把进入时那张全景图再贴进后续回复。它不覆盖成功终止 Route 之后的完成态。本决定只改那一句的适用范围：进入 Step 时仍只展示一次，同一步内不重复；Workflow 变为 completed 之后，local_agent 以及任何必须把 Panorama `data.content` 发给用户的父宿主再 render 一次，把该次精确 `data.content` 原样送出。那一次必须是 `Done` 与 `100%`。复盘步骤里的 75% 全景图不是这次投递。没有用户可见通道则 blocked。不得把写入文件当成这次投递。

不新增 Condition，不改 Step id、顺序或 executor。Runtime 仍只校验回执类型与路由，不证明用户看见了卡片（ADR-0119）。强制点是全景图 Skill 里的这一句，以及 `scripts/validate-configuration.py --check-panorama-skills` / `--reject-missing-done-panorama`：

`after a successful terminal Route, render panorama once more and deliver exact data.content; the in-step retrospect panorama is not the Done 100% delivery.`

Skill 里若保留「最终回复不得重复 Panorama」，同一检查失败。
