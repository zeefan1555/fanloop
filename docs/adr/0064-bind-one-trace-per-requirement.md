---
status: accepted
date: 2026-08-20
supersedes_in_part: ADR-0048
---

# 一个 Requirement 只绑定一个 Trace

Trace Integration 是 Requirement 的一次性绑定。首次 `trace bind` 提交 Trace 文档与
Registry；之后重复绑定同一文档和同一 Registry 返回 `unchanged`，绑定其他文档或切换
Registry 返回 `INVALID_ARGUMENT`，不修改 State、Event 或远端 Registry。

这里的“同一文档”沿用现有文档 identity：相同 host 与相同 docx/wiki token 视为同一
Trace，不要求 URL 字符串逐字一致。`flow init` 的自动 provision 与显式 `trace bind`
共同服从该约束。

本决策取代 ADR-0048 中“显式改绑后，当前 Trace 链接以 Integration 为准”的部分；保留
Trace Integration 权威、Bootstrap source 校验、Card 只读投影、本地事实先提交和远端
同步 best-effort 的其余边界。不新增 Requirement ID、迁移器、兼容层或远端旧行清理。

Runtime 测试必须隔离开发机真实的 `lark-cli` 和 `botmux`。只有显式安装在该测试临时
可执行目录中的 fake 可以被测试子进程调用；普通测试不能依赖或写入生产飞书资源。

验收契约：同一 Requirement 的首次绑定成功，同文档同 Registry 重复绑定幂等，异文档
或异 Registry 二次绑定失败且本地事实不变；在开发机已安装并登录真实 `lark-cli` 时，
普通 Runtime 测试也不会调用它。Workflow Step、Route、Condition 与 executor 不变。
