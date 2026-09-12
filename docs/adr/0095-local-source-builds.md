---
status: accepted
date: 2026-09-08
supersedes: ADR-0082, ADR-0088
supersedes_in_part: ADR-0009, ADR-0024, ADR-0026, ADR-0032, ADR-0041, ADR-0049, ADR-0053, ADR-0054, ADR-0062, ADR-0065, ADR-0085
amends: ADR-0094
---

# 从源码只构建本机配套目录

Fanloop 目前只供用户本机使用。GitHub 托管源码；本机构建生成可直接运行的目录，停止 npm
包、npm launcher、GoReleaser、跨平台归档以及 GitHub 包发布。现有 CI 继续验证源码、公开
契约与隔离安装，不发布或上传分发制品。

## 本地构建与安装

`./scripts/build-local.sh [OUTPUT_DIR]` 使用本机 Go 编译 `bin/fanloop`，复制统一入口、
Workflow、Skills 及范文原始资料，并生成 `release.json`。脚本的标准输出只有构建目录的
绝对路径；默认在 `dist/` 下生成唯一 `local-*` 目录，显式输出目录必须尚不存在。

版本标识使用本地构建 ID：`local-<short-sha>[-dirty]-<UTC-time>-<unique-suffix>`，CLI
记录实际源码 commit。它用于定位构建内容，不表达远端最新版本或 npm SemVer 发布顺序。
范文中的正文、图片、画板预览、来源与导读均按原字节随 Skill 复制，不重复封装四份。

`./scripts/install-local.sh [OUTPUT_DIR]` 构建后调用现有目录安装。完整目录先经过摘要、
配套内容、Skill 入口与 Doctor 校验，成功才原子切换 `~/.fanloop/current`；任一环节失败
保留原安装。用户通过 `~/.fanloop/current/bin/fanloop` 直接运行。更新从选定源码提交
重新执行本地安装，不查询 registry、不使用 npm launcher 或 `latest`，也不保留旧入口
兼容层。只构建时不安装、不切换全局 current。

## Manifest 契约

`idl/release.thrift` 仍是 Manifest 的唯一可编辑真值，现有生成链同步 Go 类型和 validator：

- Manifest Schema 由 2 升为 3，`ReleaseManifest.schema_version` 的约束同步更新。
- `CLIRelease` 新增 field 2：`required string binary_sha256`，格式为
  `^sha256:[0-9a-f]{64}$`，直接校验本地 `bin/fanloop`。
- 删除分发专属 `PlatformAsset` 与 `ReleaseManifest.assets`；field 7 永久退休。
- 其他 Manifest field ID、Skill/Workflow Artifact 与 StateSchemaSupport 保持不变。

Manifest schema 3 不读取旧分发 Manifest；不伪造归档或平台，不绕过 Doctor，也不手改生成
代码。旧版本及其已有 Requirement 继续由原有配套版本解释。

## 保留的边界

本决策只收敛构建、分发、安装入口和对应验证：

- 保留 CLI、Workflow、Skills 与完整范文配套交付，二进制和内容摘要共同验证。
- 保留本地只读 Doctor、事务目录安装、失败恢复、用户路径保护和原子 current 切换。
- 保留 `~/.fanloop`、`FANLOOP_DATA_HOME` 与四个 Skill Root；四个客户端入口不是四平台构建。
- 保留 ADR-0057/0061/0079 的唯一全局 Workflow Skill、Release-bound 原子 Skill 路径与
  Workflow/Skill 一一对应，以及 ADR-0080 的运行时通用边界。
- 保留五份 YAML 的流程真值、Step 拓扑、Condition/Route 推进、Workflow digest 和
  Requirement State Schema。生产维护 Prompt 只把安装命令改为 `./scripts/install-local.sh`。
- 保留 ADR-0094 的精确候选、隔离 Sub-agent 验收、唯一 PR、合并检查和 merge commit 本地安装；
  本次只替换其中的 npm 安装调用，不改变审核、合并或更新 current 的授权边界。

ADR-0009/0026 中的平台集合、ADR-0032/0041 的 npm 入口、ADR-0049/0053/0065 的 registry
更新及 npm 候选发布、ADR-0054 的平台资产校验、ADR-0062 的 archive/npm 校验、ADR-0085
中的 GitHub Packages 发布条款均由上述本地目录方式取代；它们其他未冲突条款继续有效。
ADR-0085 的 Fanloop 产品身份、模块名、状态路径和当前 Workflow 身份保持不变。

## 审核与验收

用户明确要求“先改造好仓库代码，然后再去做端到端测试”“不要搞四个平台，现在就我自己
本地用”“都不需要打包 npm 发布，只在本地编译就行了，GitHub 只做代码托管”。在收到
Manifest Schema 3、`CLIRelease.binary_sha256`、删除 `PlatformAsset/assets` 的具体
前后 Thrift 与影响说明后，用户回复“可以的，去干吧”。

本变更选择 `e2e` 档：聚焦验证本机构建、完整内容摘要、坏摘要拒绝、Doctor 与目录原子
切换，再运行同一最终工作树的 `./tests/run-unit` 和 `./tests/run-e2e`。退役 npm 分发
专属测试；格式、生成物新鲜度、Go vet/test、Contract 与路由矩阵仍由原测试入口执行。
冻结同一候选后，以隔离本地构建的 CLI 创建全新技术方案 Requirement，验证 Step 产物与
范文可读性；真实人工门禁仍等待人的明确决定。
