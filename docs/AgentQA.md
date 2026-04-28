# AgentQA 处理流程

## 概述

AgentQA 是一个基于 **ReAct（Reasoning + Acting）模式**的智能代理系统。与 KnowledgeQA 的静态管道不同，AgentQA 通过 LLM 动态决策工具调用，实现多轮推理和复杂任务处理。

**核心特点：**
- ReAct 循环（推理 + 行动 + 观察）
- LLM 动态决策工具调用
- 支持多轮迭代推理
- 丰富的工具生态（15+ 内置工具）
- 技能系统（Skills）支持
- MCP（Model Context Protocol）集成
- SSE 流式响应

---

## 相关文件

### 核心实现文件

| 文件路径 | 行号 | 说明 |
|---------|------|------|
| `internal/router/router.go` | 334-336 | 路由注册：`agentChat.POST("/:session_id", handler.AgentQA)` |
| `internal/handler/session/qa.go` | 423-447 | Handler 层入口：请求解析和响应处理 |
| `internal/application/service/session_agent_qa.go` | 23-199 | Service 层：Agent 配置和引擎创建 |
| `internal/agent/engine.go` | 155-256 | Agent 引擎核心：ReAct 循环实现 |
| `internal/application/service/agent_service.go` | 79+ | Agent 服务：工具注册和引擎创建 |
| `internal/handler/session/agent_stream_handler.go` | - | SSE 流事件处理 |

### Agent 核心文件

| 文件路径 | 说明 |
|---------|------|
| `internal/agent/engine.go` | Agent 引擎：ReAct 循环 |
| `internal/agent/tools/registry.go` | 工具注册表 |
| `internal/agent/skills/manager.go` | 技能管理器 |
| `internal/agent/memory/` | 记忆和上下文压缩 |

### 类型定义文件

| 文件路径 | 说明 |
|---------|------|
| `internal/types/agent.go` | AgentConfig、AgentState、AgentStep 等 |
| `internal/types/custom_agent.go` | CustomAgent 及配置定义 |
| `internal/types/qa_request.go` | QARequest 结构体 |

---

## 关键结构体

### QARequest

**文件：** `internal/types/qa_request.go:6-20`

```go
type QARequest struct {
    Session            *Session     // 会话
    Query              string       // 用户查询
    AssistantMessageID string       // 助手消息ID
    SummaryModelID     string       // 模型覆盖
    CustomAgent        *CustomAgent // 自定义Agent配置
    KnowledgeBaseIDs   []string     // 知识库ID列表
    KnowledgeIDs       []string     // 特定知识ID列表
    ImageURLs          []string     // 图片URL（多模态）
    ImageDescription   string       // VLM生成的图片描述
    UserMessageID      string       // 用户消息ID
    WebSearchEnabled   bool         // 是否启用网页搜索
    EnableMemory       bool         // 是否启用记忆功能
    QuotedContext      string       // 引用消息上下文
}
```

### AgentConfig

**文件：** `internal/types/agent.go:15-62`

```go
type AgentConfig struct {
    MaxIterations          int      // 最大迭代次数
    AllowedTools          []string  // 允许使用的工具列表
    Temperature           float64   // 温度参数
    KnowledgeBases       []string  // 知识库列表
    KnowledgeIDs         []string  // 知识ID列表
    SystemPrompt         string    // 系统提示词
    WebSearchEnabled     bool      // 是否启用网页搜索
    WebSearchMaxResults  int       // 网页搜索最大结果数
    MultiTurnEnabled     bool      // 是否启用多轮对话
    HistoryTurns         int       // 历史轮数
    SkillsEnabled        bool      // 是否启用技能
    SkillDirs            []string  // 技能目录
    AllowedSkills        []string  // 允许使用的技能
    MCPSelectionMode     string    // MCP服务选择模式
    MCPServiceIDs        []string  // MCP服务ID列表
    // ... 更多字段
}
```

### AgentState

**文件：** `internal/types/agent.go:208-214`

```go
type AgentState struct {
    CurrentRound  int             // 当前轮次
    RoundSteps    []AgentStep     // 当前轮的步骤
    IsComplete    bool            // 是否完成
    FinalAnswer   string          // 最终答案
    KnowledgeRefs []*SearchResult // 知识引用
}
```

### AgentStep

**文件：** `internal/types/agent.go`

记录每一轮中的单个步骤（思考、工具调用、观察）。

---

## 完整处理流程

### 阶段1：HTTP 请求接收与解析

**文件：** `internal/handler/session/qa.go`

#### 1.1 路由注册（`router.go:334-336`）

```go
agentChat.POST("/:session_id", handler.AgentQA)
```

#### 1.2 AgentQA 入口函数（`qa.go:423-447`）

```
Handler.AgentQA(c *gin.Context)
├─ parseQARequest(c, "AgentQA")        // 行62-166: 解析请求
│   ├─ 获取sessionID和请求体
│   ├─ resolveAgent()                   // 行170: 解析Agent配置
│   ├─ mergeKnowledgeTargets()          // 合并@提及的知识库
│   ├─ saveImageAttachments()           // 处理图片上传
│   └─ 构建qaRequestContext
│
├─ 判断Agent模式 (quick-answer vs smart-reasoning)
└─ executeQA(reqCtx, qaModeAgent, true)  // 进入统一执行流程
```

---

### 阶段2：统一执行流程

**文件：** `internal/handler/session/qa.go`

#### executeQA 函数（`qa.go:459-593`）

