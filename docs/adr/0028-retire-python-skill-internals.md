# 退役 Python Driver 与旧 Skill 内部契约

四个命令域完成 Go 行为迁移后，删除冻结版 Python Driver、Aime CR Collector 和旧编排 Skill 目录结构测试。Aime CR 在 Fanloop 中只保留进度、结论、证据、阻塞评论和回流等公开生命周期契约；评论抓取与分类由对应 Skill 自己维护。Skill 的文件布局、角色文本和原子 Skill 委派关系也由 Skill 仓库负责，不属于 CLI 运行时契约。Fanloop 继续用真实 CLI Golden 覆盖 Aime CR 生命周期，用配套发布门禁保证 CLI、Skills 与 Workflow 版本一致，不保留 Python 兼容层。
