---
status: accepted
date: 2026-09-22
supersedes_in_part: ADR-0038, ADR-0109
amended_by: ADR-0116, ADR-0118
---

# 要求 Agent 验收带上存活的 verify 证据

**Status / Supersession:** the sentence that this cut writes `verify.evidence.outcome` on only three conditions is superseded in part by [ADR-0113](./0113-encode-verify-failures-into-skills.md). `post_merge_smoke_passed` uses the same source. The outer-loop sentence is specified by ADR-0116. The `pstack-delivery` step list below is superseded by [ADR-0118](./0118-replace-pstack-delivery-with-pstack-native.md); `delivery_verified` still requires a passed verify manifest.

Fanloop 采用 poteto kitchen 的顺序。验证先于生成。CI 全绿不是验收。纠正按硬度从代码库、静态检查、规则、Skill、文风指南依次落下。园丁负责删掉多余路径。外层自动化只在这道门可靠之后才开启。本决策只落第一刀：公开 `fanloop verify` 证据成为至少一条生产完成路径的运行门禁。

## 保留

ADR-0101 的公开控制面仍是 `fanloop verify doctor|smoke|snapshot|cleanup`。证据结论仍是 `passed|failed|blocked`。清理仍然保留证据。Feature Map 仍然不是第二套契约，真值仍是叶子 `--help`、五份生产 YAML 与可观察结果。地图仍在 `workflows/fanloop-maintainer/skills/fanloop-dev-verify/references/features/`。`.cursor/skills/verify-fanloop/` 只注册 Cursor 入口，`features` 符号链接到该地图，不复制文件。

ADR-0080 仍然禁止生产 Go 写死 Workflow、Step、Condition、Output key 或 Skill id。本门禁是通用 Output source，不识别具体 Workflow。

ADR-0109 的四步 `id`、`name`、顺序、`executor` 不变。`execute_agent_acceptance` 仍不恢复。黑盒仍在 `certify_candidate`。`blocked` 仍表示不提交能推进的 Result，不是新的 Route 事实。

`pstack-delivery` 的 Step 仍是 `intake`、`select_bind_skills`、`confirm_skill_plan`、`execute`、`verify_report`，名称、顺序与 executor 不变。

## 门禁

`idl/yaml.thrift` 增加作者常量 `OUTPUT_SOURCE_VERIFY_EVIDENCE_OUTCOME`，值为 `verify.evidence.outcome`。它只能出现在 `enum_value` Output 上，且 `values` 必须是 `passed`、`failed`、`blocked` 的非空子集。

Result 提交该条件时，运行时要求恰好一条 `source=file` 的 Evidence，其 `ref` 是绝对路径、非符号链接、文件名为 `manifest.json`。文件必须是 schema 1 的 verify manifest，且 `outcome` 等于本次上报的枚举值。不匹配则 `OUTPUT_INVALID`，不写 State。路由仍按 Condition id 匹配。Evidence 不选择 Route。

本刀只把该 source 写在三个已有 Condition 上，不改 `when`，不改 Prompt：

- `pstack-delivery` 的 `delivery_verified`（`values: [passed]`）。终态完成需要 outcome 为 `passed` 的 manifest。
- `fanloop-maintainer` 的 `blackbox_verification_passed` 与 `blackbox_verification_failed`。通过需要 `passed`，回 Build 需要 `failed`。`blocked` 的 manifest 两者都不能命中。

## 取代

只加 Prompt 文本不能阻止上报 `passed`。在 Go 里写死 `verify_report` 或 `certify_candidate` 违反 ADR-0080。新增 Step 或把 `blocked` 做成 Route 会改变人工维护的拓扑。把 Evidence 用来选 Route 违反 ADR-0038 的匹配规则。

ADR-0038 “CLI 不证明 Evidence 的内容质量”仅在本 source 上被部分取代。ADR-0109 “不增加新的 IDL”仅就这一个作者常量被部分取代。其余条款仍然有效。