```
executeQA()
├─ 发射EventAgentQuery事件 (行464-478)
├─ createUserMessage()           // 创建用户消息
├─ createAssistantMessage()      // 创建助手消息
├─ setupSSEStream()             // 设置SSE流 (行261-307)
│   ├─ setSSEHeaders()
│   ├─ writeAgentQueryEvent()
│   ├─ 创建EventBus和cancellable context
│   ├─ setupStopEventHandler()
│   ├─ setupStreamHandler()     // 订阅事件处理
│   └─ GenerateTitleAsync()     // 异步生成标题
│
└─ 启动异步goroutine执行QA (行538-587)
    ├─ runVLMAnalysisIfNeeded()  // VLM图片分析
    ├─ buildQARequest()          // 构建请求
    └─ sessionService.AgentQA()  // 调用服务层
```

---

### 阶段3：AgentQA 服务处理

**文件：** `internal/application/service/session_agent_qa.go`

#### AgentQA 函数（第23-199行）

```
sessionService.AgentQA()
├─ 验证customAgent必须存在 (行36-39)
├─ resolveRetrievalTenantID()    // 解析检索租户 (行42)
├─ 加载tenantInfo (行46-62)
├─ customAgent.EnsureDefaults()  // 设置默认值 (行65)
├─ buildAgentConfig()            // 构建AgentConfig (行68)
│   ├─ 配置基础参数 (MaxIterations, Temperature等)
│   ├─ configureSkillsFromAgent()  // 配置技能
│   ├─ resolveKnowledgeBases()    // 解析知识库
│   ├─ 配置AllowedTools
│   └─ 配置WebSearch参数
│
├─ resolveChatModelID()         // 解析模型 (行79)
├─ modelService.GetChatModel()  // 获取聊天模型 (行88)
├─ modelService.GetRerankModel() // 获取重排模型 (行104)
│
├─ getContextManagerForSession() // 获取上下文管理器 (行114)
├─ 设置系统提示词 (行118-125)
├─ getContextForSession()       // 获取LLM上下文 (行128)
├─ 处理多轮对话配置 (行138-142)
│
└─ agentService.CreateAgentEngine() // 创建Agent引擎 (行146)
    └─ engine.Execute()             // 执行Agent (行184)
```

#### buildAgentConfig 详解（第203行起）

根据 `CustomAgent` 配置构建 `AgentConfig`：

1. **基础参数**：
   - `MaxIterations`：从 `customAgent.AgentConfig.MaxIterations` 或默认值
   - `Temperature`：模型温度
   - `ModelID`：指定的模型ID

2. **技能配置**：
   - `SkillsEnabled`：是否启用技能系统
   - `SkillDirs`：技能目录列表
   - `AllowedSkills`：允许使用的技能（白名单）

3. **知识库配置**：
   - `KnowledgeBases`：知识库ID列表
   - `KnowledgeIDs`：特定知识ID列表

4. **工具配置**：
   - `AllowedTools`：允许使用的工具（白名单）
   - 如果未指定，使用默认工具列表

5. **网页搜索配置**：
   - `WebSearchEnabled`：是否启用
   - `WebSearchMaxResults`：最大结果数

6. **MCP配置**：
   - `MCPSelectionMode`：选择模式（`all`、`specified`、`disabled`）
   - `MCPServiceIDs`：指定的MCP服务ID

7. **多轮对话配置**：
   - `MultiTurnEnabled`：是否启用多轮
   - `HistoryTurns`：历史轮数

---

### 阶段4：Agent 引擎执行

**文件：** `internal/agent/engine.go`

#### AgentEngine.Execute 函数（第155-256行）

```
AgentEngine.Execute()
├─ 初始化AgentState (行174-179)
├─ 构建系统提示词 (行181-210)
│   ├─ 如果技能启用: BuildSystemPromptWithOptions(包含技能元数据)
│   └─ 否则: BuildSystemPromptWithOptions(基础版本)
│
├─ buildMessagesWithLLMContext() // 构建消息列表 (行218)
├─ buildToolsForLLM()           // 获取工具定义 (行221)
└─ executeLoop()                // 进入ReAct循环 (行231)
```

#### 系统提示词构建（第186-210行）

系统提示词包含：
- **角色定义**：Agent 的行为准则
- **可用工具列表**：工具名称、描述、参数
- **知识库信息**：可访问的知识库列表
- **技能元数据**（如果启用）：可用的技能列表和描述（Level 1 Progressive Disclosure）
- **使用说明**：如何调用工具、如何返回最终答案

**技能元数据格式**（Level 1 Progressive Disclosure）：
```
## Available Skills

### skill_name
Description: ...
Usage: Run `show_skill_details("skill_name")` to see full details.
```

---

### 阶段5：ReAct 循环

**文件：** `internal/agent/engine.go`

#### executeLoop 函数（第260-394行）

```
executeLoop()
└─ 循环 (直到完成或达到MaxIterations):
    ├─ 上下文窗口管理 (行296-300)
    │   ├─ estimateCurrentTokens()
    │   └─ manageContextWindow()
    │
    ├─ [Think] callLLMWithRetry()  // 调用LLM (行315)
    │   └─ LLM返回: content + tool_calls
    │
    ├─ [Analyze] analyzeResponse() // 分析响应 (行338)
    │   ├─ 检查是否调用final_answer工具 → 完成
    │   ├─ 检查是否自然停止
    │   └─ 检查空内容重试逻辑
    │
    ├─ [Act] executeToolCalls()   // 执行工具调用 (行369)
    │   └─ 对每个tool_call:
    │       ├─ 从ToolRegistry获取工具
    │       ├─ tool.Execute(args)
    │       └─ 发射EventAgentToolCall/EventAgentToolResult
    │
    └─ [Observe] appendToolResults() // 添加工具结果到消息 (行373)
```

#### ReAct 循环详细步骤

##### 步骤1：Think（思考）

