# 一个逻辑命令只接受一个强类型 Request

每个 Fanloop 逻辑命令定义与命令同名的 `<Command>Request` 和 `<Command>Response`。调用方可用 typed flags 或完整 `--input` JSON 二选一构造 Request；Evidence 使用明确的 `source/content`。`--input` 支持内联、`@file` 和 stdin，严格拒绝未知字段、尾随 JSON 和类型不匹配。`root`、`input`、`dry-run` 是运行控制参数，不进入业务 Request。`schema list` 和 `schema describe` 是两个独立命令；写命令的 `--dry-run` 走相同业务计算但不产生副作用。

关于 `schema list` 与 `schema describe` 作为公开独立命令的决策由 ADR-0050 修订；
一个逻辑命令一个强类型 Request、typed flags/`--input` 二选一和运行控制边界保持不变。
