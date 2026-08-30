---
name: fanloop-dev-create-verification
description: 为还没有项目级验证能力的仓库创建中文 Verification Skill 与 Feature Map。
---

# 创建验证技能

仅当 `.agents/skills/verify-fanloop/SKILL.md` 不存在或缺少完整入口时使用；已有可用技能时交给
`fanloop-dev-maintain-verification`，不要重建。

## 建立基线

1. 读取公开 README、Help、测试入口、启动脚本和近期用户表面，不从内部实现猜用户操作。
2. 记录可重复的 `Launch → Doctor → Drive → Evidence → Cleanup`，使用隔离数据目录和一次性 Requirement。
3. 在 `.agents/skills/verify-fanloop/features/` 按用户 Feature 建立地图；每页写清入口、前置条件、真实操作、可观察结果、隔离边界和易错点。
4. 所有内容使用中文。外部写入默认不授权；不得回退用户 token、用户身份或历史 Requirement。

## 验证

从当前工作树真实运行至少一个最高层 Feature。保存命令、退出码、stdout/stderr、前后状态和当前
commit；Cleanup 只移除一次性环境，证据必须保留。只有入口可执行、地图可导航且证据完整时，才把
`.agents/skills/verify-fanloop/SKILL.md` 作为 `verification_skill_path`。