调用 LLM 进行推理：

```go
response, err := e.callLLMWithRetry(ctx, messages, tools, options)
```

**LLM 输入：**
- 系统提示词（包含工具列表、知识库信息等）
- 对话历史（用户消息、工具调用结果等）

**LLM 输出：**
- `content`：推理过程的文本（可选）
- `tool_calls`：要调用的工具列表（可选）

**重试逻辑：**
- 如果 LLM 返回空内容且无工具调用，进行重试（最多3次）
- 如果重试次数用尽，触发反思（reflection）

##### 步骤2：Analyze（分析）

分析 LLM 响应：

```go
verdict := e.analyzeResponse(ctx, response, &state)
```

**分析逻辑：**
1. **检查是否完成**：
   - 如果调用了 `final_answer` 工具 → 设置 `isDone = true`
   - 提取最终答案内容

2. **检查是否自然停止**：
   - 如果 LLM 返回了文本内容，且无工具调用 → 视为最终答案

3. **检查空内容**：
   - 如果内容为空且无工具调用 → 触发重试或反思

##### 步骤3：Act（行动）

执行工具调用：

```go
e.executeToolCalls(ctx, response, &step, messages)
```

**执行流程：**
1. 遍历 `response.ToolCalls`
2. 从 `ToolRegistry` 获取工具实例
3. 调用 `tool.Execute(args)`
4. 发射事件：
   - `EventAgentToolCall`：工具调用开始
   - `EventAgentToolResult`：工具执行结果
5. 记录到 `AgentStep`

**工具执行错误处理：**
- 如果工具执行失败，将错误信息作为观察结果
- 不影响循环继续（除非达到最大迭代次数）

##### 步骤4：Observe（观察）

将工具结果添加到消息列表：

```go
messages = e.appendToolResults(ctx, messages, step)
```

**消息格式：**
```json
{
  "role": "tool",
  "tool_call_id": "call_xxx",
  "content": "工具执行结果..."
}
```

这一步完成后，循环继续，LLM 会根据新的观察结果进行下一轮推理。

---

### 阶段6：上下文窗口管理

**文件：** `internal/agent/engine.go`（第296-300行）

#### 问题
LLM 有上下文窗口限制（如 4096、8192、128k tokens），随着 ReAct 循环进行，消息列表会不断增长。

#### 解决方案

1. **估算当前 tokens**：
   ```go
   currentTokens := e.estimateCurrentTokens(messages)
   ```

2. **如果超出限制，进行压缩**：
   ```go
   if currentTokens > e.config.MaxContextTokens {
       messages = e.manageContextWindow(ctx, messages)
   }
   ```

3. **压缩策略**（由 `ContextManager` 实现）：
   - **摘要历史**：将旧的消息压缩成摘要
   - **保留最近 N 轮**：保留最近的对话轮数
   - **保留工具调用**：保留最近的工具调用和结果

---

### 阶段7：事件流处理

**文件：** `internal/handler/session/agent_stream_handler.go`

#### AgentStreamHandler.Subscribe（第61-80行）

订阅 Agent 相关事件：

```go
h.eventBus.On(event.EventAgentThought, h.handleThought)
h.eventBus.On(event.EventAgentToolCall, h.handleToolCall)
h.eventBus.On(event.EventAgentToolResult, h.handleToolResult)
h.eventBus.On(event.EventAgentReferences, h.handleReferences)
h.eventBus.On(event.EventAgentGraphData, h.handleGraphData)
h.eventBus.On(event.EventAgentFinalAnswer, h.handleFinalAnswer)
h.eventBus.On(event.EventAgentReflection, h.handleReflection)
h.eventBus.On(event.EventError, h.handleError)
h.eventBus.On(event.EventSessionTitle, h.handleSessionTitle)
h.eventBus.On(event.EventAgentComplete, h.handleComplete)
```

#### 事件处理器

##### handleThought（第83行起）
处理思考过程事件：
- 提取思考内容
- 通过 `streamManager.AppendEvent()` 发送 SSE 事件
- 前端展示："🤔 思考中..."

##### handleToolCall（第131行起）
处理工具调用事件：
- 提取工具名称、参数
- 发送 SSE 事件
- 前端展示："🔧 正在调用工具: knowledge_search"

##### handleToolResult（第164行起）
处理工具执行结果事件：
- 提取工具结果
- 发送 SSE 事件
- 前端展示工具返回的内容（或摘要）

##### handleReferences（第225行起）
处理知识引用事件：
- 提取引用的知识块（知识库、文档、chunk）
- 发送 SSE 事件
- 前端展示引用来源（可点击查看）

##### handleFinalAnswer（第323行起）
处理最终答案事件：
- 累加答案内容（流式）
- 当 `Done=true` 时，标记完成
- 发送完整的 final_answer 事件

##### handleReflection（第378行起）
处理反思事件：
- 当 LLM 返回空内容或异常时触发
- Agent 进行自我反思，调整策略
- 发送反思内容到前端

---

## 支持的工具

### 内置工具（`agent_service.go:312-440`）

| 工具名称 | 描述 | 适用场景 |
|---------|------|---------|
| `knowledge_search` | 知识库搜索 | 检索知识库内容 |
| `web_search` | 网络搜索 | 搜索互联网信息 |
| `web_fetch` | 网页内容抓取 | 获取网页详细内容 |
| `final_answer` | 最终答案 | 返回最终答案（必须调用） |
| `thinking` | 顺序思考 | 结构化思考过程 |
| `todo_write` | Todo 列表管理 | 任务分解和跟踪 |
| `grep_chunks` | 文本块搜索 | 在检索结果中搜索 |
| `data_analysis` | 数据分析 | 数据查询和分析 |
| `data_schema` | 数据模式查询 | 查看数据库结构 |
| `query_knowledge_graph` | 知识图谱查询 | 查询知识图谱 |
| `get_document_info` | 文档信息获取 | 获取文档元数据 |
| `list_knowledge_chunks` | 列出知识块 | 浏览知识库内容 |
| `show_skill_details` | 显示技能详情 | 查看技能完整说明 |
| `memory_save` | 保存记忆 | 保存长期记忆 |
| `memory_query` | 查询记忆 | 检索长期记忆 |

