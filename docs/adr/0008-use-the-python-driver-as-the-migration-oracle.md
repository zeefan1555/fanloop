# 用旧 Python Driver 作为迁移期间的行为基线

原生 Go 实现必须与旧 Driver 运行同一组黑盒场景，比较命令输出、退出码、Route、状态变化和关键产物；当前 140 个 Python 测试是提炼场景的证据来源，不由新 Go 测试自行重新定义旧行为。`tests/parity/` 只在迁移期运行 Python 与 Go 精确差分；`tests/contracts/` 长期验证人工门禁、Loop、事务、Trace 和 Card 等能力，不绑定实现。行为等价通过后删除 Python 运行时，长期门禁只保留实现无关的能力契约。
