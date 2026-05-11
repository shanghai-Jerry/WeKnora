# AgentQA 处理流程

## 概述

AgentQA 是一个基于 **ReAct（Reasoning + Acting）模式**的智能代理系统。与 KnowledgeQA 的静态管道不同，AgentQA 通过 LLM 动态决策工具调用，实现多轮推理和复杂任务处理。

**核心特点：**
- ReAct 循环（推理 + 行动 + 观察）
- LLM 动态决策工具调用
- 支持多轮迭代推理
- 丰富的工具生态（17+ 内置工具，含代码解释器和 HTML 渲染器）
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
| `code_interpreter` | 代码执行 | 在沙箱中执行 Python/JavaScript，支持数据分析与图表生成 |
| `html_interpreter` | HTML 渲染 | 将 HTML 渲染为可交互的网页报告，支持模板占位符替换 |

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

## 解释器工具（Interpreter Tools）

### 概述

解释器工具是 AgentQA 的扩展能力，支持在沙箱环境中执行代码并渲染可视化报告。目前包含两个工具：

- **`code_interpreter`** — 代码执行与数据分析
- **`html_interpreter`** — HTML 渲染与报告生成

两个工具通常配合使用：`code_interpreter` 负责生成数据和图表，`html_interpreter` 负责将结果渲染为交互式网页报告。

---

### `code_interpreter` 工具

**文件：** `internal/agent/tools/code_interpreter.go`

#### 功能

在沙箱环境中执行 Python 或 JavaScript 代码，用于：
- 数据分析和计算
- 生成图表、可视化
- 处理数据文件
- 执行科学计算

#### 参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `code` | string | ✅ | 要执行的代码 |
| `language` | string | — | `python`（默认）或 `javascript` |
| `file_path` | string | — | 可选文件路径，注入为 `FILE_PATH` 变量 |

#### 可用变量

执行环境中预置了以下变量：
- `PLOT_DIR` — 工作目录，用于保存输出文件（图表、图片）
- `FILE_PATH` — 用户提供的文件路径（如果指定了 `file_path`）
- Python 环境预装了 `pandas`、`numpy`

#### 输出

```
```python
<原始代码>
```

## Output
```
<stdout 输出>
```

## Generated Images
- data/tmp/{session_id}/chart.png
```

#### 安全机制

| 机制 | 说明 |
|------|------|
| 沙箱执行 | 通过 `sandbox.Manager` 在隔离环境中运行 |
| 超时控制 | 默认 60 秒超时 |
| 命令白名单 | 仅允许 `python3`、`node`、`bash`、`cat`、`ls` 等基础命令 |
| 输出截断 | stdout 超过 2000 字符自动截断 |
| 脚本清理 | 执行后自动删除临时脚本文件 |

#### 代码示例

**Python 数据分析：**
```python
import pandas as pd
import matplotlib.pyplot as plt

# 读取数据
df = pd.read_csv(FILE_PATH)

# 生成统计摘要
summary = df.describe()
print(summary)

# 保存图表
plt.figure(figsize=(10, 6))
df['column'].hist()
plt.savefig(f"{PLOT_DIR}/histogram.png")
```

**JavaScript 计算：**
```javascript
const data = [1, 2, 3, 4, 5];
const sum = data.reduce((a, b) => a + b, 0);
console.log(`Sum: ${sum}`);
```

---

### `html_interpreter` 工具

**文件：** `internal/agent/tools/html_interpreter.go`

#### 功能

将 HTML 内容渲染为可交互的网页报告。支持三种模式：

1. **内联 HTML**（默认）：直接传入 HTML 字符串
2. **文件模式**：从工作目录读取 HTML 文件
3. **模板模式**：读取模板文件并替换 `{{KEY}}` 占位符

#### 参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `html` | string | — | 内联 HTML 内容 |
| `file_path` | string | — | 工作目录中的 HTML 文件路径 |
| `title` | string | — | 报告标题（默认"HTML Report"） |
| `data` | map[string]string | — | 模板占位符替换数据 |

#### 使用场景