### MCP 工具

通过 MCP（Model Context Protocol）协议扩展的工具：
- 由 `MCPManager` 管理
- 支持动态加载和调用
- 可以访问外部服务和数据源

---

## 技能系统（Skills）

### 概述

技能系统是 AgentQA 的高级特性，允许 Agent 执行预定义的脚本（如 Python、Shell 等），实现复杂的数据处理、计算、API 调用等任务。

**特点：**
- **沙箱执行**：默认在 Docker 容器中执行，确保安全
- **Progressive Disclosure**：分级展示技能信息（Level 1: 元数据，Level 2: 完整说明）
- **动态加载**：从指定目录加载技能定义

### 技能文件结构

```
skills/
├── skill_name/
│   ├── skill.md       # 技能说明（Level 2 详情）
│   ├── script.py      # 执行脚本
│   └── config.json    # 配置文件（可选）
```

### 技能元数据（Level 1）

在系统提示词中展示：
```
## Available Skills

### data_visualization
Description: 使用 matplotlib 生成数据可视化图表
Usage: Run `show_skill_details("data_visualization")` to see full details.
```

### 查看技能详情（Level 2）

Agent 调用 `show_skill_details("skill_name")` 工具，返回完整的 `skill.md` 内容。

---

## ReAct 循环与 KnowledgeQA 的对比

| 维度 | KnowledgeQA | AgentQA |
|------|-------------|---------|
| **处理模式** | RAG 流水线（Pipeline） | ReAct 循环（Agent） |
| **执行方式** | 预定义阶段顺序执行 | LLM 动态决策工具调用 |
| **迭代次数** | 单次（无循环） | 多次（直到完成或达到 MaxIterations） |
| **工具调用** | 固定检索工具（并行） | 动态选择多种工具（串行/并行） |
| **推理能力** | 弱（仅查询改写） | 强（多轮推理、自我反思） |
| **适用场景** | 简单知识查询 | 复杂多步任务 |

---

## 配置示例

### CustomAgent 配置（`internal/types/custom_agent.go`）

```go
type CustomAgent struct {
    ID                   string                `json:"id"`
    Name                 string                `json:"name"`
    Description          string                `json:"description"`
    AgentConfig          AgentConfig           `json:"agent_config"`
    SummaryConfig        SummaryConfig         `json:"summary_config"`
    FallbackStrategy     string                `json:"fallback_strategy"`
    EnableQueryExpansion bool                  `json:"enable_query_expansion"`
    // ... 更多字段
}
```

### config.yaml 中的相关配置

```yaml
agent:
  defaultMaxIterations: 10
  defaultTemperature: 0.7
  defaultToolCallTimeout: 60s
  
skills:
  enabled: true
  dirs:
    - "./skills"
  
mcp:
  enabled: true
  selectionMode: "all"  # all, specified, disabled
```

---

## 错误处理

### 常见错误场景

1. **达到最大迭代次数**：
   - 触发反思（reflection）
   - 返回当前最佳答案

2. **工具调用失败**：
   - 将错误作为观察结果
   - 继续循环（LLM 可以调整策略）

3. **上下文窗口溢出**：
   - 触发上下文压缩
   - 如果压缩失败，返回错误

4. **LLM 调用失败**：
   - 重试（最多3次）
   - 如果仍然失败，返回错误

### 错误事件

- `EventError`：通用错误事件
- 反思事件：当 Agent 遇到问题时，进行自我反思

---

## 性能考虑

### 延迟

AgentQA 的延迟通常高于 KnowledgeQA，因为：
- 多轮 LLM 调用
- 工具执行时间
- 上下文压缩开销

**优化策略：**
- 减少不必要的工具调用
- 使用更快的模型（如 GPT-3.5 代替 GPT-4）
- 优化工具实现（缓存、并行等）

### 成本

AgentQA 的 API 调用成本更高，因为：
- 多次 LLM 调用（每轮至少1次）
- 更长的上下文（历史消息 + 工具结果）

**优化策略：**
- 限制 `MaxIterations`
- 使用上下文压缩
- 选择合适的模型

---

## 与 KnowledgeQA 的详细对比

### 架构模式

**KnowledgeQA**（`session_knowledge_qa.go:146-183`）：
```go
// 静态pipeline组装
pipeline = types.NewPipelineBuilder().
    Add(types.LOAD_HISTORY).
    Add(types.QUERY_UNDERSTAND).
    // ... 固定阶段
    Build()

// 顺序执行
for _, eventType := range eventList {
    s.eventManager.Trigger(ctx, eventType, chatManage)
}
```

**AgentQA**（`engine.go:274-383`）：
```go
// 动态ReAct循环
for state.CurrentRound < e.config.MaxIterations {
    // 1. Think: LLM决定下一步
    response := e.callLLMWithRetry(ctx, messages, tools, ...)
    
    // 2. Analyze: 检查是否需要调用工具或完成
    verdict := e.analyzeResponse(ctx, response, ...)
    if verdict.isDone { break }
    
    // 3. Act: 执行工具
    e.executeToolCalls(ctx, response, &step, ...)
    
    // 4. Observe: 更新消息
    messages = e.appendToolResults(ctx, messages, step)
}
```

