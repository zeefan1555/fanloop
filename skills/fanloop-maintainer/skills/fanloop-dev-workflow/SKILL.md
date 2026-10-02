---
name: fanloop-dev-workflow
description: Fanloop Maintainer 公共路由入口；Define 确定配置或代码范围，后续步骤按已记录类别执行各自独立绑定的 Skill。
---

# Maintainer 路由

本 Skill 只选择 Route、加载对应 Skill、上报真实结果；具体修改、检查和交付由所选 Route 的业务 Skills 负责。

## 选择通道

1. Define 根据用户目标、允许文件和现有能力分类：已有 Schema 下的 Workflow YAML、Skill 指令/参考材料及必要说明是 `configuration`；实现、可执行脚本、新 CLI 能力、IDL/Schema、Runtime、测试、构建或发布逻辑是 `code`。混合改动归 code，不只按目录或后缀判断。
2. Define 的成功 Result 通过互斥 `configuration_change` / `code_change` 写入 `maintenance_kind`。后续 Step 从当前有效 Outputs 读取这个值，只选择包含对应类别 Condition 的 Route；缺失或非法时 blocked，不自行猜测或改类。
3. 目标、验收、权限、副作用或类别扩大时，保留当前类别，用 `requirements_changed` 回 Define；不在后续 Step 换通道跳验证。配置同一目标所需关联说明同步只补实际清单和修改前摘要，不机械回流。

## 加载所选业务 Skill

CLI 的 `current.prompt` 是此公共入口，`available_routes` 仍会列出两类候选；列表中出现不代表都要执行。

- configuration：根据所选 Route 的业务 Condition，从最新 `current.conditions[]` 读取它的 Prompt 与 required Skills 绝对路径并执行。Define 对应 `configuration_scope_recorded`，Build 对应 `configuration_changes_written`，Certify 对应 `configuration_checked`，Deliver 对应 `live_configuration_synced`。同一个 Agent 连续完成普通配置；只维护 delivery-record.md 阅读文档，不发布飞书阶段报告，不加载 fanloop-stage-report。原始日志保留，按影响自检或独立试用；Deliver 记录授权 Git 交付与公开 CLI 回读，不进入代码通道。
- code：执行所选 code Route 的原 Prompt 及其全部 required Skills，不加载配置业务 Skills。Define、Build、Certify 的成功 Route 还必须执行同一 Status 中对应的 `*_report_published` Condition 及其 `fanloop-stage-report` Skill。代码通道继续遵守原主 Agent监督、验证、认证与交付职责。

只执行当前 Step 需要的业务 Skill，不预读另一条通道或后续 Step 的实现说明。生产配置变更直接维护仓库 skills/<workflow-id>/ 完整包，使用显式本地源校验并在下一次 CLI 调用读取。
YAML 不设置专属人工审批；按实际语义选择严格加载、受影响矩阵及自检/独立试用。涉及 Thrift/Schema 或通用 Runtime 时回 Define 改走 code。

## 推进

读取最新 Status → 执行所选业务 Skill → 按实际事实提交完整 Condition 组合 → 重新读取 Status。每次 Result 携带 Define 确定的类别；错误、范围变化和回流只选择当前 Status 暴露的 Route。

进入 Step 后先执行 required Panorama Skill，完整展示本次 render 内容并保留精确快照回执；同一次进入不重复展示，最终普通回复不重复 Panorama。

把四步作为必要工作上的轻量记录：同一步不重复读取未变化材料，不重新规划或长篇解释阶段切换；只执行当前 Step 必需 Skill，不预读另一通道或后续阶段，不上报尚未发生的事实。

已有 Requirement 按通用 `fanloop-workflow` 使用其固定控制器和数据/Skill Roots，清除外部来源覆盖，由控制器解析自身配置；禁止改写 State 或回退全局 CLI。新 Requirement 使用当前全局控制器。全局 CLI 切换前需要保护已有 Requirement 时使用本次源码工作树中的 `scripts/pin-controller-release.sh`；该固定快照工具需要 Python 3 标准库以共享源目录锁，普通 CLI 无此依赖。
