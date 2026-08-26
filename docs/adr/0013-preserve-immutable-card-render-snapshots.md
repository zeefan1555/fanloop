# 保留不可变 Card 渲染快照

每次非 dry-run `card render` 把确定性 payload 作为原始 Card JSON 保存为 `.fanloop/card/<timestamp>.json`，并在审计事件中记录精确相对路径和内容哈希。调用方必须使用本次响应的 `snapshot_path`，把该文件直接交给 `botmux send --card-file`，不能扫描目录猜测最新文件，也不能从响应中二次拼装。快照用于定位输出劣化和支撑真实飞书 E2E，但不参与 State 恢复，也不保存投递回执或重试状态。
