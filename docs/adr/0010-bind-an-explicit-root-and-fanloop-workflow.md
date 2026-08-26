# 初始化显式绑定需求目录与 Fanloop Workflow

Agent 调用以绝对 `--root` 指定活动需求根目录，`flow init` 同时要求 `--workflow <id>` 并把发布包解析出的准确版本写入状态；后续命令不重新猜测。人类可以从当前目录向上发现已有 `.fanloop`，但初始化不使用该便利行为，也不再依赖需求内 CLI 快照推导目录。
