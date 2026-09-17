---
name: fanloop-dev-agent-acceptance
description: 对冻结 certification_head 做隔离安装，并由无源码上下文的全新 Sub-agent 使用公开 CLI 完成黑盒认证。
---

# Independent Black-box Verification

只接受工作树 clean 且 HEAD/tree 与冻结 certification identity 一致的候选。创建隔离
`FANLOOP_DATA_HOME` 和四类 Skill Root，清除 Botmux 环境，从 certification_head 执行
`./scripts/install-local.sh`。回读隔离 version、binary SHA-256、Doctor，并证明全局 current 未变。

派发恰好一个不继承实现上下文的全新 Sub-agent。它只能获得隔离 CLI 绝对路径、环境变量、
`fanloop-dev-verify/SKILL.md`、Feature Map 和 requirements.md 的 Acceptance Set；不得读取源码、内部 Go、
私有 helper 或历史 Requirement。先读取叶子 `--help`，再用公开 CLI 驱动全部 Acceptance Set，保存 argv、
stdout、stderr、退出码、Status/Event/文件与副作用证据。

`verify smoke` 只证明隔离安装、初始化、第一次 Route 推进、取证与清理，不能替代 Acceptance Set 或受影响
Feature 验收。Cleanup 后证据必须仍存在，全局 current 必须未变。

全部通过返回 passed 和原始 receipt；确定产品失败返回 failed；基础设施、权限或外部环境不可达返回
blocked。不得修改候选、push、建 PR、合并或更新全局 CLI。
