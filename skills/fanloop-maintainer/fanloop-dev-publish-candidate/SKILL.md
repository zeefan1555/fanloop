---
name: fanloop-dev-publish-candidate
description: 把冻结 candidate_head 幂等发布为唯一 GitHub PR，不执行合并。
---

# 发布候选 PR

要求 origin 精确为 zeefan1555/fanloop、当前分支不是 main、工作树干净，且 HEAD 等于
candidate_head。先查询该 head 分支全部 PR：多个权威 PR、错误 base、draft 或 head OID 漂移均
停止。无 PR 时创建 base=main 的 PR；唯一 open PR 时只更新标题和正文。

推送后回读 baseRefName=main、headRefOid=candidate_head、state=OPEN、isDraft=false，一致才输出
pull_request_url。不得创建第二个 PR、合并、rebase、直接 push main 或发送 MR 交接。
