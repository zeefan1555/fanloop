---
status: accepted
date: 2026-10-21
amends: ADR-0136-distribute-cli-from-github-releases
---

# 持久化 npm 启动器 PATH，并限制发布下载重试

本决定只修订 ADR-0136-distribute-cli-from-github-releases 的 npm PATH 配置和下载策略。继续使用薄 npm 包装、现有 Node fetch、sha256 与完整发布树校验、公共 Skill 链接和原子 current 激活。ADR-0140 的安装后配置同步、ADR-0143 的 Condition 所有权和 ADR-0145 的固定快照显式重绑保持有效。不增加 IDL、配置开关、依赖、Fornax HTTP 客户端或旧 Root 迁移。

npm postinstall 在校验并激活 CLI 后，把一个 `# >>> fanloop PATH >>>` / `# <<< fanloop PATH <<<` 区块写入用户 shell 启动文件。zsh 使用 ZDOTDIR，未设置时使用 HOME，写 `.zshrc` 和 `.zprofile`；bash 写 `.bashrc`，以及 `.bash_profile`、`.bash_login`、`.profile` 中优先存在的一个，均不存在时新建 `.bash_profile`；sh 写 `.profile`。区块只更新自己拥有的内容，保留其他文件字节；异常 marker 拒绝修改。路径使用安全单引号引用，含空格和单引号仍可用。npm 提供 prefix 时使用绝对 `prefix/bin`，否则使用绝对 `dataHome/current/bin`。当前终端的手动 export 提示仍保留。

不支持的 shell、相对或包含控制字符/PATH 分隔符的路径、profile 读取或写入失败必须使 postinstall 明确失败，不打印安装成功。PATH 操作发生在 CLI 激活后，因此 profile 失败可能留下已激活 CLI 和部分已写 profile；不能宣称整次安装原子回滚。下载、校验或激活之前的失败仍保留旧 current。

每个发布文件下载最多四次，每次独立 30 秒截止，覆盖响应头、最多五跳重定向和完整正文；前三次失败后的等待为 100、200、400ms。只重试明确 TimeoutError/Undici timeout，或收到 2xx 后正文阶段的 UND_ERR_SOCKET、UND_ERR_RES_CONTENT_LENGTH_MISMATCH。HTTP 错误、证书错误、未知错误、不安全地址、过大响应、校验与解包失败立即拒绝。每次请求前校验地址：HTTPS，或本地 loopback 测试 HTTP，且无 userinfo。使用手动重定向，每跳沿用同一次截止。终端只打印安全错误分类，不包含底层 URL、userinfo、query 或 token。

发布的 fanloop-workflow 入口为未指定目录的新需求选 `~/fanloop/workspace/<workflow-id>/<requirement-root>`；传入 CLI 前仍解析为绝对 Root。已有 Requirement 使用原绝对 Root 继续，不迁移目录。

契约位于 `packaging/npm/install.test.js`，实际安装、新 shell、固定截止和显式 Root 由 installation-health、workflow-selection Feature Map 的公开入口验收。
