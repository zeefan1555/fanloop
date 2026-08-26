---
status: amended by ADR-0032
---

# 用一个 current 软链切换完整安装版本

macOS 和 Linux v1 把每个配套版本安装到用户数据目录的不可变 `releases/<version>`，npm 启动器以及 `.codex/skills`、`.agents/skills`、`.trae/skills` 的官方 Skill 软链都通过唯一 `current` 指向该版本；更新完成时只替换 `current`，避免 CLI、Skills 和 Workflows 混装。默认用户数据目录及成功输出由 ADR-0032 修订。首次安装遇到同名外部软链时，安装器接管该入口，但只替换软链本身，不修改或删除原链接目标；后续步骤失败时恢复原软链。普通文件和目录仍视为用户内容并拒绝覆盖。v1 只发布 macOS/Linux 的 amd64、arm64 Asset，Windows 等出现真实需求后再设计。