### 配置方式

**KnowledgeQA** - 使用 `CustomAgentConfig`：
- `EnableQueryExpansion` - 查询扩展
- `EnableQueryIntentExplore` - 查询意图探索
- `EnableRewrite` - 查询重写
- `FallbackStrategy` - 回退策略
- `RerankTopK`、`VectorThreshold` 等检索参数

**AgentQA** - 使用 `AgentConfig`：
- `MaxIterations` - 最大迭代次数
- `AllowedTools` - 允许的工具列表
- `SkillsEnabled` / `SkillDirs` - 技能管理
- `MCPSelectionMode` - MCP 服务选择
- `MultiTurnEnabled` - 多轮对话

### 系统提示词

**KnowledgeQA**：
- 使用 `SummaryConfig.Prompt` 和 `ContextTemplate`
- 静态模板，包含检索结果格式化

**AgentQA**（`engine.go:186-210`）：
```go
// 动态构建，包含知识库信息、技能元数据等
systemPrompt = BuildSystemPromptWithOptions(
    e.knowledgeBasesInfo,
    e.config.WebSearchEnabled,
    e.selectedDocs,
    &BuildSystemPromptOptions{
        SkillsMetadata: skillsMetadata,
        Language:       language,
        Config:         e.appConfig,
    },
    e.systemPromptTemplate,
)
```

### 适用场景

| 场景 | KnowledgeQA | AgentQA |
|------|-------------|---------|
| 简单知识查询 | ✅ 适合（快速） | ✅ 可用（但过重） |
| 复杂多步推理 | ❌ 不支持 | ✅ 适合 |
| 需要工具调用 | ❌ 固定检索工具 | ✅ 动态选择多种工具 |
| 数据分析 | ⚠️ 单轮分析 | ✅ 多轮探索性分析 |
| 响应速度 | ✅ 较快（固定路径） | ⚠️ 较慢（多轮迭代） |
| 成本 | ✅ 较低（单次调用） | ⚠️ 较高（多次调用） |

---

## 总结

AgentQA 是一个强大的智能代理系统，通过 ReAct 模式实现复杂的推理和工具调用。

**优势：**
1. **灵活性**：LLM 动态决策，适应各种复杂任务
2. **扩展性**：丰富的工具生态 + 技能系统 + MCP 集成
3. **推理能力**：多轮推理、自我反思、上下文压缩
4. **可观测性**：详细的事件流，方便调试和优化

**劣势：**
1. **延迟**：多轮调用导致响应时间较长
2. **成本**：多次 API 调用增加成本
3. **复杂性**：系统更复杂，调试难度更大

**选择建议：**
- 如果是简单的知识库查询，使用 **KnowledgeQA**
- 如果需要多步推理、工具调用、数据分析，使用 **AgentQA**

---

## Query Intent Explore 集成方案

### 背景

Query Intent Explore（查询意图探索）是 KnowledgeQA 流水线中的一个插件（`PluginQueryIntentExplore`），负责将用户原始查询拆解为多个分析路径，生成针对性搜索查询，并行执行检索并汇聚结果。该功能在 AgentQA 模式下尚未启用，需要集成以提升 Agent 在复杂知识检索场景下的召回质量。

### 与 KnowledgeSearchTool.queries 的关系分析

**核心问题**：`knowledge_search` 工具已支持 1-5 个 queries，LLM 在 ReAct 循环中可以自行生成多个搜索查询。前置意图探索是否与之冲突？

**两者对比：**

| 维度 | KnowledgeSearch.queries（Agent 现状） | Intent Explore（KnowledgeQA 插件） |
|------|--------------------------------------|----------------------------------|
| 查询生成者 | Agent LLM 在通用推理中随手生成 | 专用提示词引导 LLM 深度分析意图 |
| 分析深度 | 表层：LLM 快速拆出几个搜索角度 | 深层：识别实体、维度、实体间关系路径 |
| 检索执行 | 单次工具调用，1-5 个 queries 并行搜索 | 多路径完全并行，每路径独立检索+去重 |
| 检索时机 | ReAct 循环中，LLM 决定何时调用 | 流水线固定阶段，查询后立即执行 |
| 可靠性 | 依赖 LLM 自觉拆解，质量不稳定 | 专用提示词强制结构化输出，稳定 |

**结论**：两者**功能有重叠但不冲突**，关键差异在于"分析深度"和"检索时机"。Agent 模式下 LLM 可能给出泛泛的 queries，而意图探索的专用提示词能更系统地拆解查询维度。但如果前置意图探索的结果仅以纯文本注入上下文，LLM 很可能忽略它，仍然自己调用 `knowledge_search` 做重复检索——这才是需要解决的核心问题。

### 集成方案：双通道注入（意图分析注入 System Prompt + 检索结果注入初始上下文）

**核心思路**：前置意图探索的输出分两部分利用：
1. **意图分析结果**（分析路径、实体、维度、关系）→ 注入 **System Prompt**，使 LLM 在后续推理中"知道"查询已被如何拆解，引导其更精准地使用 `knowledge_search` 的 queries 参数
2. **多路径检索结果** → 注入 **初始上下文消息**，使 Agent 首轮即可获得检索信息，减少盲目工具调用

#### 为什么不能只注入搜索结果

| 方案 | 问题 |
|------|------|
| 仅注入搜索结果到 user message | LLM 可能忽略这段上下文，仍自行调用 `knowledge_search` 用泛泛的 queries 重复检索 |
| 仅注入意图分析到 system prompt | LLM 知道了拆解维度但没有检索结果，仍需调用工具搜索，只是 queries 质量会更好 |
| **双通道注入**（推荐） | System Prompt 中的意图分析引导 LLM 更精准地检索；初始上下文中的检索结果减少首轮盲目调用 |

