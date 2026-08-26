---
status: accepted
date: 2026-08-20
amends: ADR-0009, ADR-0024, ADR-0040, ADR-0049, ADR-0050, ADR-0053
---

# 由 npm launcher 负责显式 update 子命令

Fanloop 的已安装用户只使用 `fanloop update` 更新到官方 latest。持久 npm
launcher 在进入当前 Go payload 前拦截该无参数子命令，从唯一空临时目录在线解析
`https://bnpm.byted.org` 的
`fanloop-cli@latest`，并执行该候选包自身的 `fanloop install`。候选包
继续通过 Manifest、平台归档摘要、staged Doctor、Skill 软链预检和原子 `current` 切换
安装完整配套 Release；失败保留当前 Release。首次安装和旧版本恢复仍使用同一官方
npm install 入口。

`fanloop update` 只向前推进。launcher 向候选包的 `fanloop install` 传入
`FANLOOP_UPDATE_FORWARD_ONLY=1`；候选安装器读取 `current/release.json` 的
`release_version`，候选版本严格低于已装版本时保留现状，不解包、不改 Skill 软链、不切换
`current`。版本比较只接受稳定 SemVer；已装 Manifest 缺失、不可读或任一侧不是稳定
SemVer 时按未命中处理继续安装，候选与已装版本相同时同样继续安装，保留 ADR-0041 的
同版本本地修复。不带该环境变量的直接 `fanloop install` 保持无条件安装，`latest`
回滚后经由该入口回退。该环境变量只由 launcher 的 `update` 分支设置，不是用户可配置项，
也不进入 Thrift 契约。

旧 Go CLI 不再读取和解释未来 npm 包的 Release Manifest。`fanloop update --action
check|update|switch`、历史版本 switch、可用性 check、自维护 npm 下载/解包、latest cache
和 `_notice.update` 在同一边界直接删除，不保留 alias、兼容解析、fallback 或双实现。
`doctor` 与 `_notice.drift` 保留。根 `--update` 也直接删除，不作为新子命令的 alias。
已经发布的旧 launcher 和二进制无法获得新入口，受影响用户需要一次官方 npm
重装，随后使用 `fanloop update`。

`fanloop update` 属于安装 bootstrap control，不是 Thrift 业务方法，不使用统一
Agent JSON 信封。npm launcher 只接受精确的无参数子命令，透传候选 installer 的
stdout、stderr 和退出状态，成功输出在安装时保持 `Fanloop <version> installed successfully`，在保留已装 Release 时为
`Fanloop <version> is already up to date`。
Go 根 Help 仅保留该 bootstrap 子命令的可发现占位；绕过 npm launcher 直接运行
payload 时明确拒绝更新。Ops Thrift 删除
Update Request/Response、action/effect enum 和 Service 方法；仅供该命令使用的两个错误码
以及 typed UpdateNotice 同时退役，数值和 field ID 永不复用。公开 Thrift/Cobra 叶子命令
由 12 个变为 11 个；launcher bootstrap 不计入该目录。

本决策取代 ADR-0024 的 check/update/switch 公开面和“当前 Go CLI 负责整包切换”的边界，
修订 ADR-0049 中已安装用户绕过 npx 的决定，修订 ADR-0040/0050 的公开命令数量与所有
公开更新都必须是 Thrift 叶子的范围，并将 ADR-0053 的初始化前刷新入口替换为
`fanloop update`，保留其只在新 Requirement 初始化前刷新的时机。保留 ADR-0009 的配套 Release 与发布门禁、
ADR-0026/0032 的版本目录和成功输出、ADR-0041 的同版本本地修复、ADR-0049 的
candidate→精确冒烟→latest→真实 latest 冒烟顺序、ADR-0054 的 Manifest 真值以及
ADR-0055 的双测试入口。Workflow YAML、Release Manifest、State/Event 和 Requirement
语义不改变。保留 ADR-0062 的当前 Workflow Bundle 与 digest 边界；验证按 ADR-0063 的
`e2e` 风险档执行。公开行为由根命令 Help、npm launcher 测试、发布冒烟与对应契约测试共同验收。
