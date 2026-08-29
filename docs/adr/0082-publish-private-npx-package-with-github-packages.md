---
status: accepted
date: 2026-08-29
amends: ADR-0009, ADR-0049, ADR-0065
---

# 使用 GitHub Packages 发布私有 npx 安装包

Fanloop 源码进入私有仓库 `zeefan1555/fanloop`，配套 npm 包使用 GitHub Packages 的
`@zeefan1555/fanloop-cli`。包继续携带四个平台归档、Release Manifest、Workflows 与
Skills；不再发布无 scope 的 `fanloop-cli`，也不增加 npmjs、bnpm 或 Git URL fallback。

正式发布只由仓库的 `Release` GitHub Actions Workflow 手工触发。Workflow 使用当前仓库的
`GITHUB_TOKEN` 构建并发布 `candidate`，依次完成精确版本安装冒烟、`latest` 提升和真实
`latest` 安装冒烟；最终冒烟失败时恢复旧 `latest`。发布者不需要配置长期 npm token。

私有包的首次安装和 launcher 修复统一使用经过 GitHub Packages 鉴权的入口：

```bash
NPM_CONFIG_REGISTRY=https://npm.pkg.github.com \
  npx --yes --prefer-online --package=@zeefan1555/fanloop-cli@latest -- fanloop install
```

本机读取私有包需要 classic PAT 的 `read:packages` 权限。已安装用户仍执行 `fanloop update`，
launcher 从同一 registry 解析同一 scoped package，保留 ADR-0065 的只向前安装、同版本修复、
失败保留当前 Release 与输出契约。

本决策将 ADR-0009 的 bnpm/Luban 发布边界迁到私有 GitHub 仓库与 GitHub Packages，将
ADR-0049 的匿名制品冒烟改为同一 token 下的鉴权冒烟，并修改 ADR-0065 的 package identity
与 registry。候选制品、精确版本验证、latest 原子提升、完整配套 Release 和摘要校验保持不变。
