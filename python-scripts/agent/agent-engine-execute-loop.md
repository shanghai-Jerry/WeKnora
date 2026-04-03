executeLoop 执行逻辑详解
整个 executeLoop 实现的是经典的 ReAct（Reason + Act）循环，下面逐个回答你的四个问题。

1. AgentStep 的作用
AgentStep 是单次迭代（Round）的完整快照，记录 Agent 在一轮循环中的"思考+行动"：

type AgentStep struct {
    Iteration int        // 第几轮（0-indexed）
    Thought   string     // LLM 的推理文本（Think 阶段的输出）
    ToolCalls []ToolCall // 本轮调用的所有工具及其结果
    Timestamp time.Time  // 发生时间
}

每一个 ToolCall 内嵌了完整的执行证据：

type ToolCall struct {
    ID         string                 // LLM 分配的 function call ID
    Name       string                 // 工具名称
    Args       map[string]interface{} // 调用参数
    Result     *ToolResult            // 执行结果（Output / Error / Success）
    Reflection string                 // （可选）反思内容
    Duration   int64                  // 执行耗时（ms）
}
生命周期：

每轮开始 → step := AgentStep{Iteration: N, Thought: LLM回答}
           ↓  遍历工具调用
           step.ToolCalls = append(..., toolCall)
           ↓  本轮结束
state.RoundSteps = append(state.RoundSteps, step)
AgentState.RoundSteps 积累了所有轮次的 AgentStep，最终随 EventAgentComplete 事件一起发送给上层，用于 **消息持久化** 和 **前端渲染执行步骤面板**。

2. 工具调用的处理流程
完整流程分为 识别 → 发事件 → 执行 → 收集结果 四步：

2.1 LLM 推理阶段识别工具调用

response, err := e.streamThinkingToEventBus(ctx, messages, tools, ...)
// response.ToolCalls 包含 LLM 决定调用的所有工具
在流式响应阶段，streamThinkingToEventBus 内部的 emitFunc 会在检测到 ResponseTypeToolCall 的 chunk 时，立即向 EventBus 发一个 EventAgentToolCall 的 "pending"（预告）事件，让前端提前显示"正在调用xxx工具"的状态。

2.2 特殊检查：final_answer 工具
在执行普通工具前，代码先检查是否有 final_answer 工具调用（步骤3）：

for _, tc := range response.ToolCalls {
    if tc.Function.Name == agenttools.ToolFinalAnswer {
        // 解析 answer 字段，设置 state.FinalAnswer，直接 break 跳出
        hasFinalAnswer = true
    }
}
if hasFinalAnswer { break }  // 终止整个循环
final_answer 是特殊工具，不走 toolRegistry.ExecuteTool，它的 answer 内容在流式阶段（ResponseTypeAnswer chunk）就已经实时推送给前端了。

2.3 普通工具调用执行

for i, tc := range response.ToolCalls {
    // a. 发送 EventAgentToolCall 事件（工具调用开始）
    e.eventBus.Emit(ctx, event.Event{Type: event.EventAgentToolCall, ...})
    
    // b. 执行工具（阻塞等待）
    result, err := e.toolRegistry.ExecuteTool(
        ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments),
    )
    
    // c. 构建 ToolCall 记录（包含入参+结果）
    toolCall := types.ToolCall{ID, Name, Args, Result, Duration}
    
    // d. 发送 EventAgentToolResult 事件（工具结果给前端）
    e.eventBus.Emit(ctx, event.Event{Type: event.EventAgentToolResult, ...})
    
    // e. 发送 EventAgentTool 事件（内部监控用）
    e.eventBus.Emit(ctx, event.Event{Type: event.EventAgentTool, ...})
    
    // f. （可选）Reflection：对工具结果再次 LLM 推理
    if e.config.ReflectionEnabled { ... }
    
    // g. 记录到 step
    step.ToolCalls = append(step.ToolCalls, toolCall)
}
注意：多个工具调用是串行执行的（for 循环，非并发），每个工具执行完再执行下一个。

3. 工具调用结果如何填充到上下文
这在 appendToolResults 方法中完成，在每轮循环末尾调用：

// 循环末尾
state.RoundSteps = append(state.RoundSteps, step)
messages = e.appendToolResults(ctx, messages, step)  // ← 填充上下文
state.CurrentRound++
appendToolResults 严格按照 OpenAI Function Calling 的消息格式 来组装：

