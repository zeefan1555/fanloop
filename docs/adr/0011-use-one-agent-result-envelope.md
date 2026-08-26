# 四个命令域使用统一 Agent 结果信封

Fanloop 的成功结果只向 stdout 输出 `ok/data/meta/_notice` JSON，普通错误只向 stderr 输出包含稳定类型、代码、消息、恢复提示和可重试标记的 JSON。多目标操作发生部分失败时，借鉴 Lark CLI 在 stdout 返回 `ok:false` 和完整逐目标结果，同时使用稳定非零退出状态且不写 stderr；Flow 的附带自动同步失败仍作为 `_notice`。四个业务域共享这一协议，不引入表格、CSV、分页或认证身份字段。
