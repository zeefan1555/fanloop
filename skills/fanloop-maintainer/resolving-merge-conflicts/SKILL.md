---
name: resolving-merge-conflicts
description: 在最终 PR 交接阶段解决普通 main 集成冲突，同时保留已批准需求与 main 新行为。
---

1. 读取当前 merge 状态、两个 parent、全部冲突文件，以及能解释两边意图的提交和需求。
2. 解决普通文本与集成冲突，同时保留已批准需求行为和当前 main 行为，不发明新功能。
3. 若两边无法同时保留，且必须新增需求、扩大范围或改变未批准的 IDL、Workflow、Schema 或公开契约，不选边、不提交、不中止 merge；保留现场并上报 blocked。
4. 解决后运行受影响检查，完成 merge，并记录两个 parent SHA、冲突路径、命令和结果。
