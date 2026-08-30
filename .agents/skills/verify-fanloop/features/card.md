# Card

用户从已提交 Requirement 事实渲染当前视图和全景，不让 UI Agent 重建 Workflow 状态。

## 用户入口

- fanloop card render --view current --format markdown
- fanloop card render --view panorama --format markdown
- fanloop card render --view current --format lark-json

## 真实驱动

1. 初始化全新 Requirement，并保存紧邻的 flow status。
2. current Markdown dry-run 必须包含标题、当前 Stage/Job/Step。
3. panorama Markdown dry-run 必须包含五个 Stage、十二个 Job 和十六个 Step；多 Job Stage 必须保留
   Job 名称与边界，并包含“机器人端到端验收”和“自动合码”。
4. current lark-json dry-run 必须是有效 JSON，并表达同一当前 Step。
5. 记录 card 目录，证明 dry-run 不新增快照；再执行一次非 dry-run，保存返回的 snapshot_path 与文件。

Card 来自已绑定 Workflow；修改 YAML 后必须使用全新 Requirement。机器人验收的内层隔离 CLI 不应
捕获 Botmux Card Binding，存在 .fanloop/card/config.json 即治理失败。
