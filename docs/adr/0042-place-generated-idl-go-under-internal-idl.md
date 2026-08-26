---
status: accepted
date: 2026-08-17
---

# IDL Go 派生实现统一放在 internal/idl

可编辑 Thrift 继续位于根级 `idl/`，thriftgo 和 validator 生成的七个领域 Go 包统一位于 `internal/idl/<name>idl/`；生成的 CommandSpec 与其运行载体继续位于 `internal/idl` 根包。根级旧 `*idl` 包直接删除，不保留兼容 import 路径。生成新鲜度门禁必须同时验证完整生成文件集合与内容。

本决策补充 ADR-0040 的生成物布局，不改变 Thrift-first 真值、公共 CLI JSON 契约、Service 接口、错误目录、Workflow YAML 或运行语义。
