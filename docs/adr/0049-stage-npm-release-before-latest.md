---
status: accepted
date: 2026-08-17
amends: ADR-0009, ADR-0046
amended_by: ADR-0053, ADR-0061
---

# 验证 npm 候选制品后再提升 latest

Fanloop 的内部 npm 发布先把不可变版本写入 `candidate` dist-tag。Luban 的认证
发布命令随后对该精确版本执行完整匿名安装冒烟；只有版本、commit、Manifest、
Doctor、默认 Workflow、持久 launcher 和三套 Skill 根目录全部匹配，才把
`latest` 原子指向候选版本，并用真实匿名 `@latest` selector 再执行同一冒烟。

认证发布命令在开始前记录旧 `latest`。候选发布或精确版本冒烟失败时不修改
`latest`；提升后的最终冒烟失败时恢复旧 tag 并让发布失败。候选版本一旦发布
仍然不可变，不覆盖、不复用，也不通过重复 Publish 修复。

正式 `fanloop-workflow` Skill 在新会话开始时显式执行
`fanloop update --action update`，由当前 Go CLI 直接解析最新配套包、校验并原子
切换完整 Release；它不再让已安装用户重复经过 npx 的 `@latest` packument 缓存。
`npx --prefer-online ...@latest -- fanloop install` 继续作为首次安装和持久 launcher
修复入口；发布版本解析同样要求在线重校验完整版本列表。

本决策修订 ADR-0009 中发布即更新 `latest` 的顺序，以及 ADR-0046 中每次 Workflow
会话都调用 npx 的入口。ADR-0009 的 Luban 唯一发布边界与完整配套制品、
ADR-0024 的 Doctor 和整包切换、ADR-0026/0032 的目录和输出、ADR-0041 的同版本
本地修复语义保持不变。不增加用户缓存清理、无限重试、后台更新、bootstrap 包、
registry 代理或兼容路径。

完整症状、平台证据和验收条件见 `docs/specs/atomic-latest-release.md`。
