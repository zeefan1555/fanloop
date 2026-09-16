---
status: accepted
date: 2026-09-16
amends: ADR-0095
---

# 本地构建使用稳定语义版本

根级 `VERSION` 是本地 Release 的唯一版本真值，内容使用 `major.minor.patch`。干净源码构建的 `release_version` 与 CLI version 直接使用该值；精确源码身份继续由独立的 `commit_sha` 表达，不再把 commit、时间和进程号混入版本。

未提交源码构建使用 `<VERSION>-dev.<source-digest-prefix>`，使不同内容不会覆盖同一个不可变 Release，同时仍是可辨认的 SemVer 预发布版本。同一干净 commit 的重复构建使用同一版本并复用完全一致的 Release；修改正式内容但未更新 `VERSION` 时，安装器现有的不可变目录校验会拒绝覆盖。

本决策修订 ADR-0095 的本地构建 ID。它不修改 Thrift IDL、Manifest Schema、Workflow YAML、State Schema 或公开 `version` 响应字段，只改变这些既有字段的取值来源。

本变更选择 `e2e` 验证档：验证 VERSION 格式、干净与未提交构建版本、安装复用、Doctor、完整 unit 与 requirement E2E。