#### System Prompt 注入内容设计

在 System Prompt 末尾追加意图分析区块（类似 skills metadata 的追加方式）：

```
## Intent Explore Analysis

The user's query has been pre-analyzed with the following intent structure:

Original Query: "药物A和药物B的相互作用"

### Analysis Paths
| Path | Entity | Dimensions | Search Strategy |
|------|--------|-----------|-----------------|
| 1 | 药物A | 药代动力学, 副作用 | "药物A的药代动力学机制" |
| 2 | 药物B | 药代动力学, 副作用 | "药物B代谢途径和副作用" |
| 3 | 药物A↔药物B | 相互作用机制, 临床意义 | "药物A与药物B的相互作用机制" |

### Pre-searched Queries
["药物A的药代动力学机制", "药物B代谢途径和副作用", "药物A与药物B的相互作用机制"]

### Guidance
- Multi-path search has already been executed. Results are provided in the conversation context.
- If you need to search further, use the analysis paths above as reference for constructing precise queries.
- Focus on queries that the pre-search may have missed (e.g., specific clinical guidelines, dosage adjustments).
```

**效果**：LLM 在每一轮推理中都能看到意图分析结构，知道查询已被从哪些维度拆解，从而：
- 构造更精准的 `knowledge_search` queries（不再泛泛搜索）
- 避免重复搜索已有结果
- 针对性地补充预搜索未覆盖的角度

#### 初始上下文注入内容设计

将多路径检索结果作为一条 user 消息，插入在 system prompt 之后、用户 query 之前：

```
[Pre-search Results from Intent Explore]
Based on the intent analysis of your query, the following multi-path search has been performed:

Search Queries: ["药物A的药代动力学机制", "药物B代谢途径和副作用", "药物A与药物B的相互作用机制"]
Total Results: 15

=== Search Results ===

[Source Document: 药物A说明书]
Result #1:
  Content: 药物A主要通过CYP3A4酶代谢...
  ...

[Source Document: 药物相互作用数据库]
Result #2:
  Content: 药物A与药物B存在竞争性抑制...
  ...
```

#### 流程设计

```
sessionService.AgentQA()
  ├─ ... 现有流程（buildAgentConfig, resolveModel 等）
  │
  ├─ [新增] 检查是否启用意图探索
  │   ├─ 读取 customAgent.Config.EnableQueryIntentExplore
  │   ├─ 若为 nil，回退到全局配置 s.cfg.Conversation.EnableQueryIntentExplore
  │   └─ 若未启用，跳过意图探索，直接进入引擎创建
  │
  ├─ [新增] 执行意图探索 executeIntentExplore()
  │   ├─ 读取意图探索提示词（config.Conversation.IntentExplorePrompt / IntentExplorePromptUser）
  │   ├─ 调用 LLM 进行查询拆解
  │   ├─ 解析 LLM 输出为 intentExploreOutput（analysis_paths + final_search_queries）
  │   ├─ 对每个 final_search_query 并行执行检索
  │   ├─ 合并去重搜索结果
  │   ├─ 发送 EventQueryIntentExplore 事件（SSE → 前端）
  │   └─ 返回 (IntentExploreData, []*SearchResult)
  │
  ├─ [新增] 构建双通道注入内容
  │   ├─ IntentExploreSystemBlock → 写入 AgentConfig.IntentExploreSystemBlock
  │   └─ IntentExploreContext → 写入 AgentConfig.IntentExploreContext
  │
  ├─ agentService.CreateAgentEngine()  // 现有流程
  │
  └─ AgentEngine.Execute()
      ├─ [新增] System Prompt 末尾追加 IntentExploreSystemBlock
      ├─ [新增] 初始 messages 中 system prompt 后插入 IntentExploreContext
      ├─ executeLoop() (ReAct循环)
      │   → LLM 首轮即可看到意图分析 + 检索结果
      │   → 后续轮次仍受 system prompt 中意图分析引导
      │   → 可自行调用 knowledge_search 补充检索
      └─ 完成
```

#### 详细实现步骤

##### 步骤1：在 `session_agent_qa.go` 中添加意图探索执行逻辑

**文件**：`internal/application/service/session_agent_qa.go`

在 `AgentQA()` 函数中，`buildAgentConfig()` 之后、`CreateAgentEngine()` 之前，添加：

```go
// 读取意图探索开关：智能体配置 > 全局配置
enableIntentExplore := s.cfg.Conversation.EnableQueryIntentExplore
if req.CustomAgent.Config.EnableQueryIntentExplore != nil {
    enableIntentExplore = *req.CustomAgent.Config.EnableQueryIntentExplore
}

if enableIntentExplore {
    intentData, searchResults := s.executeIntentExplore(ctx, req.Query, summaryModel, agentConfig, eventBus, sessionID)
    if intentData != nil {
        agentConfig.IntentExploreSystemBlock = formatIntentExploreSystemBlock(intentData)
        agentConfig.IntentExploreContext = formatIntentExploreContext(intentData, searchResults)
    }
}
```

新增 `executeIntentExplore()` 方法，逻辑参考 `PluginQueryIntentExplore.OnEvent()`：

