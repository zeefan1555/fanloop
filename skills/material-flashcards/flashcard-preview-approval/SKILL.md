---
name: flashcard-preview-approval
description: 校验质量报告与草稿绑定，在已确认私密位置展示精确预览和安全 Panorama，并记录全新明确的人类决定。用于 material-flashcards 的预览并确认卡片 Human Step。
---

# 预览与人工批准

本 Skill 不生成、修订或自行批准卡片。只有完整性、目标路径、投递位置和隐私门禁全部通过，才能展示预览并请求决定。

## 展示前完整性

从 Current Evidence 取得 `quality_review` 报告相对路径和 SHA-256，先复算原始字节摘要，再读取。核对报告中的草稿路径等于有效 `card_draft_path`，并复算草稿摘要与报告值一致。任何 locator、报告或草稿绑定不一致时：

- 写入权限受限的本地诊断；
- 只上报 `preview_draft_changed`；
- 不展示预览、Panorama，也不请求批准。

重新检查授权 Vault root 和规范化 Vault 相对目标：拒绝绝对路径、`..`、静态 root escape、父目录符号链接/非目录和已存在的最终分量。确认当前投递位置与目标文件中明确批准的位置一致且适合本批内容。失败时保持 `blocked`；不得泄露预览来解释失败。

## 精确预览与记录

只在已确认位置以文件方式真实展示自包含预览，包括精确候选卡、目标、必要来源/时间与归因、排除或延后项、质量结论、Vault 相对目标、草稿相对路径和 SHA。预览正文不得进入 CLI 参数、stdout、Event、Trace、Panorama 或未确认群聊。

本地 `preview_record` 必须绑定：

- `quality_review_path` / `quality_review_sha256`；
- `draft_path` / `draft_sha256`；
- `target_path`；
- 预览文件身份、已确认投递位置和真实展示事实。

上报 `card_preview_published` 的严格对象前，确认字段与 Workflow Condition 完全一致，无缺失、无额外字段。

## Panorama 与人类决定

精确预览成功后，先清空 Current Evidence，并把 Summary 限定为 progress、card_count、已确认可展示的 Vault 相对 `target_path` 和 review_status，再调用 renderer-owned Panorama Skill。Panorama 只含流程信息，不含正文或个人信息。

随后等待**预览之后**的全新明确决定。只有已认证真实人类可以批准；`sender_type=human`，Bot、Agent、历史回复、模糊肯定和本 Skill 自身都不得自行批准。批准记录必须在本地保存未经改写的原始回复、不同于预览消息的消息身份、时间关系和 `quality_review → draft → preview → target` 绑定；CLI 只收到记录相对路径和 SHA。

批准时同时上报 `card_preview_published`、`panorama_card_published` 和 `card_preview_approved`。本地 `approval_record` 保存上述批准记录。修改意见写入本地反馈，按最早受影响层回流；无可定位意见的明确拒绝回 `draft_cards`，信息仍不足则目标 Step `blocked`。任何回流后旧批准都失效，持久化前必须取得新的明确人类批准。
