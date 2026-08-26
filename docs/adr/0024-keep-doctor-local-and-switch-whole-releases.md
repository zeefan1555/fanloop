# Doctor 保持本地只读，Update 只切换完整发布

Doctor 不访问网络，也不自动修复；它按稳定 ID 返回 installation/requirement 范围内的 passed、warning、failed 或 skipped check，并聚合为 healthy、warning 或 unhealthy。`update --action check` 单独查询远端版本；`--action update` 安装最新配套发布；`--action switch --target-version <version>` 切换指定版本。CLI、Skills 和 Workflows 必须作为一个由 `release.json` 与 SHA-256 固定的整体暂存、诊断并切换，任一组件失败都保持当前版本。
