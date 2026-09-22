---
status: accepted
date: 2026-09-22
supersedes_in_part: ADR-0112
---

# 把 verify 失败模式写进 Skill，并关上 post-merge 枚举

**Status / Supersession:** optional `create-verification-skill` and `maintain-verification-skill` bindings on `verify_report_flow` are removed by [ADR-0114](./0114-keep-one-verify-path.md). `principle-prove-it-works` stays required. Transplanted pstack `SKILL.md` bodies stay unchanged.

Cut 1 的运行门禁仍是唯一硬约束。本决策把已知失败模式堆到那道门上。纠正顺序不变。代码库先于静态检查，然后是规则，然后是 Skill，最后是文风。不给 manifest 增加签名。

## 门禁

`post_merge_smoke_passed` 的 Output 增加已有 source `verify.evidence.outcome`，`values` 仍是 `[passed]`。Result 必须带恰好一条绝对路径、非符号链接、文件名为 `manifest.json` 的 file Evidence。文件必须是 schema 1，且 `outcome` 等于 `passed`。否则 `OUTPUT_INVALID`，不写 State。`when`、Step `id`、`name`、顺序和 `executor` 不变。

`main_agent_acceptance_passed` 仍是主 Agent 决定，不是 verify outcome，所以不加该 source。`self_validation_passed`、`remote_checks_passed` 与 `merge_tree_verified` 也不是这次 verify 运行的 outcome，保持原样。

## Skill

项目入口仍是 `.cursor/skills/verify-fanloop/SKILL.md`。它写明铺好的路径是 `fanloop verify smoke` 或 `fanloop verify snapshot` 留下的 schema-1 manifest，并写明 `OUTPUT_INVALID` 的恢复方式。手写 `manifest.json` 或在没有这次 verify 运行时上报 `passed` 是明确的反模式。

不改写移植来的 pstack `SKILL.md` 正文。`pstack-delivery` 的 `verify_report` 把已有 `principle-prove-it-works` 改为必选，Binding prompt 指向上述入口。`fanloop-maintainer` 的 `blackbox_verification_condition` 与 `post_merge_smoke_condition` 必选绑定已有 `fanloop-dev-verify`。pstack catalog 仍是 47 个 skill 目录。

手写但能通过 schema 1 解析的 manifest 仍能满足运行时。本刀不发明签名。Skill 禁止这条路。签名留给以后的结构变更。

## 取代

ADR-0112 “本刀只把该 source 写在三个已有 Condition 上”仅就 `post_merge_smoke_passed` 被部分取代。ADR-0112 的其余条款、ADR-0080 与 ADR-0038 的匹配规则仍然有效。
