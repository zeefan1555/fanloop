---
name: flashcard-goal-framing
description: 把材料制卡意图收敛为可检验的长期复习目标、create-new 目标和安全预览边界。用于 material-flashcards 的确认复习目标 Step；不理解材料、不选卡。
---

# 确认复习目标

只确定“这次要长期保留什么能力”和安全交付边界，不读取私密正文来提前制卡。

## 收敛内容

将以下信息写入 Requirement Root 内权限受限、通用命名的目标文件，并回读确认：

- 材料身份、来源与必要时间范围；路径只在文件内保存，输出只引用相对路径和 SHA-256；
- 目标读者、希望长期保留的判断/解释/行动能力、范围、排除项和优先级；
- 隐私级别，以及外部概念核对是禁止、仅允许去标识问题，还是已有明确范围授权；
- 唯一授权 Vault root、规范化 Vault 相对 `target_path` 和 `write_mode=create_new_only`；
- 已明确确认且适合展示本批完整卡片的预览投递位置；
- 恶意并发移动或替换已检查祖先目录属于 v1 非目标，不能声称覆盖该攻击。

材料可以生成 `0..N` 张卡；不得在目标阶段承诺数量或类型配额。

## 路径门禁

拒绝绝对 `target_path`、`..`、规范化后逃逸授权 root 的路径和静态不安全路径。对当时已经存在的每级父路径做 no-follow 检查，符号链接或非目录都拒绝；对最终分量做 no-follow 存在性检查，文件、目录或符号链接都视为冲突。

这里的检查只用于尽早发现问题。真正写入仍必须执行原子 create-exclusive/no-replace；不得把预检查描述成防止最终分量竞态。

目标已存在时只请求新的不存在路径；预览位置不明确或不适合时只请求新的安全位置。缺任一项即保持 `blocked`，不报告目标完成。

## 隐私与返回

Requirement 标题和工件名不得含个人信息。私密正文、私密 URL、消息 ID、投递位置、身份与详细错误不得进入 CLI 参数、Summary、Evidence、Event、Trace 或未确认群聊。Summary 只写进度和非敏感状态；Evidence 只写通用 Requirement-root-relative 路径、`sha256:<64 lowercase hex>`、固定分类与最小摘要。

首次成功上报 `review_goal_framed`；已经接受的目标需要变化时才上报 `review_goal_changed`。
