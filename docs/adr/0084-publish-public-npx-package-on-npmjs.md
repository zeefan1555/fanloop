---
status: accepted
date: 2026-08-29
amends: ADR-0009, ADR-0049, ADR-0065, ADR-0083
supersedes_in_part: ADR-0082
superseded_by: ADR-0085
---

# 在 npmjs 发布公开 Commonloop npx 包

Commonloop 源码仓库继续保持私有，配套包 `@zeefan1555/commonloop-cli` 改为在 npmjs 公开发布。
用户无需 GitHub PAT 或 registry 配置，统一使用一行入口：

```bash
npx @zeefan1555/commonloop-cli@latest install
```

包继续携带四个平台归档、Release Manifest、Workflows 与 Skills。公开 npm 制品不改变源码仓库
可见性，也不改变当前 `UNLICENSED` 声明。GitHub Packages 停止发布和读取，不增加双发布、镜像、
fallback 或旧 registry 兼容路径。

正式发布仍只由私有仓库的 `Release` GitHub Actions Workflow 手工触发，使用仓库的
`NPM_TOKEN` 发布公开 `candidate`。精确版本和真实 `latest` 都在不携带发布凭证的隔离 npm
配置中完成公开安装冒烟；只有精确版本通过后才提升 `latest`，最终冒烟失败时恢复旧 tag。

本决策恢复 ADR-0049 的匿名制品冒烟，取代 ADR-0082 的 GitHub Packages registry、私有包鉴权
与 `GITHUB_TOKEN` 发布边界，并修订 ADR-0065/0083 的 registry 描述。Commonloop 名称、npm package
identity、私有仓库、候选制品顺序、只向前 update、完整 Release 与摘要校验保持不变；Workflow、
Skill、Step、Route、Condition、Trace、Card、State 和 Thrift 契约均不改变。
