# State 只保存当前事实

State Schema 3 只保存 Requirement、Release 与不可变 WorkflowRef、全局唯一 current Step ID、当前状态与摘要、有效 outputs、外部 integrations 和最近 Event 指针。Stage 展示上下文、Step Instruction、历史 Evidence、门禁、回流记录和同步结果不进入 State。WorkflowState 由 State 与绑定 Workflow 派生，不落盘；回流时直接删除已失效 outputs。
