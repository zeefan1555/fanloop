# 卡片渲染与发送保持分离

Fanloop 保持现有集成边界：本地提交后可以通过 `trace sync` 触发 Trace 和 Registry 投影；状态变化响应根据 Workflow 返回独立 Card 指令，`card render` 只生成确定性卡片 JSON，发送仍由调用方负责。在出现明确的重试需求前，不增加 `card send` 或投递状态，避免在流程运行时中再建设一套远端投递系统。
