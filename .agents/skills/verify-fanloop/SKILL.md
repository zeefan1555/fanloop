---
name: verify-fanloop
description: 从当前工作树使用隔离二进制、一次性 Requirement 和公开命令验证 Fanloop 的真实 CLI 行为，并保留本地证据。
---

# 验证 Fanloop

Fanloop 是短生命周期 Go CLI。只驱动公开命令，不调用内部 Go 方法充当证据。先读取
[功能地图](features/README.md)，再选择对应 Feature 配方。

## Launch

从待验证工作树运行：

~~~bash
set -euo pipefail
FANLOOP_VERIFY_REPO="$(git rev-parse --show-toplevel)"
cd "$FANLOOP_VERIFY_REPO"
FANLOOP_VERIFY_SESSION="$(mktemp -d "${TMPDIR:-/tmp}/fanloop-verify.XXXXXX")"
FANLOOP_VERIFY_EVIDENCE="$FANLOOP_VERIFY_REPO/tests/requirement_e2e/runs/verify-skill/$(date -u +%Y%m%dT%H%M%SZ)-$$"
FANLOOP_VERIFY_REQUIREMENT="$FANLOOP_VERIFY_SESSION/requirement"
FANLOOP_VERIFY_BIN="$FANLOOP_VERIFY_SESSION/bin/fanloop"
export FANLOOP_DATA_HOME="$FANLOOP_VERIFY_SESSION/data"
unset BOTMUX_CHAT_ID BOTMUX_SESSION_ID
mkdir -p "$FANLOOP_VERIFY_SESSION/bin" "$FANLOOP_VERIFY_REQUIREMENT" "$FANLOOP_VERIFY_EVIDENCE"
git rev-parse HEAD >"$FANLOOP_VERIFY_EVIDENCE/commit.txt"
git status --short >"$FANLOOP_VERIFY_EVIDENCE/git-status-before.txt"
go build -buildvcs=false -o "$FANLOOP_VERIFY_BIN" . \
  >"$FANLOOP_VERIFY_EVIDENCE/build.stdout" \
  2>"$FANLOOP_VERIFY_EVIDENCE/build.stderr"
~~~

没有需要常驻的服务。每个会写状态的配方使用不同 session 与 Requirement。

## Doctor

源码二进制没有安装清单，先用只读 version 检查当前 Bundle：

~~~bash
"$FANLOOP_VERIFY_BIN" version | tee "$FANLOOP_VERIFY_EVIDENCE/version.json"
rg -q '"id": "fanloop-maintainer"' "$FANLOOP_VERIFY_EVIDENCE/version.json"
rg -q '"id": "technical-solution-design"' "$FANLOOP_VERIFY_EVIDENCE/version.json"
~~~

安装与打包行为按 [安装与 Release](features/installation-release.md) 使用隔离技能目录执行
npm run install:local，再要求安装后二进制的 fanloop doctor 健康。

## Drive

- 构造请求前读取叶子命令 --help；每次 Workflow 动作前读取最新 flow status。
- 写操作先 dry-run，再比较 State/Event 哈希；Requirement 命令即使只读也会追加 CLI 日志。
- 本地配方保持 BOTMUX 环境为空。未经单独授权，不执行 trace bind、trace sync、真实机器人或外部写入。
- 不复用历史 Requirement，也不用全局已安装二进制证明当前候选。

## Evidence

保存命令、退出码、stdout/stderr、操作前后 Status、State、Output、Events、Card/Trace 结果、当前 commit
和验证前后 git status。证据位于 tests/requirement_e2e/runs/verify-skill/，不得把 CLI 日志中的秘密复制
到报告。

## Cleanup

只把精确 mktemp session 移到废纸篓，绝不删除证据：

~~~bash
case "$FANLOOP_VERIFY_SESSION" in
  "${TMPDIR:-/tmp}"/fanloop-verify.*)
    if command -v trash >/dev/null; then
      trash "$FANLOOP_VERIFY_SESSION"
    elif command -v gio >/dev/null; then
      gio trash "$FANLOOP_VERIFY_SESSION"
    else
      echo "没有可用的安全废纸篓命令，保留 session: $FANLOOP_VERIFY_SESSION" >&2
      false
    fi
    ;;
  *) echo "拒绝清理不安全路径: $FANLOOP_VERIFY_SESSION" >&2; false ;;
esac
test -s "$FANLOOP_VERIFY_EVIDENCE/commit.txt"
git status --short >"$FANLOOP_VERIFY_EVIDENCE/git-status-after.txt"
~~~

不新增自定义 harness；复用 Go build、公开 Fanloop CLI、rg、shasum、./tests/run-unit 与
./tests/run-e2e。
