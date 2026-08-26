# 先在 Python Driver 旁孵化，再拆出独立仓库

状态：已被 ADR-0027 取代。

原生 Go 与 Python Driver 及其测试对照期间，Fanloop 保留在 `social_skills_repo/fanloop-cli`。行为等价门禁通过后，再把该子目录提取为独立产品仓库；这段临时同仓关系让迁移基线就在旁边，同时不把 Skills 仓库变成长期发布边界。
