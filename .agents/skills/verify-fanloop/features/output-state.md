# Output 与 State

用户依赖类型化 Output 随生产 Step 持久化，并在显式 Loop 回流时失效目标及下游结果。

## 用户入口

- fanloop flow status --root ROOT
- fanloop doctor --root ROOT 仅用于隔离安装后的 Release
- ./tests/run-e2e 用于完整 Route Matrix 与失效覆盖

## 真实驱动

1. 完成 requirement-flow.md 的 init 与第一次真实结果。
2. 从 status 读取 issue_workspace_path、类型 path、值 issue-workspace 和 producer bootstrap_techdesign。
3. 保存 flow/state.json、output/state.json 和 trace/events.jsonl；与公开 Status 对照。
4. 沿 Status 返回的正常 Route 前进后再次读取，要求上游 Output 仍存在。
5. 需要证明回流失效时运行 ./tests/run-e2e，并保存 fanloop-maintainer Route Matrix 报告中的
   invalidated_outputs；不得伪造当前 Step 不提供的 Loop。

Output State 在第一次注册前可以不存在。测试报告只能补充，不能替代公开 CLI 的前后证明。
