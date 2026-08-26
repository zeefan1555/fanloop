# 分开维护 Skill、Fanloop Workflow 和 CLI 运行时

Fanloop 采用 Lark CLI 式单仓产品边界：`cmd/` 只装配四个命令域，`internal/` 承载公共运行时，`errs/` 固定错误契约，`workflows/` 保存版本化 Fanloop Workflow，`skills/` 只保存 Agent 指导，`tests/` 保存迁移差分、行为契约和端到端场景。只复用命令域、Help 协议、stdout/stderr 边界、结构化信封、类型化错误、版本漂移提示和领域测试映射；不引入认证、Profile、插件、API 元数据、分页、多格式输出或三层 API 调用。Skill、Fanloop Workflow、CLI、质量和发布维护者可以通过目录与 CODEOWNERS 独立评审；只有公开契约变化才要求跨边界协同。
