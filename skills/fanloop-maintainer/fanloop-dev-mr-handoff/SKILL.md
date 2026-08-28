---
name: fanloop-dev-mr-handoff
description: 在本地验证和代码审查通过后，幂等 push Fanloop CLI 分支、创建或更新 GitHub PR，并用 Botmux 发起人工审核话题。
---

# MR Handoff

只在本地验证报告与代码审查报告都覆盖当前 HEAD、工作树无未提交产品变更时执行。不得查询或
等待远端 checks，不 approve、merge 或发布。

## 幂等创建或更新 GitHub PR

1. 读取当前分支与 HEAD，要求 `origin` 精确指向 `zeefan1555/fanloop` GitHub 仓库；未配置远端时
   停止并报告，不自行创建仓库或添加远端。用
   `gh pr list --repo zeefan1555/fanloop --state open --head <branch> --json number,url,baseRefName,headRefName`
   查询。
2. 先用 `git push -u origin HEAD` 幂等推送。唯一命中时用
   `gh pr edit <number> --repo zeefan1555/fanloop --title <title> --body-file <body-file>` 更新；
   无命中时用
   `gh pr create --repo zeefan1555/fanloop --base main --head <branch> --title <title> --body-file <body-file>`
   创建；多命中或目标分支不是 `main` 时停止，不猜测。
3. MR 描述详细写：背景、问题、解决方案、改动点/影响范围、本地测试、ADR impact、风险/非目标。
   默认评审者不了解需求和实现背景；使用通俗、直接、可独立理解的语言，必要术语或缩写首次出现时
   给出简短解释。
4. 创建或更新后用 `gh pr view` 回读真实 PR URL、base、head 与当前 commit，不声称远端 checks 已通过。

## Botmux 人工审核话题

固定审核群：`oc_9f25fc928e2e5a6a602e58fa80b4750a`。真实 mention 张菲帆
`ou_3b0b9cf8364168c5eb999bd6c5a33b95:张菲帆` 作为唯一审核人，并在同一卡片末行 cc
苏文钦 `ou_fdc66f7d48be8c75fa926e0ec27ee809:苏文钦`、吴瑜明
`ou_0cc015d1f9aadf74f752989a0992b869:吴瑜明`。不通知上述三人以外的人员或任何机器人，也不另发 cc 消息。

将下列正文写入 Issue Workspace 的 UTF-8 `handoff-topic.md`；四个业务区块的标题、顺序和数量固定：

```markdown
## MR !<编号> · <标题>

@张菲帆 请审核：
[查看 MR !<编号>](<真实 MR URL>)

**背景**
<内容>

**问题**
<内容>

**解法**
<内容>

**影响范围**
<1-3 条>

cc：@苏文钦 @吴瑜明
```

正文不得加入测试清单、ADR、风险、上线时间、自动审核或自动合并说明；这些事实保留在详细 MR 描述。
首个 cc 名称前必须保留上述全角冒号；`cc@苏文钦` 不符合 Botmux 的内联 mention 边界，会把苏文钦移到
`发送给`尾栏。
多行正文只通过文件发送，不做 JSON stringify，也不把换行转义为字面量 `\n`：

```bash
botmux send --top-level --chat-id oc_9f25fc928e2e5a6a602e58fa80b4750a \
  --mention ou_3b0b9cf8364168c5eb999bd6c5a33b95:张菲帆 \
  --mention ou_fdc66f7d48be8c75fa926e0ec27ee809:苏文钦 \
  --mention ou_0cc015d1f9aadf74f752989a0992b869:吴瑜明 \
  --content-file <issue-workspace>/handoff-topic.md
```

## 完成记录与重试

发送前读取 Issue Workspace 的 `handoff.json`。若它已经是 `phase=complete`，且记录的是相同 MR URL 与 reviewed HEAD，
直接复用，不重复发送。其他旧格式或不完整记录不做兼容迁移；停止并
报告，避免猜测或覆盖历史事实。

`botmux send` 退出码为零后解析 stdout JSON，要求 `success=true`、`messageId` 和 `sessionId` 非空，且
`mentioned` 恰好包含张菲帆、苏文钦和吴瑜明的三个 OpenID。随后用临时文件加 rename 原子写入：

```json
{
  "phase": "complete",
  "requirement": "<absolute Requirement Root>",
  "mrUrl": "<真实 MR URL>",
  "reviewedHead": "<当前 HEAD>",
  "messageId": "<botmux messageId>",
  "sessionId": "<botmux sessionId>"
}
```

只有 MR 存在、Botmux 发送成功且 `handoff.json` 回读与本次 MR URL、reviewed HEAD 和回执一致时，才上报
`merge_request_created`、`merge_request_handed_off`、`handoff_record_written`。失败时上报
`merge_request_handoff_failed`；先回读 MR 和本地记录，只重试未完成动作，不重复创建 MR。
