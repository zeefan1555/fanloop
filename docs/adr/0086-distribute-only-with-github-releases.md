---
status: accepted
date: 2026-08-30
amends: ADR-0009, ADR-0032, ADR-0041, ADR-0046, ADR-0049, ADR-0053, ADR-0065, ADR-0085
supersedes: ADR-0082
---

# 只使用私有 GitHub Releases 分发 Fanloop

Fanloop 的唯一正式分发源改为私有仓库 `zeefan1555/fanloop` 的 GitHub Releases。停止使用
npmjs、GitHub Packages、任意 npm registry、package publish、dist-tag 和 `npx`；不保留双发布、
镜像、registry fallback 或旧 package 安装兼容层。历史 `@zeefan1555/fanloop-cli@0.1.1`
只作为既有事实保留，首个 GitHub Release 从 `v0.1.2` 开始。

每个 Release 直接携带 `fanloop-install.sh`、`fanloop-install.js`、`fanloop-launcher.sh`、
`release.json` 和 macOS/Linux、amd64/arm64 四个平台归档。用户只需用 GitHub CLI 完成一次仓库
鉴权，然后执行：

```bash
gh auth login
gh release download -R zeefan1555/fanloop -p fanloop-install.sh -O - | bash
```

安装器从同一个 Release 下载 Manifest、安装逻辑与当前平台归档，继续校验平台归档 SHA-256、
安全解包、staged Doctor、Skill 软链和完整配套内容，通过后才原子切换 `~/.fanloop/current`。
成功输出仍为单行。安装器同时原子写入 `~/.local/bin/fanloop`；若同名文件不是 Fanloop 管理的
launcher，则拒绝覆盖。

持久 shell launcher 只拦截无参数 `fanloop update`，重新运行真实 latest 的同一安装入口并设置
只向前标记；其他命令直接执行当前 Go payload。候选版本低于已安装版本时保留当前 Release，
同版本仍允许修复；失败时不切换当前 Release。绕过 launcher 直接执行 payload 的 `update`
占位继续明确拒绝，不新增第二套下载或更新实现。

正式发布只由 `main` 上手工触发的 GitHub Actions `Release` Workflow 执行，权限收敛为仓库内置
`GITHUB_TOKEN` 的 `contents: write`。Workflow 先创建草稿 Release，对精确 tag 完成真实安装、
版本/commit、Manifest、Doctor、全部 Workflow 初始化和 Skill 投影冒烟，再发布为 latest 并从
真实 latest 重跑；最终冒烟失败时撤销新 latest 并恢复旧 latest。版本号从 GitHub Releases
（包括草稿和预发布）解析，不查询或修改 npm registry。

Node.js 继续承载现有的窄安装 Adapter 和仓库测试，但 npm 不再属于发布或安装协议。
Workflow YAML、Step、Route、Condition、Trace、Card、State、Release Manifest Thrift 和公开 CLI
Thrift 均不改变。本决策取代 ADR-0082 的 GitHub Packages 边界，并修订 ADR-0085 的发布部分；
其余 Fanloop 命名、私有仓库、完整配套 Release、Doctor 与原子切换决策保持有效。
