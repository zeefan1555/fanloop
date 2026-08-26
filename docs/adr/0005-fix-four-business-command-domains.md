# 固定四个业务命令域

Fanloop 面向 Agent 的业务命令固定分为 `flow`、`loop`、`trace`、`card` 四个域，并分别放入独立命令目录；`schema`、`version`、`doctor`、`update` 只是根级运维命令。行为等价阶段保留旧 Driver 已有子命令、别名、退出码和副作用，不能以重新分层为由删除能力；隐藏的旧入口只进入兼容夹具，不继续作为推荐用法。

## 兼容阶段命令面

- `flow`：`init`、`update`、`approve`、`status`、`migrate`
- `loop`：`feedback`
- `trace`：`bind`、`record`、`render`、`sync`、`status`、`migrate`，以及 `list` 别名
- `card`：`render`、`panorama`