- `code_interpreter` 生成图表后，用此工具渲染包含图表的完整报告
- 展示数据仪表盘或交互式可视化
- 生成格式化的数据分析报告

#### 输出

返回 HTML 内容，前端以独立文档形式在侧边面板展示：
```json
{
  "output_type": "html",
  "title": "数据分析报告"
}
```

#### 与 `code_interpreter` 的配合

典型工作流：
```
1. code_interpreter: 生成数据和图表 → 保存到 PLOT_DIR
2. html_interpreter: 读取图表 + 生成完整 HTML 报告 → 前端展示
```

#### 自动包裹

如果传入的 HTML 不包含 `<!DOCTYPE` 或 `<html` 标签，工具会自动包裹为完整 HTML 文档：
```html
<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"></head>
<body>
  <!-- 用户传入的内容 -->
</body>
</html>
```

---

### 工具注册

**文件：** `internal/application/service/agent_service.go:439-447`

```go
case tools.ToolCodeInterpreter:
    sandboxMgr := s.getSandboxManager(ctx)
    toolToRegister = tools.NewCodeInterpreterTool(sandboxMgr, sessionID)

case tools.ToolHtmlInterpreter:
    workDir := fmt.Sprintf("data/tmp/%s", sessionID)
    toolToRegister = tools.NewHtmlInterpreterTool(sessionID, workDir)
```

- `code_interpreter` 依赖沙箱管理器（`sandbox.Manager`）执行代码
- `html_interpreter` 只需工作目录路径即可运行

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

## max_tokens 和 max_completion_tokens 参数说明

### 参数概述

| 参数 | 类型 | 说明 | 主要用途 |
|------|------|------|---------|
| `max_tokens` | int | 最大 token 数 | 主要用于 Ollama 等本地模型（`num_predict` 参数） |
| `max_completion_tokens` | int | 最大完成 token 数 | 用于 OpenAI、阿里云等远程 API |

### 定义位置

1. **ChatOptions 结构体** - `internal/models/chat/chat.go:33-34`
   ```go
   MaxTokens           int `json:"max_tokens"`
   MaxCompletionTokens int `json:"max_completion_tokens"`
   ```

2. **CustomAgentConfig 结构体** - `internal/types/custom_agent.go:91`
   ```go
   MaxCompletionTokens int `yaml:"max_completion_tokens" json:"max_completion_tokens"`
   ```

3. **AgentConfig 运行时** - `internal/types/agent.go`
   - Agent 模式下通过 `CustomAgentConfig.MaxCompletionTokens` 传递到 `SummaryConfig`

### 默认值和来源

#### 系统级默认值

| 来源 | 文件位置 | 值 | 说明 |
|------|---------|-----|------|
| 主配置文件 | `config/config.yaml:31` | `65535` | `conversation.summary.max_completion_tokens` |
| 组织配置 | `config/config-org.yaml:28` | `2048` | 组织级默认配置 |
| 前端默认值 | `frontend/src/views/agent/AgentEditorModal.vue:1492` | `2048` | Agent 编辑器默认值 |
| 前端默认值 | `frontend/src/views/settings/AgentSettings.vue:665` | `2048` | 设置页面默认值 |
| 数据库迁移默认 | `migrations/versioned/000006_custom_agents.up.sql:94,143` | `2048` | 新建 agent 的默认值 |

#### 内置 Agent 配置（AgentQA 相关）

| Agent | 文件位置 | max_completion_tokens 值 | 模式 |
|-------|---------|------------------------|------|
| Smart Reasoning (ReAct) | `config/builtin_agents.yaml:72` | `2048` | Agent 模式 |
| Deep Researcher | `config/builtin_agents.yaml:121` | `4096` | Agent 模式（多步推理需要更多 token） |

### 取值范围

- **最小值**: `1`（后端验证，见 `internal/handler/tenant.go:914`）
- **最大值**: `100000`（后端验证和前端限制）
- **前端输入限制**: `min=100`, `max=100000`（`frontend/src/views/agent/AgentEditorModal.vue:310`）
- **验证错误**: `"max_completion_tokens must be between 1 and 100000"`

