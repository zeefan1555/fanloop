# Fanloop 机器人验收 Rubric

Stage 3 的协调者冻结 Case 与独立预期；Stage 5 只复用这些事实做真实机器人验收，不现场改题。

## Case 字段

每个 Case 固定 case_id、原始症状或意图、功能地图条目、前置条件、输入、操作步骤、观察项、独立
预期、不覆盖项、candidate_head、Release 和证据目录。两个 Case 使用不同的随机目录、顶层话题和
Requirement，可并行执行，不共享中间状态。

## Rubric（10 分）

| 项目 | 分值 | 得分条件 |
| --- | ---: | --- |
| 机器人身份 | 2 | driver、target、群与本轮 open ID 可回读；没有用户 token 或 --as user |
| Release 来源 | 2 | 安装来自 candidate_head；version 一致；Doctor 健康 |
| Case 保真 | 2 | 输入、步骤、观察项与冻结 Case 一致，并从功能地图走公开入口 |
| 行为证据 | 2 | 独立预期满足；命令、退出码、输出和前后状态可回读 |
| 治理隔离 | 2 | 内层清除 BOTMUX 环境；没有 Card Binding、Trace Integration、远端 Trace Event、用户 Trace/CLI 文档；到 merge_code 前停止 |

只有两个 Case 都为 10/10 且无红线才通过。

## 失败分类

- product_failed：身份、隔离和 Case 有效，但独立预期未满足。
- case_invalid：输入漂移、预期依赖实现或观察项缺失；不评价产品。
- infra_blocked：精确机器人会话、群、网络、接单或回读不可用。
- governance_failed：使用用户身份、Botmux 环境泄漏、生成 Card Binding/Trace Integration/远端 Trace
  或用户文档、执行合码、候选漂移。

确定失败只归属需求、方案、实现、验证技能或功能地图之一；平台故障保持 blocked。
