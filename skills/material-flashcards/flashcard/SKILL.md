---
name: flashcard
description: 在当前 Obsidian vault 里创建或修复 Decks 卡片。适用于 Decks、二级标题卡片、卡片标签、把笔记整理成可复习卡片。
---

# Flashcard Skill

这个 Skill 只服务于当前 Obsidian vault 里的 Decks 插件。

默认产物是 Decks 可解析的 Markdown 文档：每一个 `##` 小节就是一张卡片。不要把这个 Skill 扩展成通用学习法、复习算法或长文整理 Skill。

## 目标格式

使用 Decks 的“标题-段落”格式：

```markdown
---
tags:
  - decks/<领域>/<主题>
---

# 分组或上下文

## 卡片正面

这里写卡片背面内容。
```

基本规则：

- `tags` 必须包含 Decks 卡片标签。
- Decks 标签按领域和模块分层，不要只写裸 deck name。
- 推荐格式是 `decks/<领域>/<主题>`，例如 `decks/career/captain-planning`。
- Decks 标签最多三级；需要表达模块时，把主题和模块合并到第三层，例如 `captain-planning`。
- 每个聚合文档通常只放一个最精确的 Decks 标签；不要同时放父标签和子标签。
- `#` 是分组或上下文，不是卡片正面；它可以让复习时知道卡片属于哪个层级。
- `##` 是卡片正面。每篇原文、每个明确问题、每个明确概念，通常对应一个 `##`。
- `##` 下方内容是卡片背面。
- 不要使用 `#decks/title`，除非用户明确要求使用这个标签。

## 标签规则

Decks 标签要表达“领域 / 主题”，而不是只表达主题名。

推荐写法：

```yaml
---
tags:
  - decks/career/captain-planning
---
```

命名建议：

- 第一层固定用 `decks`，让 Decks 插件识别。
- 第二层写领域，例如 `career`、`tech`、`promotion`、`english`。
- 第三层写主题、来源或主题-模块合并名，例如 `captain-planning`、`backend`、`interview`。
- 不要写第四层；如果需要模块维度，用短横线并入第三层。
- 不要为了表达父子关系同时写两个标签；Decks 的嵌套标签树会让 `decks/career/captain-planning` 自动出现在 `decks/career/captain` 下面。
- 如果用户已经给了明确标签，优先遵守用户标签。
- 如果用户只给了来源名，要补上领域层级，不要直接写成 `decks/<来源名>`。

## 解析边界规则

Decks 识别 Markdown 标题的规则是：

```text
^(#{1,6})\s+
```

当 Decks 按 H2 解析卡片时，后续任何 Markdown 标题都可能关闭当前卡片正文。因此，在 `##` 卡片正文里继续写 `###` 或 `####`，会导致复习界面看起来像“背面内容被截断”。

必须这样处理：

- 真正的卡片标题保留为 Markdown `##`。
- 分组或上下文保留为 Markdown `#`。
- 卡片背面里的文章内部标题，全部改成 HTML 标题，不要用 Markdown 标题。
- HTML 标题保留原文视觉层级：
  - 原文 `## 标题` -> `<h2>标题</h2>`
  - 原文 `### 标题` -> `<h3>标题</h3>`
  - 原文 `#### 标题` -> `<h4>标题</h4>`
- HTML 标题不是以 `#` 开头，所以会留在卡片背面，不会把卡片切断。
- HTML 标题后面不要额外空一行；下一段正文直接跟在 `</h2>`、`</h3>`、`</h4>` 下一行，复习界面更紧凑。

示例：

```markdown
# 职业阶段

## 职业规划之-新手期

正文第一段。

<h2>1. 如何找到优秀的团队？</h2>
这一节仍然属于同一张卡的背面。

<h3>补充判断</h3>
这也仍然属于同一张卡。
```

## 批量导入资料

导入一个仓库或一批 Markdown 文章时：

- 优先按照源 README 或目录结构生成少量聚合 Decks 文档。
- 不要默认生成 100+ 个单独笔记文件，除非用户明确要求一篇文章一个文件。
- README 的一级业务板块通常可以变成一个聚合文档。
- README 的子板块通常可以变成 Markdown `#` 分组标题。
- 每篇原文通常变成一个 Markdown `##` 卡片。
- 如果原文第一个 `# 标题` 和 `##` 卡片正面重复，要去掉这个原文标题。
- 原文内部 Markdown 标题必须按上面的解析边界规则改成 HTML 标题。
- 除非用户要求总结，否则保留原文正文内容。

## 迁移旧 root Decks 目录

旧的根目录 `Decks/` 只作为迁移来源，最终卡片要回到对应 PARA 路径。

迁移规则：