### 参数优先级（AgentQA 流程）

1. **系统配置默认值**: `config.yaml` → `SummaryConfig.MaxCompletionTokens`
2. **CustomAgent 配置**: `custom_agent.config.max_completion_tokens`
3. **运行时覆盖**: 通过 `applyAgentOverridesToChatManage()` 应用到 `ChatManage.SummaryConfig`

### 代码中的使用

#### 1. Agent 配置覆盖 - `internal/application/service/session_qa_helpers.go:140-143`

```go
if customAgent.Config.MaxCompletionTokens > 0 {
    cm.SummaryConfig.MaxCompletionTokens = customAgent.Config.MaxCompletionTokens
    logger.Infof(ctx, "Using custom agent's max_completion_tokens: %d", customAgent.Config.MaxCompletionTokens)
}
```

#### 2. API 请求构建 - `internal/models/chat/remote_api.go:184-188`

```go
if opts.MaxTokens > 0 {
    req.MaxTokens = opts.MaxTokens
}
if opts.MaxCompletionTokens > 0 {
    req.MaxCompletionTokens = opts.MaxCompletionTokens
}
```

#### 3. Agent 引擎调用

在 AgentQA 的 ReAct 循环中，每次调用 LLM 时都会传递 `ChatOptions`，其中包含 `MaxCompletionTokens`：

```go
// internal/agent/engine.go - callLLMWithRetry()
opts := &chat.ChatOptions{
    Temperature:         e.config.Temperature,
    MaxCompletionTokens: e.config.MaxCompletionTokens,  // 从 agent config 获取
    // ...
}
```

### AgentQA 特殊说明

- **多轮推理影响**: AgentQA 的 ReAct 循环可能进行多轮 LLM 调用，`max_completion_tokens` 限制的是**单次调用**的最大生成 token 数，而非整个会话的总 token 数
- **与 MaxIterations 的区别**:
  - `max_completion_tokens`: 限制单次 LLM 调用的输出长度
  - `MaxIterations`: 限制 ReAct 循环的最大迭代次数（默认 10 次）
- **Deep Researcher Agent**: 由于需要进行更深入的分析和推理，默认 `max_completion_tokens` 设置为 `4096`，比普通 agent 的 `2048` 更高

### 两个参数的区别

- **`max_tokens`**: 传统参数，部分模型（如 Ollama、旧版 OpenAI API）使用。在 AgentQA 中较少直接使用。
- **`max_completion_tokens`**: OpenAI 新版 API 使用的参数（2023年11月后），是 AgentQA 中主要使用的参数，通过 `CustomAgentConfig` 配置。
- **兼容性处理**: `remote_api.go` 会同时检查两个参数，根据模型 API 的要求设置相应的字段。

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

**后端：**

| 文件 | 改动说明 |
|------|---------|
| `internal/types/agent.go` | `AgentConfig` 新增 `IntentExploreSystemBlock`、`IntentExploreContext` 运行时字段 |
| `internal/agent/prompts.go` | `BuildSystemPromptOptions` 新增 `IntentExploreBlock` 字段，`BuildSystemPromptWithOptions` 追注意图分析区块 |
| `internal/agent/engine.go` | `Execute()` 中传入 `IntentExploreBlock`；注入 `IntentExploreContext` 到初始 messages |
| `internal/application/service/session_agent_qa.go` | 新增 `executeIntentExplore()`、`formatIntentExploreSystemBlock()`、`formatIntentExploreContext()` 方法，在 `AgentQA()` 中调用 |
| `internal/application/service/chat_pipeline/query_intent_explore.go` | 提取 `parseOutput()` 和搜索逻辑为可导出函数，供 `session_agent_qa.go` 复用 |

**前端（AgentQA 模式下持久展示 intentExplore）：**