Step 1: 追加 assistant 消息（包含 tool_calls 列表）
    {
        "role": "assistant",
        "content": step.Thought,          // LLM 的推理文本
        "tool_calls": [                   // LLM 本轮所有工具调用
            { "id": tc.ID, "type": "function", "function": {name, arguments} }
        ]
    }

Step 2: 对每个 ToolCall，追加 tool 消息（工具结果）
    {
        "role": "tool",
        "content": toolCall.Result.Output,  // 工具输出（失败时为 "Error: ..."）
        "tool_call_id": toolCall.ID,        // 与 assistant.tool_calls[i].id 对应
        "name": toolCall.Name
    }
同时这两类消息也会写入 contextManager（用于跨会话持久化）：

if e.contextManager != nil {
    e.contextManager.AddMessage(ctx, e.sessionID, assistantMsg)  // assistant 消息
    e.contextManager.AddMessage(ctx, e.sessionID, toolMsg)       // tool 消息
}
这样下一轮 LLM 调用时，messages 就包含了完整的"历史推理 + 工具结果"，LLM 可以基于此继续决策。

4. 上下文压缩在哪一步执行
压缩发生在 contextManager.AddMessage() 内部，即每次向持久化上下文写入消息时触发：

// context_manager.go: AddMessage() 内
func (cm *contextManager) AddMessage(..., message chat.Message) error {
    messages, _ := cm.storage.Load(ctx, sessionID)  // 加载历史
    messages = append(messages, message)              // 追加新消息

    // ↓↓↓ 压缩触发点 ↓↓↓
    tokenCount := cm.compressionStrategy.EstimateTokens(messages)
    if tokenCount > cm.maxTokens {                    // 超出 token 上限时
        compressed, _ := cm.compressionStrategy.Compress(ctx, messages, cm.maxTokens)
        messages = compressed
    }
    // ↑↑↑

    cm.storage.Save(ctx, sessionID, messages)         // 保存（可能是压缩后的）
}
压缩的触发时序：

每轮循环末尾
  └─ appendToolResults()
        └─ contextManager.AddMessage(assistantMsg)  → 检查 token → 可能压缩 → 持久化
        └─ contextManager.AddMessage(toolMsg_1)     → 检查 token → 可能压缩 → 持久化
        └─ contextManager.AddMessage(toolMsg_2)     → ...
重要区别：压缩影响的是持久化存储中的上下文（供下一个会话或 GetContext() 使用），不直接修改当前循环内的内存 messages 切片。当前循环内的 messages 是纯内存追加，不会被压缩截断——这意味着一次 Agent 执行如果工具调用很多，当轮传给 LLM 的消息可能会越来越大，直到触发 LLM 自身的 context limit。

整体流程总结图
Execute()
  └─ buildMessagesWithLLMContext()   // 组装初始消息 [system, history..., user]
  └─ buildToolsForLLM()              // 准备工具定义
  └─ executeLoop()
        │
        ├─ [Round N]
        │    ├─ 1. Think: streamThinkingToEventBus()
        │    │         └─ ChatStream() → 流式输出 → EventBus(EventAgentThought)
        │    │         └─ 检测到 toolCall chunk → EventBus(EventAgentToolCall pending)
        │    │
        │    ├─ 2. 检查 finish_reason==stop && no toolCalls → 直接结束，emit FinalAnswer
        │    │
        │    ├─ 3. 检查 final_answer tool → 提取 answer，结束循环
        │    │
        │    ├─ 4. Act: 遍历 toolCalls，执行每个工具
        │    │         ├─ emit EventAgentToolCall
        │    │         ├─ toolRegistry.ExecuteTool()
        │    │         ├─ emit EventAgentToolResult
        │    │         └─ step.ToolCalls = append(...)
        │    │
        │    ├─ 5. Observe: appendToolResults()
        │    │         ├─ messages += assistant msg (with tool_calls)
        │    │         ├─ messages += tool msg × N
        │    │         └─ contextManager.AddMessage() → 检查 token → 【压缩】→ 持久化
        │    │
        │    └─ state.CurrentRound++  →  进入下一轮
        │
        └─ 超过 MaxIterations → streamFinalAnswerToEventBus() 强制生成最终答案