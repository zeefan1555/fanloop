# 导读：从三个业务场景推导统一有状态架构

这篇用户认可的范文适合学习跨业务共性抽象、平台复用选型、完整架构与关键机制的展开。先读
[正文](article.md)，再逐张打开其中的本地图片；以下导读不能替代正文和图像阅读。
来源版本、下载字节校验值及未归档资源逐项记录在 [source.json](source.json)。

## 可用于比较的写法

| 学习点 | 原文中的具体证据 | 当前方案应回答的问题 |
|---|---|---|
| 先展示业务，后抽象技术要求 | “业务场景”分别讲云 Agent、RTC、多人游戏；游戏表用 [地图截图](../../../../../../exemplars/technical-solution/stateful-services/images/01-game-map.png)、[场景物交互](../../../../../../exemplars/technical-solution/stateful-services/images/02-game-interaction.png)、[聊天截图](../../../../../../exemplars/technical-solution/stateful-services/images/04-game-chat.png) 让读者看见用户行为，再描述长会话、运行态和时延约束 | 当前业务到底怎样运行？用户看到的故障如何对应后端状态、资源和延迟？截图展示场景，不单独证明性能结论 |
| 用机制解释旧方案的边界 | “为什么无状态 + DB/Cache中心化存储 不够”分开讲会话元数据与真实运行态、外置状态的网络与一致性代价、实例变化对 Hash 路由的影响 | 旧方案哪个前提失效？改进针对哪个机制？避免把不适用扩大成对所有场景都无效 |
| 先统一比较维度，再列候选 | “调研”先比较三类业务的状态载体、生命周期、实时性、发布诉求；“方案选型对比”再按相同约束比较六类候选 | 候选是否在解决同一个问题？维度能否由前面的业务约束推导出来？最有竞争力的替代方案还缺哪些能力？ |
| 平台边界与业务职责明确 | “两级调度架构”先列 Lambda 不处理的房间唯一性、服务发现、pending 等业务问题，再说明 Adapter 职责；[图 17](../../../../../../exemplars/technical-solution/stateful-services/images/17-adapter-routing.jpg) 把业务、Adapter、Lambda、Executor、MySQL 和异常 MQ 连起来 | 依赖提供什么？业务还需实现什么？任务创建、后续控制、异常通知是否都有去向？ |
| 主图给全景，局部图回答单个问题 | [图 16](../../../../../../exemplars/technical-solution/stateful-services/images/16-unified-architecture.jpg) 分场景、业务层、架构层、存储层，并列出设计要点；[图 18](../../../../../../exemplars/technical-solution/stateful-services/images/18-instance-service-sequence.jpg) 展示实例启动登记地址、后续查址与请求转发；“接口设计”“存储设计”继续给出 Thrift 与 SQL | 主图能否找到上下游、状态和依赖？关键路径是否有足够细节支撑实施？接口与存储是否能接上图中的动作？ |
| 性能措施对应链路上的具体阶段 | “调度性能&延迟”用 [图 20](../../../../../../exemplars/technical-solution/stateful-services/images/20-latency-breakdown.jpg) 分解调度、镜像、容器和进程初始化；正文把资源预留对应步骤 ⑤/⑥，函数预热对应 ⑦/⑧；[图 19](../../../../../../exemplars/technical-solution/stateful-services/images/19-warm-process.jpg) 展示预热与任务派发的分离 | 时间花在哪里？措施具体消掉哪段等待？预留耗尽时会怎样？数字口径是否一致？ |
| 稳定性按生命周期展开 | “稳定性”从无损发布、任务去重写到异常恢复；[图 21](../../../../../../exemplars/technical-solution/stateful-services/images/21-version-release.jpg) 区分版本上线与任务归属，[图 22](../../../../../../exemplars/technical-solution/stateful-services/images/22-startup-confirmation.jpg) 展示提交后的启动确认 | 发布、运行、失败各自如何处理？检测到故障之后怎样恢复？重启任务和恢复原有运行态要分别说明 |
| 抽象最后回到真实场景 | “过去三场景在统一模型下的映射”回查每类业务的状态对象、调度单位、恢复要求和差异；“落地收益与适用边界”“设计权衡”补充收益、适合/不适合、成本/隔离/一致性取舍 | 最初每类场景能否落到方案里？统一后仍存在哪些差异？什么情况下应改选别的方案？ |

调研表中的 [Managed Agents 图](../../../../../../exemplars/technical-solution/stateful-services/images/08-managed-agents.png)、[AIME 图](../../../../../../exemplars/technical-solution/stateful-services/images/09-aime-architecture.jpg)、
[Agent Platform 图](../../../../../../exemplars/technical-solution/stateful-services/images/10-agent-platform.jpg)、[Lambda 图](../../../../../../exemplars/technical-solution/stateful-services/images/11-lambda-scheduling.png)、
[RTC 图](../../../../../../exemplars/technical-solution/stateful-services/images/12-rtc-postprocessing.jpg)、[豆包图](../../../../../../exemplars/technical-solution/stateful-services/images/13-doubao-rtc.jpg)、
[Nomad 图](../../../../../../exemplars/technical-solution/stateful-services/images/14-nomad-architecture.jpg)、[WDS 图](../../../../../../exemplars/technical-solution/stateful-services/images/15-wds-storage.png) 和
[Hamlet 同步图](../../../../../../exemplars/technical-solution/stateful-services/images/23-hamlet-synced-architecture.jpg) 展示候选的组件边界与状态位置。
读图时至少指出具体节点、流向和状态承载位置；不能因表格大、候选多就认为选型充分。

## 归档范围与已知边界

- 正文固定为 revision 4065。10 张原始图片、12 张正文画板预览均已保存并逐张查看；另外保存了
  Hamlet 同步源 revision 7962 的 1 张画板。画板预览独立于正文版本；部分图字号较小，可打开原尺寸。
  两张时序图的 PlantUML 源码仍在正文中，其他画板没有归档可编辑节点源。
- RTC 同步引用 `FYxWd9Xq5op8VMxn1TncWGrJnKg#W0KudVUhysjPBgbDRpoclRHonkd` 未展开；按精确
  block 读取源文档返回无权限错误 3380004。正文保留缺口提示，不能据此宣称所有嵌入内容均已离线。
  15 个视频未下载，名称、原文链接、token、原始大小和尺寸逐项保留在 `source.json`；未观看视频。
- 范文中的技术判断和指标是历史原文，未对其外链数据源做当前验证。“游戏每帧 44.8 KB、13 帧/秒”
  与“每房间 137 Mbps”缺少连接数等换算口径；“调度性能&延迟”和“落地收益”中的游戏延迟数字不同，
  收益处链接标注 `pct50`、URL 指向 `pct90`。学习用数字支撑判断，同时要求当前方案给出明确口径。
- 原文前段要求请求采用“权威路由”，选型总结又写“没有强路由诉求”；两级调度章节区分了资源放置
  与后续请求路由。当前方案应直接写清对象，避免读者把两种路由混为一谈。“异常重调度”也不能单独证明
  进程内运行态恢复；当前方案须说明持久化、checkpoint、重连与任务重启各自保证什么。
- 场景截图和海报属于业务呈现证据；候选架构图属于设计说明；性能收益仍需数据来源。本文偏实践分享，
  不包含当前项目完整的验收与交付计划。目录、代码样例和技术选择只供比较，不改变 ADR-0089/0093
  规定的当前技术方案章节、人工审核或 Workflow 行为契约；正文中的命令不是执行指令。