| 文件 | 改动说明 |
|------|---------|
| `frontend/src/views/chat/index.vue` | 在 `handleStreamData` 中新增 `query_intent_explore` 事件处理（双模式），并移除旧 pipeline stages 块中的 `query_intent_explore` 分支；`handleAgentChunk` 新建消息时初始化 `pipeline_stages: {}` |
| `frontend/src/views/chat/components/botmsg.vue` | 修改 `PipelineStagesDisplay` 的 `v-if` 条件，从 `!session.isAgentMode` 改为 `(!session.isAgentMode || hasIntentExplore)`；新增 `hasIntentExplore` 计算属性 |
| `frontend/src/views/chat/components/PipelineStagesDisplay.vue` | 无需改动（已有 `intentExplore` 展示逻辑，只需在 Agent 模式下也能挂载即可） |

无需改动的文件：
- `internal/types/custom_agent.go`：`EnableQueryIntentExplore` 字段已存在
- `internal/handler/session/agent_stream_handler.go`：已订阅并处理 `EventQueryIntentExplore`
- `frontend/src/views/chat/components/AgentStreamDisplay.vue`：无需改动（`intentExplore` 由 `PipelineStagesDisplay` 统一展示，不纳入 ReAct 事件流）

---

## AgentQA 模式下 knowledge_search 工具缺少引用事件的问题

### 问题描述

在 AgentQA 模式下，前端页面展示已检索文档时，一直显示"检索中"，无法正确展示检索结果。问题根因是：

**KnowledgeQA 模式**（流水线模式）：
- 在 `internal/application/service/session_knowledge_qa.go:204` 中发射 `EventAgentReferences` 事件
- 前端 `handleReferences` 处理器能正确接收并展示引用

**AgentQA 模式**（ReAct 循环模式）：
- `knowledge_search` 工具执行后（`internal/agent/act.go`）
- 只发射了 `EventAgentToolResult` 和 `EventAgentTool` 事件
- **缺少 `EventAgentReferences` 事件发射**
- 导致前端 `handleReferences` 未被触发，引用列表为空，一直显示"检索中"

### 解决方案

需要在 AgentQA 模式的工具执行流程中添加 `EventAgentReferences` 事件发射：

#### 修改文件 1：`internal/agent/tools/knowledge_search.go`

在 `Execute` 方法中，调用 `formatOutput` 生成结果后，将原始搜索结果（`*types.SearchResult` 数组）附加到 `ToolResult.Data` 中，以便后续事件发射使用。

**改动位置**：在 `formatOutput` 调用之后，`return result, nil` 之前（约第 413-418 行）

**添加代码**：
```go
// 将原始搜索结果附加到 Data 中，用于发射 EventAgentReferences
var rawResults []*types.SearchResult
for _, r := range deduplicatedResults {
    rawResults = append(rawResults, r.SearchResult)
}
if result.Data != nil {
    if dataMap, ok := result.Data.(map[string]interface{}); ok {
        dataMap["raw_results"] = rawResults
    }
}
```

#### 修改文件 2：`internal/agent/act.go`

添加辅助函数 `emitReferencesIfNeeded`，并在工具执行后调用。

**添加辅助函数**（在文件末尾或合适位置）：
```go
// emitReferencesIfNeeded 检查工具结果中是否包含搜索结果，如果有则发射 EventAgentReferences 事件
func (e *AgentEngine) emitReferencesIfNeeded(ctx context.Context, result *types.ToolResult, toolCallID, sessionID string) {
    if result == nil || result.Data == nil {
        return
    }
    
    dataMap, ok := result.Data.(map[string]interface{})
    if !ok {
        return
    }
    
    rawResults, ok := dataMap["raw_results"]
    if !ok {
        return
    }
    
    searchResults, ok := rawResults.([]*types.SearchResult)
    if !ok || len(searchResults) == 0 {
        return
    }
    
    e.eventBus.Emit(ctx, event.Event{
        ID:        toolCallID + "-references",
        Type:      event.EventAgentReferences,
        SessionID: sessionID,
        Data: event.AgentReferencesData{
            References: searchResults,
        },
    })
    
    logger.Debugf(ctx, "[Agent] Emitted EventAgentReferences with %d results", len(searchResults))
}
```

