# Go CommandSpec 是命令 IDL 和错误目录的唯一真值

14 个逻辑命令在 Go `CommandSpec` 中绑定 ID、Request/Response 类型、读写模式、Requirement 范围、dry-run 能力、Workflow 依赖和稳定错误目录。`internal/schema` 使用固定版本的 `google/jsonschema-go` 从类型生成基础 JSON Schema，再叠加 Workflow 条件、说明和合法 examples。`schema list/describe` 直接读取这份真值；不维护手写 Schema 或第二套错误注册表。公开字段删除或改义必须更新 Contract Golden 并关联 ADR。
