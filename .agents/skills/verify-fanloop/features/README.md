# Fanloop 功能地图

这里是验证 Fanloop 用户可见 CLI 行为的唯一功能地图。先执行父 Skill 的 Launch 与 Doctor，再选择
对应 Feature 页面。

## 基线

- 记录待验证 checkout 的 HEAD；Go 1.23+ 可用，安装验证还需要 Node.js 18+。
- FANLOOP_VERIFY_BIN、FANLOOP_VERIFY_REQUIREMENT、FANLOOP_VERIFY_EVIDENCE 来自父 Skill。
- Requirement 与 FANLOOP_DATA_HOME 必须一次性且相互隔离。
- 本地配方清除 BOTMUX_CHAT_ID 与 BOTMUX_SESSION_ID；外部 Trace/Lark 写入默认不授权。

## 驱动与证据

- 先读叶子 --help 和最新 flow status，不猜请求或 Route。
- 只驱动公开 CLI；测试可补充覆盖，不能替代真实用户路径。
- 保存命令、退出码、输出、前后状态与 commit；写入操作必须有二次只读回读。
- 外部路径不可达时记录缺少的授权、身份、权限或服务，不用本地 mock 冒充通过。
- Cleanup 只移除一次性 session，证据保留在 tests/requirement_e2e/runs/verify-skill/。

## Feature

- [Requirement 与 Flow](requirement-flow.md)：初始化、Status 驱动、dry-run 与推进证明。
- [Output 与 State](output-state.md)：Output 归属、持久化和回流失效。
- [Trace](trace.md)：本地状态与投影，以及远端同步授权边界。
- [Card](card.md)：current/panorama 的 Markdown、Lark JSON 与快照。
- [安装、Release 与 Skills](installation-release.md)：隔离安装、version、Doctor 与 Skill 闭包。
