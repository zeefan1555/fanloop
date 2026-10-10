---
name: workflow-panorama
description: Fixture copied from the 2026-10-07 FOMO batch, where local_agent saved a panorama file and reported panorama_presented.
---

# File-save delivery (this fixture must fail)

### `local_agent`

```bash
fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown
```

Save data.content to artifacts/panorama-resolve_home_tag.md and report panorama_presented.
panorama rendered local_agent markdown; snapshot .fanloop/card/20261007T085310.491110+0800.json; content saved artifacts/panorama-resolve_home_tag.md