**修改 `executeSingleToolCall` 函数**（约第 160-203 行）：
在发射 `EventAgentToolResult` 和 `EventAgentTool` 事件之后，添加：
```go
// 发射引用事件（如果工具返回了搜索结果）
e.emitReferencesIfNeeded(ctx, result, toolCall.ID, sessionID)
```

**修改 `executeToolCallsParallel` 函数**（约第 91-158 行）：
在结果处理循环中，发射完 `EventAgentToolResult` 和 `EventAgentTool` 之后，添加：
```go
// 发射引用事件（如果工具返回了搜索结果）
e.emitReferencesIfNeeded(ctx, result, toolCall.ID, sessionID)
```

### 验证方式

1. 启动后端服务（`make dev-app`）
2. 在 AgentQA 模式下发送包含知识库检索的查询
3. 观察 SSE 流中是否包含 `references` 事件
4. 前端应正确展示检索到的文档列表，不再显示"检索中"

### 影响范围

- **后端**：仅修改 `knowledge_search.go` 和 `act.go` 两个文件
- **前端**：无需修改（已有 `handleReferences` 处理逻辑）
- **兼容性**：不影响 KnowledgeQA 模式的现有逻辑（两个模式独立运行）

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

---

## AI问数（SQL Query）功能

### 概述

AI问数功能扩展了AgentQA的能力，支持连接外部MySQL数据库进行数据查询。该功能通过新增`sql_query`工具实现，允许用户通过自然语言查询外部数据库。

**核心特点：**
- 只读查询：仅支持SELECT语句，确保数据安全
- 结果限制：最多返回50条数据 + 总数
- SQL注入防护：多层验证机制
- 多数据库支持：预留扩展接口，支持后续添加PostgreSQL、SQLite等

### 相关文件

#### 核心实现文件

| 文件路径 | 说明 |
|---------|------|
| `internal/datasource/dbconnector.go` | 数据库连接器接口定义 |
| `internal/datasource/connector/mysql/connector.go` | MySQL连接器实现 |
| `internal/agent/tools/sql_query.go` | SQL查询工具实现 |
| `internal/agent/tools/definitions.go` | 工具常量定义（ToolSQLQuery） |

#### 类型定义文件

| 文件路径 | 说明 |
|---------|------|
| `internal/types/qa_request.go` | QARequest添加DataSourceIDs字段 |
| `internal/types/agent.go` | AgentConfig添加DataSourceConfigs字段 |
| `internal/handler/session/types.go` | CreateKnowledgeQARequest添加DataSourceIDs |

#### 服务层文件

| 文件路径 | 说明 |
|---------|------|
| `internal/application/service/agent_service.go` | 注册sql_query工具 |
| `internal/application/service/session_agent_qa.go` | 解析数据源配置 |

---

### 架构设计

#### 数据库连接器接口

```go
// DBConnector is the interface that all database connectors must implement
type DBConnector interface {
    // Type returns the database type identifier
    Type() string
    
    // ValidateConnection tests the database connection
    ValidateConnection(ctx context.Context, config map[string]interface{}) error
    
    // ExecuteQuery executes a read-only SQL query
    ExecuteQuery(ctx context.Context, config map[string]interface{}, query string, maxRows int) (*QueryResult, error)
    
    // GetTableSchema returns schema information for all tables
    GetTableSchema(ctx context.Context, config map[string]interface{}) ([]TableInfo, error)
    
    // GetTableSchemaForTable returns schema for a specific table
    GetTableSchemaForTable(ctx context.Context, config map[string]interface{}, tableName string) (*TableInfo, error)
}
```

#### 扩展性设计

系统使用`DBConnectorRegistry`管理所有数据库连接器：

```go
// GlobalDBConnectorRegistry is the global registry for database connectors
var GlobalDBConnectorRegistry = NewDBConnectorRegistry()
```

添加新数据库支持只需：
1. 实现`DBConnector`接口
2. 在`init()`函数中注册连接器

示例（添加PostgreSQL支持）：

```go
// internal/datasource/connector/postgresql/connector.go
package postgresql

func init() {
    datasource.GlobalDBConnectorRegistry.Register(NewConnector())
}
```

---

### 安全机制

#### 1. 只读查询验证

