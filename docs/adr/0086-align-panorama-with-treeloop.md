---
status: accepted
date: 2026-08-30
amends: ADR-0003, ADR-0080, ADR-0081, ADR-0087
---

# 对齐 Treeloop 的单源 Panorama

Panorama 回归为 renderer 对当前 Workflow 事实的紧凑投影：标题与进度、Stage/Step、
状态全景、各阶段 URL Output、当前执行证据和当前行动。Markdown 与 Lark JSON 共用
同一份 `cardContent`，不再展示内部 Step/Condition/Route ID、Prompt 或 Output 注册细节。
URL Output 的人类可读名称继续只取生产 YAML 的 `output.description`。

ADR-0081 要求 Panorama Skill 重新读取审核产物并拼装自包含材料的部分被取代。
审核 Step 仍按自身 Prompt/Skill 组织审核内容；Panorama Skill 只识别已知 Agent 人设，
在 `botmux`、`local_agent`、`aime` 或 `aiden` 中选择唯一通道，原样展示本次
render 的内容或快照，不二次编排、扫描旧快照、fallback 或双发。

ADR-0087 的 Agent 独立批准 Route 保持不变：该 Route 不展示 Panorama。只有选择人工审核
Route 时才执行 `panorama_card_published`，并与人工结论一起上报本次快照路径。

`panorama_card_published` 仍是 Human Step Route 中的原子 Condition，但 Output 改为
`panorama_snapshot_path:path`。只有宿主成功展示或发送后，Skill 才原样返回本次
non-dry-run render 的 Requirement Root 相对 `snapshot_path`；这与 Treeloop 的 Condition
语义一致，也避免本地 Agent 需要在最终回复发出前获取未产生的 Event ID。

本次不修改两套 `workflow.yaml`、Step `id`、`name`、顺序或 `executor`，不修改 Route
组合、YAML/State/Event Schema、`idl/*.thrift`、生成物或公开 CLI Request/Response。不引入
Treeloop 的持久 LaunchMode；Fanloop 保留已有人设选择边界。
