# Fanloop Feature Map

本地图只导航用户可观察行为，不维护全仓函数索引。收到模糊反馈时先匹配下表，再读取对应叶子命令
`--help` 和当前 Requirement 的 `flow status`；验证只走现有公开入口。

| Feature | 症状 / 意图 | 公开命令 | 稳定代码 Seam | 最小真实验证 | 证据 |
| --- | --- | --- | --- | --- | --- |
| Requirement / Flow | “新需求起不来”“当前步骤不对”“上报后没推进或错误回流” | `fanloop flow init/status/report progress/report result` | `cmd/flow.go`、`internal/flow/`、`internal/workflowview/`、`workflows/<id>/{workflow,condition,flow,loop,prompt}.yaml` | 在全新绝对目录先观察 `NOT_INITIALIZED`，再依次执行 `init --dry-run`、`init`、`status`；按 Status 中当前 Step 和 Route 构造一次 `report ... --dry-run`，获授权后才真实上报 | 命令、退出码、stdout/stderr、`.fanloop/flow/state.json`、`.fanloop/trace/events.jsonl` 的前后快照 |
| Output / State | “Output 丢了”“回流后旧结果还在”“STATE_CORRUPT / STATE_SCHEMA_UNSUPPORTED” | `fanloop flow status --root <root>`、`fanloop doctor --root <root>` | `internal/store/`、`internal/state/`、`idl/storage.thrift`、`idl/flow.thrift` | 用公开 Flow Case 写入 Output，再按生产 Route 前进或回流；通过 `flow status` 比较 `outputs` 与 `invalidated_outputs`，不手改 State | CLI Response、`.fanloop/output/state.json`、State/Event 快照和错误码 |
| Trace | “Trace 没绑定”“本地投影不更新”“同步飞书失败” | `fanloop trace bind/status/render/sync` | `cmd/trace.go`、`internal/trace/`、`internal/store/trace_projection.go`、`internal/traceconfig/registry.yaml` | 先 `trace status`；本地问题执行 `trace render --dry-run`。只有外部写入已获授权时才执行真实 `bind/sync`，否则止于 dry-run | Trace Response、`.fanloop/trace/{config.json,events.jsonl,events.md}`、各 target 状态与上游错误 |
| Card | “卡片白屏/字段不对”“全景和当前状态不一致” | `fanloop card render --view current\|panorama --format markdown\|lark-json` | `cmd/card.go`、`internal/card/`、`internal/workflowview/` | 对同一 Requirement 分别执行 current/panorama 的 `--dry-run`；与紧邻的 `flow status` 对照当前 Step、Outputs、Evidence 和 Human Step 提示 | 返回的 `data.content`、退出码；非 dry-run 时再保存 `.fanloop/card/` snapshot path |
| Install / Update / Doctor | “装到旧版本”“update 不生效”“doctor 不健康” | `fanloop version`、`fanloop doctor`、npm launcher 下的 `fanloop update` | `scripts/install.js`、`internal/release/install/`、`internal/ops/`、`cmd/root.go` | 源码候选必须从同一最终工作树运行 `npm run install:local`，随后 `fanloop version`、`fanloop doctor`；不得用已发布版本替代 | `release_version`、`commit`、State Schema、Skill/Workflow 列表、Doctor checks、安装前后 `current` 指向 |
| Release / Skills | “Skill 找不到”“Release 组件或摘要不一致”“Agent 仍执行旧 Prompt” | `fanloop version`、`fanloop doctor` | `.goreleaser.yml`、`scripts/build-release.sh`、`internal/release/`、`entrypoints/`、`skills/` | 运行已确认的聚焦 Release/Skill Contract；真实 Agent 验收前重新本地安装最终 HEAD，并从 Status 回读 Release-bound 绝对 Skill path | Contract 输出、`release.json`、Skill 目录摘要、`version` 中的 commit、真实 Agent Requirement/线程证据 |

## Case 选择规则

1. 一个反馈只选能覆盖它的最高层现有 Seam；需要多个 Feature 时拆成 1–3 个独立 Case。
2. `baseline` 与 `candidate` 必须复用相同输入、前置条件和观察项；只允许 commit、Release 和实际结果变化。
3. 先保存退出码、stdout/stderr 与状态/文件差异，再解释原因；“测试通过”不是单独证据。
4. Skill、Prompt 或用户可观察 Agent 行为变化还要执行真实机器人 E2E；纯说明文档可记录 `N/A` 和理由。
