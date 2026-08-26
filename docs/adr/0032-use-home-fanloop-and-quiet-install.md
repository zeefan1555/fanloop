---
status: accepted
date: 2026-08-12
---

# 默认安装到 ~/.fanloop 并保持成功输出单行

Fanloop 在 macOS 和 Linux 上统一把匹配 Release、`current` 软链及相关安装数据放到 `~/.fanloop`，不再选择 `~/Library/Application Support/fanloop` 或 XDG data 目录；`FANLOOP_DATA_HOME` 仍可用于测试和非默认客户端。安装器直接切换到新默认目录，不读取、迁移或删除旧目录。用户执行 `npx ... fanloop install` 成功时只输出 `Fanloop <version> installed successfully`；内部安装 JSON 与持久 npm launcher 的成功日志保持静默，失败时仍返回原始错误。该决策修订 ADR-0026 的默认目录和用户输出部分。
