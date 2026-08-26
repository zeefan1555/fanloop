# 在独立 Fanloop 仓库继续开发

Fanloop CLI 的代码、测试、技术方案和 ADR 统一维护在独立 Codebase `github.com/zeefan1555/fanloop`。迁移期曾使用冻结版 Python Driver 做差分基线；四域切换完成后该快照随兼容运行时一起删除，只保留能力清单、长期 Contracts 和 ADR。知识树原仓库不再承载 Fanloop CLI 的后续开发。
