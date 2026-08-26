---
status: accepted
date: 2026-08-17
---

# 用已验证制品修复不健康的同版本本地 Release

官方 npm 安装器在完成 Manifest、平台归档摘要和 staged Doctor 校验后，可以用同一版本的已验证制品事务性替换不健康的本地 `releases/<version>`。这只修复用户机器上的 Release 缓存，不复用或改写 bnpm 版本；健康目录继续复用，失败时恢复原目录、`current` 和 Skill 软链，用户拥有的普通路径继续拒绝覆盖。发布 Skill 的运行入口不得向 Release 目录写入 bytecode cache。

本决策修订 ADR-0026 对本地不可变目录的解释：不可变性约束版本身份和健康内容，不要求永久保留已损坏的本地副本。保留 ADR-0009 的完整制品校验、ADR-0024 的只读 Doctor 与整包切换、ADR-0032 的目录和单行成功输出。
