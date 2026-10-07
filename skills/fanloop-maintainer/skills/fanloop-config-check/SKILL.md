---
name: fanloop-config-check
description: 按实际语义检查配置，普通模板自检，风险变更独立试用；仅在 Maintainer Certify 配置 Route 使用。
---

# 检查配置

对照 `configuration_scope`、实际 diff 和 delivery-record.md 的原始证据。核验已完成检查，不重复未失效的工作；代码混入或目标/验收/权限/副作用扩大回 Define，配置缺陷用 `configuration_needs_fix` 回 Build。同目标关联说明补齐清单即可。

## 选择检查范围

- 错字、排版：检查 diff 和受影响引用，行为不变可有据免测。
- 模板、生成要求及 Prompt 文案：检查相关引用、必要命令帮助；用现有兼容 CLI 在隔离完整源加载受影响包，检查实际代表性结果。仅改 YAML 文案不跑路由矩阵。
- 全景图展示 Skill：运行 `python3 scripts/validate-configuration.py --check-panorama-skills --config-source <绝对源>`。缺少 `report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.`，提到 `local_agent` 却缺少父/子交接句子（以 `The child must not submit panorama_presented.` 开头），缺少 `after a successful terminal Route, render panorama once more and deliver exact data.content; the in-step retrospect panorama is not the Done 100% delivery.`，仍写「最终回复不得重复 Panorama」，或把写入 `artifacts/panorama-*` 当成投递时，命令失败并写出 Skill 路径和应补的句子。粘贴的 evidence 用 `--reject-file-save-evidence <文件>`；`content saved artifacts/panorama-` 必须失败。复盘后停住的转录用 `--reject-missing-done-panorama <文件>`；停在 75% 且 workflow completed、没有后续 `` `Done` `100%` `` 必须失败。CLI 不证明用户看见了消息。
- 学习卡契约：运行 `python3 scripts/validate-configuration.py --check-learning-card-gates --config-source <绝对源>`。组合卡交易黑话用 `--reject-combo-slang <文件>`，`create --content-file` 加 `[[id:]]` 用 `--reject-combo-markdown-ref <文件>`。两条都必须能在 2026-10-07 的失败样本上失败，并指出 `plain_words`/`one_pass` 或 `{"remId":"..."}` / `object-set-text`。
- 复盘经验表：运行 `python3 scripts/validate-configuration.py --check-notion-experience-sync --config-source <绝对源>`。`retrospect_run` 的 prompt、学习卡契约和 `references/notion-experience.md` 必须包含 `When findings is non-empty, POST one page per finding with ntn api v1/pages to data source 「remnote workflow 经验表」; zero findings skip the write; a failed ntn call records notion_sync and still allows retrospective_recorded.`，以及字段映射、`ntn login` / `NOTION_KEYRING=0`、`FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID` 这三句。参考文件还要保留默认 database id `80b2f42e-0e34-4d21-ab84-245bd4404f13` 和 data source id `ac9ebbdb-47b5-4f87-b821-df8a3eba5509`。缺任一句或默认 ID 时命令失败并写出路径。
- Step、Condition、Route、绑定：严格加载和引用校验，运行 `python3 scripts/validate-configuration.py --config-source <绝对源> --binary <已有兼容CLI> --output-root <新证据目录> --workflow <受影响ID>`；多个 ID 重复传参。不跑无关 Workflow，不手工重造解析器错误用例；加载器回归仍由代码测试负责。

路由/绑定变化还需确认两类互斥、代码原业务门禁有效、所有正常/终止/回流 Route 保留 Panorama。完整包及完整源摘要、候选未变化检查始终保留。禁止为验证指引创建未授权真实业务数据。

## 核验实际行为

1. 按最终 diff 复核 `trial_plan.method`。`self_check` 由当前 Agent 实际执行最小代表性样例及必要失败边界，回读原始结果，不能只复述改动。`not_applicable` 写行为未变的理由，outcome 也为 not_applicable，不伪装 passed。
2. 路由、绑定、权限、安全、审批或外部写入规则变化使用 `independent`：固定摘要后派发一名不继承实现上下文的全新 Agent，只给自然任务、验收点、Skill 实际入口、工具/数据、允许副作用与停止位置，不给 diff、实现答案或旧结论。已有兼容 CLI 即可，无需构建安装；环境或授权不足保持 blocked。
3. 独立 Agent 真正使用隔离副本，记录 Skill 路径、argv/stdout/stderr/退出码、文件变化及状态；不得改冻结候选、真实笔记、全局配置或发布真实 PR。试用按授权停止，避免递归启动无界验收。当前 Agent 回读原始产物逐项核验，不只接受子 Agent 结论。
4. 确认被检查文件摘要未变、没有越权后才记 passed。行为失败回 Build；环境或权限不足保持 blocked。候选变化使相关旧检查失效，重查受影响部分；独立试用覆盖的候选变更后须重新独立试用。

## 完成

同一 delivery-record.md 记录仓库、Workflow、Git 基线、完整包摘要、检查命令/结果和原始证据引用。`configuration_checked` 返回上述身份和 `trial`：method、outcome、理由、场景覆盖、证据路径；只有独立试用填写真实 Agent 标识。failed/blocked 不能上报成功。

不执行 Go 全量、Golden 更新、代码 CR/黑盒、构建、安装或代码 manifest；不生成第二份阅读报告。证据保留在 Requirement Root 或仓库外，清理授权临时对象后仍保留真实证据。
