---
status: accepted
date: 2026-10-17
amends: ADR-0132-git-workflow-configuration-center
---

# Fetch Workflow configuration over HTTPS, then SSH once

`flow init`, `fanloop update config`, and `fanloop config drift` resolve the production remote in one place. The default URL is `https://github.com/zeefan1555/second-brain-skills.git`, ref `refs/heads/main`. Git sets `GIT_TERMINAL_PROMPT=0`, so it cannot wait on a credential prompt. Remote probes have a 15-second budget per attempt. Each fetch and checkout attempt has a separate two-minute budget, matching the existing release client's request budget. If HTTPS fails while the parent context is still live, the operation prints one notice and retries `git@github.com:zeefan1555/second-brain-skills.git` once. Later Git calls in that operation use the SSH URL without a second notice. A canceled parent does not start a retry. A local import path is not rewritten.

The October 26, 2026 amendment replaces the shared 15-second transfer cutoff. Each checkout attempt owns a new private staging repository. Only a complete checkout is renamed to the fixed checkout path. Failed attempts never share a Git directory with the retry. Cleanup removes only directories created for this operation, including their interrupted Git locks. It never searches for or deletes locks in user repositories.

Git failures report only timeout, canceled, authentication, lock conflict, network, or other categories. HTTPS and SSH causes remain separate, and callers can identify both `ErrFetch` and the underlying context cancellation or deadline with `errors.Is`. Raw Git stderr, credential-bearing URLs, and environment values are not printed. `CommandContext` retains its direct-child cancellation behavior. A bounded `WaitDelay` closes orphan-held pipes without killing unrelated processes or Git's descendants. Failures leave `config/current` unchanged.

`configuration.json` records the URL that succeeded. An existing snapshot stays valid for its commit when the stored URL is the SSH form, or any other form: reuse, doctor, and drift compare the commit and the package bytes, not the repository string. The stored file is not rewritten and the snapshot is not re-cloned only because the URL form differs.

This amends ADR-0132 where the only production URL was the SSH remote fetched through system Git/SSH. Fornax archive metadata, the IDL fetch hint, and Git's own environment URL rewrite are unchanged.