- `Decks/<PARA 路径>/<文件名>.md` 的目标是 `<PARA 路径>/<文件名>.md`。
- 文件名里的 ` (闪卡)` 是旧迁移后缀；目标文件名通常要去掉这个后缀。
- 迁移时保留 `%%dk:h:...%%` 锚点；这些锚点比文件路径更能保留 Decks 的复习身份。
- 如果源文件有 `decks-id`，目标文件可以保留；旧复习调度字段不要迁移。
- 旧的 `decks/title`、`decks/review/migration` 标签要替换成精确的 `decks/<领域>/<主题>` 标签。
- 同一个目标文件只保留一个最精确的 frontmatter Decks 标签；旧标题行里的 inline `#decks/...` 能删就删。
- 如果目标文件已经覆盖源文件全部 `%%dk:h:...%%` 锚点，不要重复追加卡片；备份后删除 root `Decks/` 下的源文件。
- 没有 `%%dk:h:...%%` 的 root Decks 文件，通常是旧索引、草稿或迁移辅助文件；只有确认它包含目标文件没有覆盖的真实知识时才迁移，否则删除。
- 删除 root `Decks/` 前先做 tar 备份，再用锚点覆盖率和卡片数校验；不要手改 Decks 插件数据库。

## 卡片粒度

默认粒度：

- 一篇原文 = 一张卡片。
- 一个明确问题或概念 = 一张卡片。
- 不要因为文章内部有很多小标题，就自动拆成很多张卡。
- 只有用户明确要求更细粒度时，才继续拆分。

目标是让结构可复习，不是追求卡片数量最大化。

## 概念学习卡

当用户要求术语卡、概念卡、费曼解释、小白解释，或要求把概念结合本次材料的实际使用来制作卡片时，读取 `references/term-concept-card.md`。

主规则：

- 只为长期复用的概念缺口制卡。
- 按 reference 判定为定义型、区分型或运作型。
- 字段固定为 `Why`、`What`、`How`。
- `##` 卡片正面必须单独占一行；不要把 `【Why】`、`【What】`、`【How】` 写进标题行。
- 全程使用费曼讲法，用普通语言、类比、反例或最小故事解释；不要用术语套术语。

## 校验清单

创建或修复 Decks 卡片后，必须先做文件级校验，再汇报完成。

```bash
python3 - <<'PY'
from pathlib import Path
import re

root = Path("PATH_TO_DECK_DOCS")
files = sorted(root.glob("*.md")) if root.is_dir() else [root]

print("files", len(files))
print("cards", sum(len(re.findall(r"^##\s+", p.read_text(encoding="utf-8"), re.M)) for p in files))
print("markdown_h3_plus", sum(len(re.findall(r"^#{3,6}\s+", p.read_text(encoding="utf-8"), re.M)) for p in files))
print("decks_title", sum(p.read_text(encoding="utf-8").count("decks/title") for p in files))
print("deck_tags", sum(len(re.findall(r"decks/[^\s]+", p.read_text(encoding="utf-8"))) for p in files))
print("heading_field_leak", sum(len(re.findall(r"^##\s+.*【(?:Why|What|How|背景|问题|对比|手段|应用|一句话定义|费曼解释|目标|解决方案|记忆钩子)", p.read_text(encoding="utf-8"), re.M)) for p in files))
PY
```

H2 Decks 卡片的常规期望：

- `cards` 等于预期 `##` 卡片数量。
- `markdown_h3_plus` 通常应该是 `0`。
- `decks_title` 应该是 `0`，除非用户明确要求这个标签。
- `heading_field_leak` 应该是 `0`；如果大于 `0`，说明背面字段被写进了卡片正面，Decks 会把第一张卡解析错。
- 每个聚合文档 frontmatter 都有一个最精确的 `decks/<领域>/<主题>` 标签。

如果 Decks 已经索引过这些文件，可以用当前数据库 schema 做只读校验：

```bash
sqlite3 --readonly ".obsidian/plugins/decks/flashcards.db" \
  "select source_file, count(*) from flashcards where source_file like 'PATH_PREFIX/%' group by source_file order by source_file;"
```

注意：当前表字段是 `source_file`，不是 `source_path`。

如果 Obsidian 正在占用数据库，导致 sqlite 报 disk I/O 或锁错误，不要编辑数据库。先做文件级校验，再让用户刷新或重开 Decks。

## 常见坑

- `##` 卡片下面的 `###` 会被 Decks 当成标题边界，导致卡片背面被截断。
- `##` 标题行混入 `【Why】`、`【What】`、`【How】` 会让 Decks 把背面内容当作正面。
- 卡片背面里的 Markdown `##` 会变成新的卡片；原文内部标题要改成 HTML。
- 标签不要只写 `decks/<主题>`；优先写 `decks/<领域>/<主题>`。
- 数据库宽泛查询可能把旧文件也算进去；要限定准确路径前缀。
- README 链接里的文件名可能和真实文件名不完全一致，常见差异是破折号、中文引号、英文引号。
- 本地 Obsidian 文件偶尔会报 `Resource deadlock avoided`。可以重试、在安全时处理扩展属性，或从源材料重新生成；不要因此直接改 Decks 数据库。
- 旧的散落笔记不要急着删，等聚合 Decks 卡片在 Obsidian 复习界面确认正常后再清理。

## 汇报格式

汇报完成时说明：

- Decks 文档放在哪里；
- 生成了几个聚合文档；
- 校验到多少个 `##` 卡片；
- 是否还有 Markdown H3+；
- 使用了什么 `decks/<领域>/<主题>` 标签；
- 是否完成数据库只读校验。
