---
status: accepted
date: 2026-09-23
amends: ADR-0062, ADR-0108, ADR-0110, ADR-0111, ADR-0114, ADR-0115, ADR-0116, ADR-0119
---

# 工作流注册中心托管生产 Workflow 与 Skills

工作流注册中心是生产五份 YAML 与完整 Skill 资源的发布真值；代码仓库不再保存生产 `workflows/`。
每个 Workflow 对应一个 Skill ZIP，包含外层 `SKILL.md` 和
`workflows/<id>/{workflow,flow,condition,loop,prompt}.yaml`、`skills/**`。图片、脚本和参考文档必须自包含。
本次保留三套生产Workflow：fanloop-maintainer、pstack-native、remnote-flashcards。
十五份YAML中的十三份字节和全部保留Step id/name/order/executor不变。
Maintainer 的 flow.yaml 配置terminal去掉四个Git事实，prompt.yaml只修正配置回流/交付记录两处文本；
用户在本host问题 `call_SzQpQTOpl6Mv0HUQwewclrVT` index0 明确回复“批准，按这三处修改继续”。
Condition定义、Prompt ID/绑定及代码Route不变。
用户明确要求“技术方案包不要了，给我删了”：退役technical-solution-design全部生产YAML、绑定Skill、入口场景与用户选择，删除其工作流注册中心主包及四个专用资源包。原始范文/图片不在删除范围。
一套Workflow的全部文件对应一个工作流注册中心 Skill；移除resources.json和分包下载/合并机制，不保留未来扩展或fallback。
旧技术图只作为明确测试fixture保留，不是生产库存。

`$FANLOOP_DATA_HOME/workflow-selection.json` 只指定空间与 Workflow/Skill 的选择，不含凭证，也不属于 Release。
默认数据目录为 `~/.fanloop`。配置消费命令启动时通过注册中心 CLI 查找已发布通道的成功发布，再按
明确版本取得 ZIP 元数据。不能直接安装未指定版本的 key：该接口可能返回未发布的 commit。
下载使用标准库 HTTPS；校验 ZIP 路径、重复项、类型和大小后安全解压，再以现有通用加载器校验完整树。
凭证由注册中心 CLI 管理，不转发到 ZIP 下载地址，不输出签名 URL。

经校验的完整选择集替换 `current/workflows`，删除远端已移除的资源。同步和读取使用同一文件锁，
失败不推进 Requirement、不使用旧内容作 fallback。源代码目录不是加载备选。Help、version 和不需要
Workflow 的清理操作不联网。外部平台内容版本不进入 Requirement State、CLI Release 或五文件 Schema。

Dashboard 保留只读文件、拓扑和源码锚点，删除写接口和编辑 UI。它启动时同步，HTTP 读取与配置
替换协调；长运行服务不能持有写锁阻塞其他 CLI。取代 ADR-0108/0111 的 whole-file 保存条款。

构建继续只包含 CLI、统一 `fanloop-workflow` 入口及 Release Manifest。生产安装通过工作流注册中心准备配置；
显式本地配置源只用于隔离测试/验收，不作为生产启动失败后的回退。保留 CLI 精确 commit、Doctor、
安装保护、Requirement 固定控制器与独立认证。

代码 CI 使用测试专用固定行为 fixture 与惰性 Skill stub，不依赖个人凭证或可变远端版本；fixture
不是生产默认源，生产配置更新不要求同步它。保留通用解析、路由、状态、契约、安装和安全覆盖。
生产发布另用公开 CLI 对固定下载集执行全部 Workflow Route Matrix，记录空间/key/version 与文件摘要。
本决策取代 ADR-0119 的配置 Git PR 交付与代码 CI 直接绑定生产文本；生产配置更改由工作流注册中心的 commit、
审核、release 和回读交付，五 YAML 与 Thrift 审核边界继续有效。

代码验证 Feature Map 的唯一位置为 `.cursor/skills/verify-fanloop/features/`，不再是指向生产 Skill 的
符号链接。绑定 Skill 在维护代码时指向该入口，配置资源不复制第二套代码验证地图。

主 Agent 在 host turn `01a0cc59-6761-7290-8b86-5a7551d69e9f` 审核并批准精确契约：
requirements SHA256 `93e99a7409996830b67cb3883b3f04a10ac831c79cb68e9fe2431085d051d860`；
YAML review SHA256 `15206e3bcb4985ac190b8bf34d5551debdf5945a22207016b669ea872dd96dcc`；
receipt `decision-ff597f8d892208da41146b4a4c200d355ca4ea7149f2464d36542a40f69dccd4`。
最终三包契约 SHA256 `f422352a1b04d16a22880d30e5107c8e68ea199b868f63dc8baa26883d337997`，
主Agent回执 `decision-660c88818992c0579d2d849eaf1ccaf0fe422f3c94729a24c8a97016bbfd5076`。
不修改 Thrift IDL、Schema 或 Step 拓扑，生产 YAML 仅含上述明确批准三处例外。
