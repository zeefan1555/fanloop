---
status: accepted
date: 2026-08-19
supersedes_in_part: ADR-0009, ADR-0037, ADR-0038, ADR-0051, ADR-0052, ADR-0054, ADR-0055, ADR-0060
---

# Workflow 源码使用扁平、无业务版本的当前 Bundle

Fanloop 源码与每个新 Release 对每个 Workflow ID 只携带一套不可拆分的五文件 Bundle，
路径固定为 `workflows/<id>/`。删除 Workflow 自身业务 version、版本子目录、
`workflows/defaults.json`、default 标记和 `id@version` selector。Git 保存源码历史，已发布
的完整不可变 Release 保存历史运行环境。

Workflow 的准确内容身份由 `id + normalized digest` 表达；Requirement State/Event、Output
Registry 和 Card Projection 都绑定该引用。整体 CLI/Skills/Workflows/Schema 配套版本仍由
`release_version` 表达。Release Manifest 每个 Workflow ID 只声明一个 `id/path/sha256`
Artifact，安装、更新、Doctor、npm integrity、archive SHA-256 和原子切换继续以完整
Release 为边界。

五份 YAML 的 `schema_version` 只表达文件结构协议，不作为内容修订号。字段结构不变的
Step、Route、Condition、Prompt 或 SkillBinding 编辑不提升 schema，由 Git Diff 与 digest
识别。本次删除 `workflow.yaml.version`，因此 workflow schema 6→7；其他四份 schema
4/2/4/1 不变。当前三套 Bundle 的推进语义和 Step `id/name/顺序/executor` 全部不变。

IDL 直接删除 YAML/common/release/ops 中的 Workflow version/default 字段，保留退休 field
ID 空洞；ops WorkflowRelease 新增 field 4 digest。Release Manifest schema 2，State/Event/
Output Registry/Card Projection schema 11/11/3/4。不保留旧 decoder、双读写、migration、
alias、Feature Flag 或 fallback。运行中旧 Requirement 必须继续由其已安装旧 Release 解释，
或在切换前结束。

本决策取代 ADR-0009 的历史 id@version 共存、defaults 和禁止删除源码条款；修订
ADR-0037/0038/0052 的可版本化 Bundle 描述但保留五文件、digest、原子 Condition、Route
和 release-bound 边界；修订 ADR-0051 的 durable WorkflowRef 与 schema；部分取代 ADR-0054
中 `release.json` Schema Version 保持为 1 和 Workflow 唯一默认版本由通用 Validator 校验的
条款：Manifest schema 改为 2，每个 Workflow ID 只允许一个 Artifact，且不再存在 default 字段
或默认版本校验；修订 ADR-0055 的 E2E 枚举来源；取代 ADR-0060 的新增 14.0.0/defaults
发布方式但保留其当前 Route 行为与 12 Step 契约。ADR-0014、0035、0053、0057 以及完整
Release、安装、更新和 Doctor 的未冲突条款继续有效。
