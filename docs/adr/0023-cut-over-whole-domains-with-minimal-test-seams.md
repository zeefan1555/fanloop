# 整域切换原生实现，只保留三个测试接缝

原生迁移按 `flow → loop → trace → card` 逐个完整命令域进行；一个域通过全部差分和能力契约后，由命令装配代码一次性从冻结版 Python 切到 Go，未迁移域继续走 `internal/legacy`，不增加运行时配置或 Feature Flag。四域完成后删除兼容运行时。Go 实现只为 Clock、Event ID 和 Lark CLI 子进程保留显式函数接缝，文件测试使用 `t.TempDir()` 和真实文件系统，不引入 Factory、依赖注入容器或 VFS。