```go
func validateReadOnlyQuery(query string) error {
    // 禁止的关键词
    forbiddenKeywords := []string{
        "INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
        "TRUNCATE", "REPLACE", "RENAME", "GRANT", "REVOKE",
    }
    
    // 必须以SELECT或WITH开头
    if !strings.HasPrefix(trimmed, "SELECT") && !strings.HasPrefix(trimmed, "WITH") {
        return fmt.Errorf("invalid query: only SELECT statements are allowed")
    }
}
```

#### 2. SQL注入防护

```go
func validateSQLSafety(sql string) error {
    // 检查多语句
    if strings.Contains(cleanSQL, ";") {
        return fmt.Errorf("multiple statements are not allowed")
    }
    
    // 检查危险函数
    dangerousFunctions := []string{
        "SLEEP(", "BENCHMARK(", "LOAD_FILE(", "INTO OUTFILE",
    }
}
```

#### 3. 结果限制

- 自动添加LIMIT子句（默认50条）
- 返回总行数供参考

---

### 工具参数

#### sql_query 工具输入

```typescript
interface SQLQueryInput {
    data_source_id: string;  // 数据源ID（从@选择获取）
    sql: string;             // SELECT查询语句
}
```

#### 工具输出格式

```markdown
=== Query Results ===

Returned 10 rows (Total matching rows: 150)

| id | name | created_at |
|----|------|------------|
| 1  | 示例 | 2024-01-01 |
| ...| ...  | ...        |
```

---

### 配置流程

#### 1. 添加数据源

通过数据源管理API添加MySQL连接：

```json
POST /api/v1/data_sources
{
    "name": "生产数据库",
    "type": "mysql",
    "config": {
        "host": "localhost",
        "port": 3306,
        "username": "reader",
        "password": "****",
        "database": "mydb",
        "charset": "utf8mb4"
    }
}
```

#### 2. 对话时选择数据源

在聊天请求中指定数据源ID：

```json
POST /api/v1/agent-chat/:session_id
{
    "query": "查询用户表有多少条记录",
    "agent_id": "agent-123",
    "data_source_ids": ["ds-456"]
}
```

---

### 处理流程

```
用户输入查询 + @选择数据源
    ↓
Handler解析DataSourceIDs
    ↓
sessionService.AgentQA()
    ↓
buildAgentConfig()
    ├─ 读取DataSourceIDs
    └─ resolveDataSourceConfigs() 获取数据源配置
    ↓
[条件注入] 检查DataSourceConfigs是否非空
    ├─ 若有数据源引用 → 获取数据库表结构信息
    │   ├─ 调用DBConnector.GetTableSchema() 获取所有表结构
    │   ├─ 格式化为"数据库上下文信息"（格式见下文）
    │   └─ 注入到System Prompt末尾
    └─ 若无数据源引用 → 跳过，进入正常流程
    ↓
agentService.CreateAgentEngine()
    ↓
registerTools()
    ├─ 检查DataSourceConfigs
    └─ 注册sql_query工具（仅当有数据源时注册）
    ↓
AgentEngine.Execute()
    ↓
executeLoop() (ReAct循环)
    ├─ [Think] LLM分析问题
    │   ├─ 若System Prompt中有数据库上下文 → LLM理解表结构，生成SQL
    │   └─ 若无数据库上下文 → 按常规Agent流程处理
    ├─ [Act] 调用sql_query执行查询（若有SQL）
    │   ├─ 验证SQL（只允许SELECT）
    │   ├─ 建立MySQL连接
    │   ├─ 执行查询
    │   └─ 格式化结果（Markdown表格）
    ├─ [Observe] 获取查询结果
    ├─ [Think] 基于结果生成答案
    └─ [Act] 调用final_answer返回结果
    ↓
前端展示答案和查询结果
```

---

### 条件注入逻辑

**触发条件**：用户在对话中通过 `@` 选择了一个或多个数据源（`data_source_ids` 参数非空）

**注入内容**：
1. 数据库名称
2. 可用表列表（逗号分隔）
3. 每张表的完整 `CREATE TABLE` 语句（包含字段注释）
4. 每张表的示例数据（前3行）

