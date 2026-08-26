# 把输出守卫检查放回各自责任边界

原生 CLI 不保留每次响应后重复运行的统一 `output_guard`：状态游标、产物门禁和 Route 在提交前校验，Check 豁免由 Workflow 与运行时分类器校验，Response 和 Card 由强类型输出器及契约测试保障，Trace 的完整健康状态由只读 `doctor` 检查。任何本地持久化错误仍阻止提交；`doctor` 不写 State 或 Event。这样保留旧六类保护，但每条规则只有一个实现位置。
