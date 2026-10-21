---
status: accepted
date: 2026-10-15
supersedes: ADR-0122
supersedes_in_part: ADR-0095, ADR-0123, ADR-0132
amends: ADR-0108
amended_by: ADR-0137-take-workflow-configuration-from-fornax, ADR-0138-refresh-cli-and-git-config-on-fresh-init, ADR-0146-persist-launcher-path-and-bound-release-downloads
---

# 从 GitHub Release 分发 CLI

用户安装 Fanloop CLI 的方式改为 GitHub Release 上的平台归档和一层薄 npm 包装。不使用 npm registry，也不把包装发布到 GitHub Packages。仓库内的 `./scripts/build-local.sh` 与 `./scripts/install-local.sh` 继续服务本地开发。

推送形如 `vX.Y.Z` 的 tag 时，GitHub Actions 只在 `ubuntu-latest` 上交叉编译 darwin/linux 的 amd64 与 arm64。tag 过滤是 `v[0-9]+.[0-9]+.[0-9]+`。一次发布上传：

- 每个平台一个 `fanloop-<version>-<os>-<arch>.tar.gz`
- `checksums.txt`
- 分发清单 `release.json`（schema 1）
- `npm pack` 生成的 `fanloop-<version>.tgz`

安装命令是：

```bash
npm i -g --foreground-scripts --allow-scripts=fanloop https://github.com/zeefan1555/fanloop/releases/download/vX.Y.Z/fanloop-X.Y.Z.tgz
```

`--allow-scripts=fanloop` 允许较新的 npm 运行 postinstall。`--foreground-scripts` 让安装输出可见；npm 默认会藏起成功的 postinstall 日志。包装里的 `scripts/install.js` 按本机平台选择归档，用 `checksums.txt` 校验 sha256，并把下载限制在 256MB。归档里的符号链接、硬链接和设备文件会被拒绝。Pax 扩展头和 macOS AppleDouble（`._*`）元数据会被跳过，不进入发布目录。内部 schema 4 `release.json` 和二进制、Skill 目录摘要全部校验通过后，才把内容发布到 `~/.fanloop/releases/<version>`。然后创建 Codex、Agent、Trae、Claude 的公共 Skill 链接，目标是 `current/<skill path>`，最后才原子切换 `~/.fanloop/current`。校验或链接失败时不切换 `current`。同一版本再次安装时比较整棵发布树；内容被改过则用已校验归档事务替换，而不是只比较二进制摘要。

`scripts/run.js` 执行 `current/bin/fanloop`。npm 全局 bin 已在 `PATH` 上时，安装后可以直接运行 `fanloop`；否则安装程序打印一条可直接执行的 `export PATH=...`。分发安装的 `current` 是符号链接且 `fanloop` 不在 PATH 上时，`fanloop doctor` 给出 warning，hint 里是同一条 `export PATH=...`。`FANLOOP_DATA_HOME` 必须是绝对路径。仓库仍是私有时，下载 404 会说明需要公开仓库或认证；安装器不发送 token。

`fanloop update` 只下载最新 GitHub Release，校验后切换 `current`。它不拉取 Workflow 配置。已安装的语义化版本比 `latest` 新时不下载归档、也不降级。`fanloop update config` 只同步 `second-brain-skills` 的 `main`。它在整个同步期间持有配置锁；`git ls-remote` 与当前快照 commit 一致时跳过 checkout。两条路径都打印每个 Workflow 的 id 和同步到的 commit。Workflow 没有独立版本号，这个 commit 就是该次同步的版本。

`FANLOOP_DATA_HOME` 仍然是数据根。显式 `FANLOOP_CONFIG_SOURCE` 继续让普通命令读取作者仓库；`update config` 更新的是数据根里的生产快照。

开发安装保持 ADR-0108 的稳定目录：`current` 是目录，`current/release` 指向 `releases/<version>`。分发安装的 `current` 是指向 `releases/<version>` 的符号链接。若 `current` 已是目录，`fanloop update` 和 postinstall 拒绝覆盖，避免破坏本地开发安装。

平台包内部的 `release.json` 仍是 `idl/release.thrift` 的 schema 4，不含平台归档字段。GitHub Release 根上的 `release.json` 是另一份分发清单，不进入该 Thrift 契约。

本决定取代 ADR-0122 的“只从源码本地安装、不发布安装制品”。它部分取代 ADR-0095 里停止跨平台归档和 GitHub 发布的条款，并部分取代 ADR-0123 中“ADR-0122 的本地源码分发保持不变”这一句。ADR-0132 里“不增加 update 命令”的条款由这里的两个命令取代；配置真值、不可变快照、完整包校验、源锁和代码/配置分离保持不变。合并后的代码验证仍可从精确 merge commit 执行 `./scripts/install-local.sh`。
