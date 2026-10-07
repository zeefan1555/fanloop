---
name: workflow-panorama
description: Fixture with the #64 local_agent text. The child renders markdown and reports panorama_presented. No parent/child handoff.
---

# Child reports panorama_presented (this fixture must fail)

## local_agent

无用户可见消息通道的 headless worker 在 `panorama_presented` 上 blocked。report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.

投递只此一条。先做一次非 dry-run render：

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown
```

只保留该次响应的 data.content 和 data.snapshot_path。把该次 data.content 原样作为第一条用户可见消息发送，然后上报 panorama_presented。

不得把 Markdown 写入 artifacts/panorama-* 后声称已展示。进度不得把 content saved artifacts/panorama- 当作投递证明。
