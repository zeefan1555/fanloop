---
name: fanloop-stage-report
description: 将 fanloop-maintainer 的 Define、Build 或 Certify 最终事实发布为飞书报告文档，回读验证后返回当前阶段的 URL Output。
---

# 阶段报告

本 Skill 只在 `define_report_published`、`build_report_published` 或
`certification_report_published` Condition 要求时执行。它不代替阶段业务 Skill，不修改候选，
也不把飞书文档当作本地原始证据。

这些门禁仅属于代码通道；配置通道不加载本 Skill，使用本地 delivery-record.md 链接。

## 输入

先读取最新 `flow status`，按当前 Step 选择唯一报告：

| Step | 标题后缀 | 内容来源 | Condition / Output |
|-|-|-|-|
| Define | Define 报告 | 已回读的 `requirements.md`、需求决定与 receipt | `define_report_published` / `define_report_url` |
| Build | Build 报告 | 已回读的 `verification-report.md` 与候选身份 | `build_report_published` / `build_report_url` |
| Certify | Certify 报告 | 已回读的 `certification-report.md`、独立回执与最终候选身份 | `certification_report_published` / `certification_report_url` |

来源缺失、与当前有效 Output 不一致或仍在变化时保持 blocked，不发布草稿或推测内容。

## 发布

1. 在 Requirement Root 的 `stage-reports/` 写入当前阶段 Markdown，标题为
   `<Requirement 标题>｜<阶段后缀>`。正文只保留可核验事实，至少包含结论、范围或变更、验证/决定、
   证据引用和下一步；不得复制 CLI 密钥、完整未脱敏日志或聊天隐私。
2. 这是对已完成本地材料的原样报告化，使用 `lark-cli docs +create --as user
   --doc-format markdown --title <标题> --content @<绝对 Markdown 路径>` 创建文档。一次调用返回结果未知时
   不自动重试，避免重复文档。
3. 检查创建响应 `ok=true`、文档 URL 非空且没有影响正文完整性的 warning。随后用
   `lark-cli docs +fetch --as user --doc <URL> --doc-format markdown --detail simple` 回读全文，核对标题、
   结论、关键身份和阶段证据均与本地来源一致。创建成功但回读失败时保持 blocked，不上报 URL。
4. 保存创建与回读的原始 stdout、stderr、exit code 和 URL 到 Requirement Root 的
   `stage-reports/<stage>-receipt.json`。不得把这些运行产物写入源码仓库。

## 返回

只有创建与回读全部成功后，返回当前阶段对应的 URL ConditionResult。URL 必须取创建响应原值：

```json
{"condition_id":"define_report_published","output":{"type":"url","value":"https://.../docx/..."}}
```

Build 和 Certify 分别替换为各自 Condition 与 Output。失败或结果不明时报告 blocked；不得使用本地路径、
Trace 文档、CLI 日志文档或未回读的 URL 冒充阶段报告。