```go
func (s *sessionService) executeIntentExplore(
    ctx context.Context,
    query string,
    chatModel chat.Chat,
    agentConfig *types.AgentConfig,
    eventBus *event.EventBus,
    sessionID string,
) (*types.IntentExploreData, []*types.SearchResult) {
    // 1. 读取提示词配置
    promptContent := s.cfg.Conversation.IntentExplorePrompt
    if promptContent == "" {
        return nil, nil
    }
    userContent := s.cfg.Conversation.IntentExplorePromptUser
    if userContent == "" {
        userContent = query
    } else {
        userContent = strings.ReplaceAll(userContent, "{{query}}", query)
    }

    // 2. 调用 LLM 进行查询拆解（ChatStream 流式收集，与 PluginQueryIntentExplore 一致）
    // 3. 解析输出为 intentExploreOutput
    // 4. 对每个 final_search_query 并行执行检索（复用搜索服务逻辑）
    // 5. 合并去重搜索结果
    // 6. 发送 EventQueryIntentExplore 事件
    // 7. 返回意图探索数据和搜索结果
}
```

**注意**：`parseIntentExploreOutput()` 和搜索逻辑可提取为共享工具函数，从 `chatpipeline` 包导出或在 `service` 包中独立实现，避免循环依赖。

##### 步骤2：AgentConfig 新增运行时字段

**文件**：`internal/types/agent.go`

```go
type AgentConfig struct {
    // ... 现有字段 ...

    // Intent explore system prompt block appended to system prompt (runtime only)
    IntentExploreSystemBlock string `json:"-"`
    // Intent explore search results injected as initial context (runtime only)
    IntentExploreContext string `json:"-"`
}
```

##### 步骤3：System Prompt 注入意图分析

**文件**：`internal/agent/prompts.go`

在 `BuildSystemPromptWithOptions()` 中，追注意图分析区块：

```go
func BuildSystemPromptWithOptions(...) string {
    // ... 现有逻辑 ...

    // Append intent explore analysis if available
    if options != nil && options.IntentExploreBlock != "" {
        basePrompt += "\n\n" + options.IntentExploreBlock
    }

    return basePrompt
}
```

**文件**：`internal/agent/prompts.go`

`BuildSystemPromptOptions` 新增字段：

```go
type BuildSystemPromptOptions struct {
    SkillsMetadata    []*skills.SkillMetadata
    Language          string
    Config            *config.Config
    IntentExploreBlock string  // [新增] 意图分析区块文本
}
```

**文件**：`internal/agent/engine.go`

在 `Execute()` 中构建 system prompt 时传入意图分析：

```go
systemPrompt = BuildSystemPromptWithOptions(
    e.knowledgeBasesInfo,
    e.config.WebSearchEnabled,
    e.selectedDocs,
    &BuildSystemPromptOptions{
        Language:           language,
        Config:             e.appConfig,
        IntentExploreBlock: e.config.IntentExploreSystemBlock, // [新增]
    },
    e.systemPromptTemplate,
)
```

##### 步骤4：初始上下文注入检索结果

**文件**：`internal/agent/engine.go`

在 `Execute()` 中，`buildMessagesWithLLMContext()` 之后，插入意图探索检索结果：

```go
messages := e.buildMessagesWithLLMContext(systemPrompt, query, sessionID, llmContext, imgs)

// [新增] 注入意图探索检索结果到初始上下文
if e.config.IntentExploreContext != "" {
    intentMsg := chat.Message{
        Role:    "user",
        Content: "[Pre-search Results from Intent Explore]\n" + e.config.IntentExploreContext,
    }
    assistantAck := chat.Message{
        Role:    "assistant",
        Content: "Understood. I have the pre-search results from intent explore analysis. I'll use these as context and search further only if needed.",
    }
    // 插入到 user query 之前，并添加 assistant 回应以保持对话结构
    insertIdx := len(messages) - 1 // user message 是最后一条
    messages = append(
        messages[:insertIdx],
        intentMsg,
        assistantAck,
        messages[insertIdx], // 原始 user query
    )
}
```

**为什么需要 assistant 回应**：直接在 user query 前插入一条 user 消息会破坏对话交替结构（连续两条 user 消息），部分 LLM 提供商会拒绝或行为异常。添加一条简短的 assistant 回应保持 user/assistant 交替。

##### 步骤5：格式化函数实现

```go
// formatIntentExploreSystemBlock 生成注入 System Prompt 的意图分析区块
func formatIntentExploreSystemBlock(data *types.IntentExploreData) string {
    var sb strings.Builder
    sb.WriteString("## Intent Explore Analysis\n\n")
    sb.WriteString("The user's query has been pre-analyzed with the following intent structure:\n\n")
    sb.WriteString(fmt.Sprintf("Original Query: %s\n\n", data.OriginalQuery))
    sb.WriteString("### Analysis Paths\n")
    sb.WriteString("| Path | Entity | Dimensions | Search Strategy |\n")
    sb.WriteString("|------|--------|-----------|-----------------|\n")
    for _, path := range data.AnalysisPaths {
        dims := strings.Join(path.Dimensions, ", ")
        if dims == "" {
            dims = "-"
        }
        searchStr := path.MergedSearchString
        if searchStr == "" && path.SourceEntity != "" {
            searchStr = fmt.Sprintf("%s ↔ %s (%s)", path.SourceEntity, path.TargetEntity, path.InteractionType)
        }
        sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n", path.PathID, path.Entity, dims, searchStr))
    }
    sb.WriteString(fmt.Sprintf("\n### Pre-searched Queries\n%v\n\n", data.FinalSearchQueries))
    sb.WriteString("### Guidance\n")
    sb.WriteString("- Multi-path search has already been executed. Results are provided in the conversation context.\n")
    sb.WriteString("- If you need to search further, use the analysis paths above as reference for constructing precise queries.\n")
    sb.WriteString("- Focus on queries that the pre-search may have missed (e.g., specific details, edge cases).\n")
    return sb.String()
}

// formatIntentExploreContext 生成注入初始上下文的检索结果
func formatIntentExploreContext(data *types.IntentExploreData, results []*types.SearchResult) string {
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("Search Queries: %v\n", data.FinalSearchQueries))
    sb.WriteString(fmt.Sprintf("Total Results: %d\n\n", len(results)))
    sb.WriteString("=== Search Results ===\n\n")
    for i, r := range results {
        sb.WriteString(fmt.Sprintf("Result #%d:\n", i+1))
        sb.WriteString(fmt.Sprintf("  Content: %s\n", r.Content))
        sb.WriteString(fmt.Sprintf("  Source: %s\n", r.KnowledgeTitle))
        sb.WriteString("\n")
    }
    return sb.String()
}
```

