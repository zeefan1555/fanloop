# Trace

用户可检查本地 Trace 状态并渲染已提交 Event；远端 Lark 投影是另一项需要明确授权的操作。

## 用户入口

- fanloop trace status --root ROOT
- fanloop trace render --root ROOT --dry-run 或真实 render
- trace bind/sync 只在单独授权远端写入时使用

## 真实驱动

1. 在全新 Requirement 提交至少一个 Flow 结果，并保持 Botmux 环境为空。
2. 保存 trace status，要求本地 Event 数与 events.jsonl 一致；未绑定是合法本地状态。
3. 记录 events.md 是否存在，执行 trace render --dry-run，要求不创建或改变投影文件。
4. 真正 render 后要求 events.md 包含 Requirement 标题和已提交推进；`fanloop-maintainer` 还必须展示
   五个 Stage、十二个 Job、十六个 Step，并保留多 Job Stage 的 Job 名称与边界。
5. 没有远端授权时记录 not authorized 并停止；本地 render 不证明远端同步。

Requirement 的只读和 dry-run 命令也会追加 CLI 日志。机器人验收的内层 CLI 必须保持未绑定，不得
生成 Trace Integration、远端 Trace Event 或用户文档。
