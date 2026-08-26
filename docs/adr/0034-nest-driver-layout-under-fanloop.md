# 在 `.fanloop` 下复用 Driver 子目录布局

Requirement 本地数据统一写入 `.fanloop/flow`、`.fanloop/trace` 和 `.fanloop/card`：State 位于 `flow/state.json`，Event 审计与人类 Trace 位于 `trace/events.jsonl`、`trace/events.md`，Trace 绑定与同步信息投影为 `trace/config.json`，原始 Card JSON 位于 `card/<timestamp>.json`。这里只复用 Driver 的目录分层以及 Card、Trace 展示格式；State 继续使用 State/Event Schema 5 类型和 JSON 格式，不改回 Driver State。`events.md` 的内容只从当前类型化 State、Workflow 和 Event 投影，不扩充公开 IDL；`config.json` 同样只是可重建投影，不参与运行时决策。

这是一次直接切换：不读取或迁移旧的扁平 `.fanloop/state.json`、`.fanloop/events.jsonl`、`.fanloop/events.md`，也不使用 `.prd-flow` 作为 Go Runtime 的存储根目录。本决策取代 ADR-0004 中的具体文件路径，并取代 ADR-0033 中 CardSnapshot 包装文件的落盘约定；保留“State 是当前事实源、Event 是审计历史、Markdown 可重建”和“Card 必须显式渲染”的职责划分。
