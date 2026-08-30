# 安装、Release 与 Skills

用户安装一个协调一致的 Fanloop Release，并期望二进制、Workflow digest、Skills、链接和 State
Schema 来自同一 commit。

## 用户入口

- 在隔离技能目录和 FANLOOP_DATA_HOME 下运行 npm run install:local
- 使用安装后二进制运行 fanloop version 与 fanloop doctor

## 真实驱动

1. 将 FANLOOP_CODEX_SKILLS_ROOT、FANLOOP_AGENT_SKILLS_ROOT、FANLOOP_TRAE_SKILLS_ROOT、
   FANLOOP_CLAUDE_SKILLS_ROOT 全部指向一次性 session；不要污染用户全局链接。
2. 运行 npm run install:local，保存输出和退出码，要求 current/bin/fanloop 与 release.json 存在。
3. 运行安装后二进制 version，要求 commit 等于 git rev-parse HEAD、两个生产 Workflow 都存在，并且
   fanloop-dev-create-verification、fanloop-dev-maintain-verification、fanloop-dev-ci-gate 和
   fanloop-dev-merge-code 出现在 Skills。
4. 运行 doctor，要求 Manifest、checksum、Skills、Skill links、Workflows 与 version drift 均通过。
5. 保存 release.json 和所有链接目标，要求全部位于一次性 session。

dist/ 是工作树共享构建目录，不并发运行多个 install:local。源码 build 的 version 不能替代安装后
Doctor。Cleanup 移除隔离安装和链接，但保留证据。
