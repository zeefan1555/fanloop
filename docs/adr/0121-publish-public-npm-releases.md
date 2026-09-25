---
status: accepted
date: 2026-09-25
supersedes_in_part: ADR-0085, ADR-0095
amends: ADR-0049, ADR-0065, ADR-0120
---

# 从 GitHub 发布公开 npm 包

Fanloop 保留 GitHub 仓库 `zeefan1555/fanloop`，通过 GitHub Actions 向 npmjs.org 公开发布
`@zeefan1555/fanloop-cli`。发行端构建 darwin/linux 的 amd64/arm64 二进制；客户端只需要 Node.js 18+
与系统 tar，不需要 Go，也不需要 GitHub Packages 或字节内部 registry 配置。

npm 包只携带 CLI、统一入口、平台归档和校验信息。生产 Workflow 继续以 Fornax 已发布包为唯一真值；
安装器在没有 `FANLOOP_CONFIG_SOURCE` 时读取用户的 `fornax.json` 并同步配置，隔离验证才显式使用固定
fixture。发布先验证本地 tarball，再发布 candidate、验证精确版本、提升 latest 并复验；失败时保留或
恢复旧 latest。

唯一发布入口是 GitHub `master` push。`VERSION` 只定义版本线，稳定 patch 从 npmjs.org 已发布版本分配，
不提交自动版本 bump。发布凭据只使用 GitHub Actions 的 `NPM_TOKEN` secret，不进入仓库。
