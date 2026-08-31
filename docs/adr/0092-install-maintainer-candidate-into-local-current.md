---
status: accepted
date: 2026-08-31
amends: ADR-0091
---

# 将自迭代候选安装到本机 current

`fanloop-maintainer` 的 `execute_agent_acceptance` 从冻结的 `candidate_head` 工作树执行
`npm run install:local`。安装子进程必须清除 `FANLOOP_DATA_HOME` 和四个 Skill Root 覆盖，使候选切换
真实的 `$HOME/.fanloop/current`；不得使用临时候选二进制、私有 bin 或隔离数据目录代替。

验收继续前必须回读 `current` 链接、`$HOME/.fanloop/current/bin/fanloop version` 与 `doctor`：commit
精确等于 `candidate_head` 且 Doctor 为 healthy。安装成功后保留候选为本机 current，不自动恢复旧版本；
任一安装证据不成立时保持 blocked，不能以旧安装运行机器人。

CI 的 `install-doctor` 和项目级 `verify-fanloop` 仍使用一次性隔离目录，避免通用测试污染开发机。只有
真实机器人验收使用默认本地安装；机器人内层 CLI 继续清除 Botmux 环境并使用刚安装的 current。

本决策不修改五份生产 YAML、Step、Stage、Job、executor、Condition、Flow、Loop、Runtime、IDL 或
State Schema。现有 `agent_acceptance_passed` 已要求安装版本、Doctor、候选 SHA 和黑盒结果全部通过，
因此只收紧该 Step 绑定 Skill 与 Contract，不新增平行门禁。

人工审核记录：用户于 2026-08-31 明确要求“自迭代流程，能把当前分支的 cli 编译安装到本地，而不是
临时 bin”。