#### 上下文压缩保护

意图探索的注入内容可能很长，需要注意与 `manageContextWindow()` 的交互：

1. **System Prompt 中的意图分析**：System Prompt 不参与上下文压缩（始终保留），但会增加每轮的 prompt tokens。需要控制 `IntentExploreSystemBlock` 的长度，必要时截断分析路径表格。
2. **初始上下文中的检索结果**：会参与上下文压缩，随着 ReAct 循环进行，可能被 `memoryConsolidator` 摘要化。这是预期行为——早期检索结果在后续轮次中不再是最重要的上下文。
3. **建议**：在 `IntentExploreSystemBlock` 中只保留分析路径摘要（约 500 tokens），不包含搜索结果全文。

#### 对 ReAct 循环的影响

1. **System Prompt 引导**：LLM 每轮都能看到意图分析结构，构造 `knowledge_search` queries 时参考已有分析路径，减少泛泛搜索
2. **首轮加速**：Agent 首轮即有检索结果，可直接基于这些内容推理，无需先调用 `knowledge_search`
3. **减少重复检索**：System Prompt 明确告知"已执行多路径搜索"，LLM 倾向于直接使用已有结果或针对性补充
4. **与 knowledge_search.queries 互补而非冲突**：预搜索覆盖主路径，Agent 可针对遗漏细节（如具体剂量、特殊人群）做补充检索
5. **Token 开销**：System Prompt 增加约 500 tokens（意图分析），初始上下文增加取决于检索结果数量。总体可能因减少迭代轮次而降低总消耗。

#### 降级策略

- 若意图探索 LLM 调用失败 → 跳过意图探索，直接进入 ReAct 循环（与当前行为一致）
- 若意图探索结果解析失败 → 同上
- 若意图探索搜索结果为空 → 仅注入 System Prompt 意图分析（无检索结果），引导 LLM 自行搜索
- 开关关闭 → 完全跳过，零开销

#### 集成后的完整流程

```
HTTP Request
  → Handler.AgentQA()
    → executeQA()
      → sessionService.AgentQA()
        ├─ buildAgentConfig()
        ├─ [新增] 读取 EnableQueryIntentExplore 开关
        ├─ [新增] executeIntentExplore() (若启用)
        │   ├─ LLM 拆解查询为多路径搜索查询
        │   ├─ 并行执行多路径检索
        │   ├─ 发送 EventQueryIntentExplore → SSE → 前端
        │   └─ 返回 IntentExploreData + SearchResult
        ├─ [新增] 构建 IntentExploreSystemBlock + IntentExploreContext
        │   ├─ SystemBlock → AgentConfig.IntentExploreSystemBlock
        │   └─ Context → AgentConfig.IntentExploreContext
        ├─ agentService.CreateAgentEngine()
        ├─ AgentEngine.Execute()
        │   ├─ [新增] System Prompt 追加 IntentExploreSystemBlock
        │   ├─ [新增] 初始 messages 插入 IntentExploreContext
        │   ├─ executeLoop() (ReAct循环)
        │   │   → [Think] LLM 看到 system prompt 中的意图分析 + 上下文中的检索结果
        │   │   → [Act] 可直接推理，或针对性调用 knowledge_search 补充
        │   │   → [Observe] appendToolResults
        │   │   → 循环...
        │   └─ 完成
        └─ 事件发射 → SSE流 → 前端展示
```

#### 需修改的文件清单

| 文件 | 改动说明 |
|------|---------|
| `internal/types/agent.go` | `AgentConfig` 新增 `IntentExploreSystemBlock`、`IntentExploreContext` 运行时字段 |
| `internal/agent/prompts.go` | `BuildSystemPromptOptions` 新增 `IntentExploreBlock` 字段，`BuildSystemPromptWithOptions` 追注意图分析区块 |
| `internal/agent/engine.go` | `Execute()` 中传入 `IntentExploreBlock`；注入 `IntentExploreContext` 到初始 messages |
| `internal/application/service/session_agent_qa.go` | 新增 `executeIntentExplore()`、`formatIntentExploreSystemBlock()`、`formatIntentExploreContext()` 方法，在 `AgentQA()` 中调用 |
| `internal/application/service/chat_pipeline/query_intent_explore.go` | 提取 `parseOutput()` 和搜索逻辑为可导出函数，供 `session_agent_qa.go` 复用 |

无需改动的文件：
- `internal/types/custom_agent.go`：`EnableQueryIntentExplore` 字段已存在
- `internal/handler/session/agent_stream_handler.go`：已订阅并处理 `EventQueryIntentExplore`
- 前端代码：开关和展示逻辑已存在

---

## 关键路径总结

```
HTTP Request 
  → Handler.AgentQA() 
    → executeQA() 
      → sessionService.AgentQA() 
        → agentService.CreateAgentEngine() 
          → AgentEngine.Execute() 
            → executeLoop() (ReAct循环)
              → [Think] callLLM
              → [Act] executeToolCalls
              → [Observe] appendToolResults
              → 循环...
            → 完成
        → 事件发射 
          → SSE流 
            → 前端展示
```
