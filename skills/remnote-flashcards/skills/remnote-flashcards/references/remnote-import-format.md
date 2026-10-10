# RemNote CLI 卡片语法

写入通过同 Workflow 的 [remnote-cli](../../remnote-cli/SKILL.md)。链路是 CLI 与插件 WebSocket。闪卡组合卡的标题卡引用不走 `create --content-file`。下列格式只表达可见结构和已有卡片识别。

combo rem-refs use flashcards apply TextSegments {"remId":"..."} or object-set-text RichText {"i":"q","_id":"..."}; do not use create --content-file with [[id:]].

2026-10-07 失败例：`create --content-file` 中的 `[[id:gTpsZTEYvBxB4S1PM]]`（分批加仓）把 T 正文写成 `[[DCF]]0`。不得再把组合卡 rem-ref 放进这个文件。

## 新建或改写卡片：统一使用多行格式

普通卡的卡型、标题、别名、STAR 原文和箭头以[学习卡契约](learning-card-contract.md)为准。原文导入／摘录仍按对应模式保真。一批完整例子见 [市盈率干跑示例](example-pe-valuation.md)。

原子术语卡：

```markdown
- PE ===
    - 正式定义：一行定义或公式。术语只放这里。
    - 白话类比：一句给大约 8–10 岁小朋友的日常场景，这一行没有未解释的术语。
    - 例子：可选的一句口算；没有就删掉。
```

组合卡的标题、别名和标题卡引用按学习卡契约。语法骨架如下。段间看似空白的 Rem 各包含一个 `U+200B`。原子术语卡不插入这些空行。完整正文见干跑示例。下面骨架把引用写成 `{ref:名称}`：它是显示示意，写入计划时 `question` 与答案都换成 `{"remId":"<该标题卡 Rem ID>"}` 段，例如 `"question":["刚学完 ",{"remId":"<PE_ID>"},"：……？ ==="]`。计划里不得出现字面 `[[`、`]]`。

```markdown
- 刚学完 {ref:PE}：一个只问一件事的问题 ===
    - S — Situation：背景
    - 场景段落
    - ​
    - T — Task：任务／目标
    - 你要学会：……
    - ​
    - A — Action：行动
    - 小数目演算
    - ​
    - R — Result：结果
    - 一句口诀
```

下面「已有卡片的语法参考」里的 `>>`、`<<` 是别的卡型的分隔符，不写进多行学习卡。学习卡的箭头禁令在契约。

## 已有卡片的语法参考

```markdown
- 正向问题 >> 答案
- 答案 << 反向问题
- 术语 <> 定义
- 概念 :: 定义
- 属性 ;; 值
- 包含{{隐藏答案}}的文本
- 有序答案 >>1.
    - 第一步
    - 第二步
- 选择题 >>A)
    - 正确答案
    - 干扰项
```

这些格式用于检查时识别已有卡片。新建或改写普通学习卡使用上方的多行格式，并遵循[学习卡契约](learning-card-contract.md)。

`>>A)` 卡片的第一个子项是正确答案。
