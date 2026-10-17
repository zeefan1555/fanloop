---
status: accepted
date: 2026-10-17
amends: ADR-0132-git-workflow-configuration-center
---

# Fetch Workflow configuration over HTTPS, then SSH once

`flow init`, `fanloop update config`, and `fanloop config drift` resolve the production remote in one place. The default URL is `https://github.com/zeefan1555/second-brain-skills.git`, ref `refs/heads/main`. That HTTPS attempt sets `GIT_TERMINAL_PROMPT=0` and is limited to 15 seconds, so it cannot wait on a credential prompt. If it fails, the same operation prints one notice and retries `git@github.com:zeefan1555/second-brain-skills.git` once. Later Git calls in that operation use the SSH URL without a second notice. A local import path is not rewritten.

`configuration.json` records the URL that succeeded. An existing snapshot stays valid for its commit when the stored URL is the SSH form, or any other form: reuse, doctor, and drift compare the commit and the package bytes, not the repository string. The stored file is not rewritten and the snapshot is not re-cloned only because the URL form differs.

This amends ADR-0132 where the only production URL was the SSH remote fetched through system Git/SSH. Fornax archive metadata, the IDL fetch hint, and Git's own environment URL rewrite are unchanged.