**注入位置**：System Prompt 末尾追加数据库上下文区块

**注入目的**：让 LLM 理解数据库结构，从而根据用户问题生成正确的 SQL 查询语句

**关键原则**：
- 仅当 `DataSourceConfigs` 非空时才触发注入，避免无数据源时浪费 token
- 注入格式严格遵循 `AI问数.md` 中定义的数据库上下文信息格式
- 如果数据源引用无效或获取表结构失败，降级为不注入（不影响正常流程）

---

### 数据库上下文注入格式

当用户引用了指定的 `datasourceID` 时，系统会将该数据源对应的数据库表结构信息注入到 System Prompt 中。格式如下：

```
## 数据库信息
- 数据库名: {database_name}
- 可用表: table1, table2, table3, ...
- 表结构:

CREATE TABLE table_name (
    column_name DATA_TYPE NOT NULL COMMENT 'column_comment' DEFAULT 'default_value',
    ...
    PRIMARY KEY (id)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='table_comment'

/*
3 rows from table_name table:
column1	column2	column3
value1	value2	value3
*/

其他表相关信息省略 ....

- 使用 'sql_query' 工具执行 SQL 查询
- **只允许 SELECT 查询，禁止 INSERT/UPDATE/DELETE/DROP/ALTER/TRUNCATE**
```

**格式说明**：
- `数据库名`：实际连接的数据库名称
- `可用表`：该数据源中所有可用的表名，用逗号分隔
- `表结构`：每张表的 `CREATE TABLE` DDL 语句，包含字段类型、注释、默认值
- `示例数据`：每张表的前3行数据，用于帮助 LLM 理解数据分布
- 末尾的 SQL 工具使用说明，引导 LLM 使用 `sql_query` 工具

---

### 代码示例

#### 注册新数据库连接器

```go
// internal/datasource/connector/postgresql/connector.go
package postgresql

import (
    "context"
    "github.com/Tencent/WeKnora/internal/datasource"
)

const ConnectorType = "postgresql"

type Connector struct{}

func NewConnector() *Connector {
    return &Connector{}
}

func (c *Connector) Type() string {
    return ConnectorType
}

func (c *Connector) ValidateConnection(ctx context.Context, config map[string]interface{}) error {
    // 实现PostgreSQL连接验证
}

func (c *Connector) ExecuteQuery(ctx context.Context, config map[string]interface{}, query string, maxRows int) (*datasource.QueryResult, error) {
    // 实现PostgreSQL查询执行
}

func (c *Connector) GetTableSchema(ctx context.Context, config map[string]interface{}) ([]datasource.TableInfo, error) {
    // 实现PostgreSQL表结构获取
}

func (c *Connector) GetTableSchemaForTable(ctx context.Context, config map[string]interface{}, tableName string) (*datasource.TableInfo, error) {
    // 实现PostgreSQL单表结构获取
}

func init() {
    // 注册到全局连接器注册表
    if err := datasource.GlobalDBConnectorRegistry.Register(NewConnector()); err != nil {
        panic(err)
    }
}
```

---

### 待完成功能

#### 前端@数据源选择

需要修改以下文件支持@数据源选择：

1. `frontend/src/components/MentionSelector.vue` - 添加数据源分组
2. `frontend/src/components/Input-field.vue` - 获取数据源列表
3. `frontend/src/views/chat/index.vue` - 发送data_source_ids

#### 多数据源支持

当前实现使用第一个数据源，后续可扩展为：
- 支持同时查询多个数据源
- 支持跨数据源JOIN查询

---

### 与现有工具的对比

| 维度 | database_query | sql_query |
|------|---------------|-----------|
| **查询目标** | 系统内部数据库 | 外部MySQL数据库 |
| **数据表** | knowledge_bases, knowledges, chunks | 用户配置的外部表 |
| **安全机制** | 自动注入tenant_id | 只读验证 + SQL注入防护 |
| **使用场景** | 查询知识库元数据 | 查询业务数据 |
| **扩展性** | 固定表结构 | 支持任意表结构 |
