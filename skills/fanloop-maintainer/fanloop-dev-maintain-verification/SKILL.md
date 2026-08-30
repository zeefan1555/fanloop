---
name: fanloop-dev-maintain-verification
description: 按当前代码变化维护中文项目验证技能和功能地图，并真实运行受影响用户路径。
---

# 维护验证技能

项目唯一真值位于 `.agents/skills/verify-fanloop/`。不要另建根目录 Feature Map，也不要复制成第二套。

## 检查变化

1. 记录当前 HEAD、工作树、需求与变更范围。
2. 对照公开 Help、启动/安装入口、五份 Workflow YAML、IDL 和近期 Diff，找出新增、删除、改名或行为变化的用户表面。
3. 逐页核对 features/README.md 与 Feature 页面；删除已失效的入口和预期。

## 维护规则

- SKILL.md 必须覆盖 Launch、Doctor、Drive、Evidence、Cleanup。
- 每个 Feature 页面必须写用户入口、前置条件、真实操作、可观察结果、隔离边界和易错点。
- 全部说明使用中文；只驱动公开入口，不调用内部 Go 方法充当端到端证据。
- 使用隔离数据目录、全新 Requirement 和当前候选二进制。外部写入未经授权时停止，不回退用户身份。
- Cleanup 只移除一次性环境，证据保留在仓库忽略的验证目录。

## 结果

真实运行所有受影响 Feature。无需修改且全部通过为 clean；修正资产并通过为 updated。产品实现阻断、
无法覆盖或证据缺失时上报 changes_requested 或 blocked，不改地图掩盖产品问题。验证资产变化必须形成
新 HEAD，旧本地验证、Review、Eval、CI 与机器人验收全部失效。
