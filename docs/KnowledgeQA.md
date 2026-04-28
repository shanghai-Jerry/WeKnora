# KnowledgeQA 处理流程

## 概述

KnowledgeQA 是一个基于 RAG（Retrieval-Augmented Generation）流水线的知识问答系统。采用事件驱动的可插拔架构，通过预定义的处理阶段顺序执行，实现从用户查询到最终答案的完整流程。

**核心特点：**
- 事件驱动的管道（Pipeline）架构
- 静态预定义的处理阶段
- 支持多模态（图片分析）
- SSE 流式响应
- 并行检索（向量 + 实体）
- 降级策略支持

---

## 相关文件

### 核心实现文件

| 文件路径 | 行号 | 说明 |
|---------|------|------|
| `internal/router/router.go` | 330 | 路由注册：`knowledgeChat.POST("/:session_id", handler.KnowledgeQA)` |
| `internal/handler/session/qa.go` | 398-408 | Handler 层入口：请求解析和响应处理 |
| `internal/application/service/session_knowledge_qa.go` | 20-221 | Service 层：核心业务逻辑 |
| `internal/handler/session/agent_stream_handler.go` | - | SSE 流事件处理 |
| `internal/application/service/chat_pipeline/chat_pipeline.go` | - | 事件管理器和插件机制 |

### 管道插件文件

| 阶段 | 插件文件 | 主要功能 |
|------|---------|---------|
| LOAD_HISTORY | `chat_pipeline/load_history.go` | 加载对话历史 |
| QUERY_UNDERSTAND | `chat_pipeline/query_understand.go` | 查询改写 + 意图分类 + 图片分析 |
| QUERY_INTENT_EXPLORE | `chat_pipeline/query_intent_explore.go` | 多路检索意图探索 |
| CHUNK_SEARCH_PARALLEL | `chat_pipeline/search_parallel.go` | 并行检索（向量 + 实体） |
| CHUNK_RERANK | `chat_pipeline/rerank.go` | 重排序过滤 |
| WEB_FETCH | `chat_pipeline/web_fetch.go` | 网页内容抓取 |
| CHUNK_MERGE | `chat_pipeline/merge.go` | 合并去重结果 |
| FILTER_TOP_K | `chat_pipeline/filter_top_k.go` | 过滤 TopK 结果 |
| DATA_ANALYSIS | `chat_pipeline/data_analysis.go` | 数据分析 |
| INTO_CHAT_MESSAGE | `chat_pipeline/into_chat_message.go` | 构建聊天上下文 |
| CHAT_COMPLETION_STREAM | `chat_pipeline/chat_completion_stream.go` | 流式聊天完成 |

### 类型定义文件

| 文件路径 | 说明 |
|---------|------|
| `internal/types/qa_request.go` | QARequest 结构体定义 |
| `internal/types/chat_manage.go` | ChatManage、PipelineRequest、PipelineState、PipelineContext |
| `internal/handler/session/types.go` | CreateKnowledgeQARequest 结构体 |

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

### CreateKnowledgeQARequest

**文件：** `internal/handler/session/types.go:40-53`

```go
type CreateKnowledgeQARequest struct {
    Query            string                 `json:"query" binding:"required"`
    KnowledgeBaseIDs []string               `json:"knowledge_base_ids"`
    KnowledgeIds     []string               `json:"knowledge_ids"`
    AgentEnabled     bool                   `json:"agent_enabled"`
    AgentID          string                 `json:"agent_id"`
    WebSearchEnabled bool                   `json:"web_search_enabled"`
    SummaryModelID   string                 `json:"summary_model_id"`
    MentionedItems   []MentionedItemRequest `json:"mentioned_items"`
    DisableTitle     bool                   `json:"disable_title"`
    EnableMemory     bool                   `json:"enable_memory"`
    Images           []ImageAttachment      `json:"images"`
    Channel          string                 `json:"channel"`
}
```

### ChatManage

**文件：** `internal/types/chat_manage.go:119-123`

包含三个核心组件：
- **PipelineRequest**：不可变配置（模型、知识库、检索参数等）
- **PipelineState**：可变中间数据（查询结果、消息列表等）
- **PipelineContext**：运行时上下文（EventBus、上下文管理器等）

---

## 完整处理流程

### 阶段1：HTTP 请求接收与解析

**文件：** `internal/handler/session/qa.go`

#### 1.1 路由匹配（第330行）
```
POST /knowledge-chat/:session_id
```

#### 1.2 KnowledgeQA 入口函数（第398-408行）
- 调用 `parseQARequest` 解析请求
- 调用 `executeQA` 执行处理

#### 1.3 parseQARequest 解析请求（第62-166行）
1. 获取 `session_id`（第67行）
2. 解析 JSON 请求体（第74-78行）
3. 验证查询内容（第81-84行）
4. **SSRF 保护**：清除客户端提供的 URL/Caption（第89-92行）
5. 获取会话信息（第101-105行）
6. 解析 Agent（第108行，调用 `resolveAgent`）
7. 合并 @ 提及项（第111行）
8. 处理图片附件（第120-136行）
9. 构建 `qaRequestContext`（第139-163行）

---

### 阶段2：执行准备

**文件：** `internal/handler/session/qa.go`

#### executeQA 函数（第459-593行）

1. **创建用户消息**（第481行）：`createUserMessage`
2. **创建助手消息**（第489行）：`createAssistantMessage`
3. **设置 SSE 流**（第503行）：`setupSSEStream`
   - 设置 SSE 头部（第263行）
   - 写入 `agent_query` 事件（第266行）
   - 创建 EventBus（第278行）
   - 设置停止事件处理器（第289行）
   - 设置流处理器（第292行，调用 `setupStreamHandler`）
   - 异步生成标题（第296-304行）

---

### 阶段3：异步服务调用

**文件：** `internal/handler/session/qa.go`（第537-587行）

在 goroutine 中执行：

1. **运行 VLM 分析**（如需要）（第560行）：`runVLMAnalysisIfNeeded`
2. **构建 QA 请求**（第563行）：`reqCtx.buildQARequest()`
3. **调用 Service 层**（第569行）：
   ```go
   serviceErr = h.sessionService.KnowledgeQA(streamCtx.asyncCtx, qaReq, streamCtx.eventBus)
   ```

---

### 阶段4：Service 层核心处理

**文件：** `internal/application/service/session_knowledge_qa.go`

#### KnowledgeQA 函数（第20-221行）

1. **解析知识库**（第35行）：`resolveKnowledgeBases`
2. **解析聊天模型 ID**（第38行）：`resolveChatModelID`
3. **初始化配置**（第44-56行）：从 `config.yaml` 加载 `SummaryConfig` 和 `FallbackStrategy`
4. **解析视觉能力**（第60-69行）：检查模型是否支持视觉
5. **解析检索租户**（第72行）：`resolveRetrievalTenantID`
6. **构建搜索目标**（第75行）：`buildSearchTargets`
7. **创建 ChatManage 对象**（第93-139行）：组装所有配置和状态
8. **应用 Agent 覆盖**（第143行）：`applyAgentOverridesToChatManage`
9. **组装管道**（第145-183行）：根据条件动态组装处理阶段

#### 管道组装逻辑（第145-183行）

**RAG 模式**（有知识库或网页搜索）：
```go
pipeline = types.NewPipelineBuilder().
    Add(types.LOAD_HISTORY).
    Add(types.QUERY_UNDERSTAND).
    AddIf(EnableQueryIntentExplore, types.QUERY_INTENT_EXPLORE).
    Add(types.CHUNK_SEARCH_PARALLEL).
    Add(types.CHUNK_RERANK).
    AddIf(WebSearchEnabled, types.WEB_FETCH).
    Add(types.CHUNK_MERGE).
    Add(types.FILTER_TOP_K).
    Add(types.DATA_ANALYSIS).
    Add(types.INTO_CHAT_MESSAGE).
    Add(types.CHAT_COMPLETION_STREAM).
    Build()
```

**纯聊天模式**（无知识库）：
```go
pipeline = types.NewPipelineBuilder().
    AddIf(hasHistory, types.LOAD_HISTORY).
    AddIf(EnableMemory, types.MEMORY_RETRIEVAL).
    Add(types.CHAT_COMPLETION_STREAM).
    AddIf(EnableMemory, types.MEMORY_STORAGE).
    Build()
```

10. **触发事件处理**（第191行）：`s.KnowledgeQAByEvent(ctx, chatManage, pipeline)`

---

### 阶段5：事件管道执行

**文件：** `internal/application/service/session_knowledge_qa.go`（第487-540行）

#### KnowledgeQAByEvent 函数

```go
for _, eventType := range eventList {
    stageStart := time.Now()
    err := s.eventManager.Trigger(ctx, eventType, chatManage)
    
    if err == chatpipeline.ErrSearchNothing {
        // 触发降级策略
        s.handleFallbackResponse(ctx, chatManage)
        return nil
    }
    
    if err != nil {
        return err.Err
    }
}
```

---

### 阶段6：管道插件详细处理

**事件管理器：** `internal/application/service/chat_pipeline/chat_pipeline.go`

#### 插件注册和执行机制

- **Plugin 接口**（第11-21行）：定义 `OnEvent` 和 `ActivationEvents` 方法
- **EventManager**（第24-29行）：管理插件注册和事件触发
- **Trigger 方法**（第71-78行）：调用事件对应的处理器

#### 各阶段插件实现详情

##### LOAD_HISTORY（`load_history.go`）
- 加载对话历史消息
- 构建上下文窗口

##### QUERY_UNDERSTAND（`query_understand.go`，第60-219行）
1. 加载对话历史（第84-93行）
2. 选择模型（第96行）：根据是否有图片选择 VLM 或普通模型
3. 构建提示词（第105行）
4. 发出图片分析事件（第118-129行）
5. 调用模型（第134-141行）
6. 解析输出（第179行）：提取改写查询、意图、图片描述
7. 发出 `query_rewritten` 事件（第182-191行）

**查询改写输出包含：**
- `rewritten_query`：改写后的查询
- `intent`：查询意图（`knowledge_query`、`chitchat`、`data_analysis` 等）
- `image_description`：图片描述（如果有）

##### QUERY_INTENT_EXPLORE（`query_intent_explore.go`）
- 多路检索意图探索
- 根据意图生成多个查询变体

##### CHUNK_SEARCH_PARALLEL（`search_parallel.go`，第91-200行）
1. 检查是否需要检索（第95行）：`chatManage.NeedsRetrieval()`
2. 克隆 ChatManage 避免并发问题（第110-113行）
3. **并行执行两个任务**：
   - **chunk_search**：向量检索（`search.go`）
   - **entity_search**：实体检索（`search_entity.go`）
4. 合并结果（第175-177行）
5. 去重（第177行）
6. 发出 `graph_data` 事件（第150-158行）

##### CHUNK_RERANK（`rerank.go`）
- 使用 Rerank 模型对检索结果重新排序
- 过滤低于阈值的結果

##### WEB_FETCH（`web_fetch.go`）
- 网页内容抓取（如果启用了网页搜索）
- 解析网页文本

##### CHUNK_MERGE（`merge.go`）
- 合并向量检索、实体检索、网页抓取的结果
- 去重处理

##### FILTER_TOP_K（`filter_top_k.go`）
- 根据配置过滤 TopK 结果
- 应用阈值过滤

##### DATA_ANALYSIS（`data_analysis.go`）
- 数据分析（如果查询意图是数据分析）
- 调用数据分析工具

##### INTO_CHAT_MESSAGE（`into_chat_message.go`）
- 构建聊天上下文
- 将检索结果格式化为上下文消息

##### CHAT_COMPLETION_STREAM（`chat_completion_stream.go`，第39-100+行）
1. 准备聊天模型和参数（第50行）
2. 准备消息列表（第57行）：`prepareMessagesWithHistory`
3. 检查 EventBus（第66行）
4. 调用模型流式接口（第89行）：`chatModel.ChatStream(ctx, chatMessages, opt)`
5. 消费流式响应并发送事件

---

### 阶段7：SSE 流事件处理

**文件：** `internal/handler/session/agent_stream_handler.go`

#### AgentStreamHandler.Subscribe（第61-80行）订阅的事件

**KnowledgeQA 管道阶段事件：**
- `EventQueryRewritten`（第74行）：查询改写事件
- `EventRetrievalQuery`（第75行）：检索查询事件
- `EventQueryExpansion`（第76行）：查询扩展事件
- `EventRetrievalVectorQ`（第77行）：向量检索事件
- `EventRetrievalKeywordQ`（第78行）：关键词检索事件
- `EventQueryIntentExplore`（第79行）：意图探索事件

**通用事件：**
- `EventAgentThought`：思考过程
- `EventAgentToolCall`：工具调用
- `EventAgentToolResult`：工具结果
- `EventAgentReferences`：知识引用
- `EventAgentGraphData`：知识图谱数据
- `EventAgentFinalAnswer`：最终答案
- `EventAgentReflection`：反思
- `EventError`：错误
- `EventSessionTitle`：会话标题
- `EventAgentComplete`：完成

#### handleFinalAnswer（第706-789行）
- 累加到 `finalAnswer`
- 通过 `streamManager` 发送 SSE 事件
- 当 `Done=true` 时完成

---

### 阶段8：响应完成

**文件：** `internal/handler/session/qa.go`

#### 完成处理（第506-534行）
1. 监听 `EventAgentFinalAnswer` 事件
2. 当 `Done=true` 时：
   - 更新助手消息（第526行）：`completeAssistantMessage`
   - 发出 `EventAgentComplete` 事件（第527-531行）
   - 消息入库（第670行）：`messageService.UpdateMessage`
   - 异步索引 Q&A 对（第675行）：`messageService.IndexMessageToKB`

---

## 流程图

```
HTTP POST /knowledge-chat/:session_id
    ↓
qa.go: KnowledgeQA (第398行)
    ↓
parseQARequest (第62行)
    ├─ 解析请求体
    ├─ 获取会话
    ├─ 解析Agent
    └─ 处理图片
    ↓
executeQA (第459行)
    ├─ createUserMessage
    ├─ createAssistantMessage
    ├─ setupSSEStream
    │   ├─ 创建EventBus
    │   ├─ 订阅事件处理器
    │   └─ 启动SSE流
    └─ 启动异步goroutine
        ↓
        runVLMAnalysisIfNeeded (图片分析)
        ↓
        session_knowledge_qa.go: KnowledgeQA (第20行)
            ├─ resolveKnowledgeBases
            ├─ resolveChatModelID
            ├─ buildSearchTargets
            ├─ 创建ChatManage
            └─ 组装pipeline
                ↓
                KnowledgeQAByEvent (第488行)
                    ↓
                    事件管道循环:
                    ├─ LOAD_HISTORY → load_history.go
                    ├─ QUERY_UNDERSTAND → query_understand.go
                    │   ├─ 改写查询
                    │   ├─ 意图分类
                    │   └─ 图片分析(可选)
                    ├─ QUERY_INTENT_EXPLORE (可选)
                    ├─ CHUNK_SEARCH_PARALLEL → search_parallel.go
                    │   ├─ chunk_search (向量)
                    │   └─ entity_search (实体)
                    ├─ CHUNK_RERANK → rerank.go
                    ├─ WEB_FETCH (可选) → web_fetch.go
                    ├─ CHUNK_MERGE → merge.go
                    ├─ FILTER_TOP_K → filter_top_k.go
                    ├─ DATA_ANALYSIS → data_analysis.go
                    ├─ INTO_CHAT_MESSAGE → into_chat_message.go
                    └─ CHAT_COMPLETION_STREAM → chat_completion_stream.go
                        └─ 流式输出 → SSE事件
                            ↓
                            agent_stream_handler.go 处理事件
                                ↓
                                SSE流发送到前端
                    ↓
                    完成处理:
                    ├─ completeAssistantMessage
                    ├─ 发出EventAgentComplete
                    └─ 异步索引Q&A对
```

---

## 关键组件和依赖

### 1. 事件系统

**文件：** `internal/event/event.go`

#### EventBus（第87-91行）
管理事件发布和订阅：
- `On(eventType, handler)`：注册处理器
- `Emit(ctx, event)`：发布事件（同步/异步）
- `EmitAndWait(ctx, event)`：发布并等待完成

#### EventType 常量（第14-71行）
- **查询处理**：`query.received`、`query.rewritten`、`query.expansion` 等
- **检索**：`retrieval.start`、`retrieval.vector`、`retrieval.keyword` 等
- **聊天**：`chat.start`、`chat.complete`、`chat.stream`
- **Agent**：`thought`、`tool_call`、`tool_result`、`final_answer` 等

---

### 2. 模型服务依赖

| 模型类型 | 用途 | 配置位置 |
|---------|------|---------|
| Chat Model (KnowledgeQA) | 对话生成、查询改写 | `ModelTypeKnowledgeQA` |
| Rerank Model | 搜索结果重排序 | `RerankModelID` |
| VLM Model | 图片分析 | `VLMModelID` |
| Embedding Model | 向量化 | 知识库配置 |

**模型 Provider**（`internal/models/provider/`）：
- OpenAI, Aliyun, SiliconFlow, Novita, GPUStack, OpenRouter 等

---

### 3. 存储和检索依赖

- **知识库服务**：`interfaces.KnowledgeBaseService`
- **知识服务**：`interfaces.KnowledgeService`
- **Chunk 存储库**：`interfaces.ChunkRepository`
- **图谱存储库**：`interfaces.RetrieveGraphRepository`
- **消息服务**：`interfaces.MessageService`

---

### 4. 配置依赖

#### SummaryConfig
**文件：** `session_knowledge_qa.go` 第44-51行

```go
SummaryConfig{
    Prompt:              s.cfg.Conversation.Summary.Prompt,
    ContextTemplate:     s.cfg.Conversation.Summary.ContextTemplate,
    Temperature:         s.cfg.Conversation.Summary.Temperature,
    MaxCompletionTokens: s.cfg.Conversation.Summary.MaxCompletionTokens,
    Thinking:            s.cfg.Conversation.Summary.Thinking,
}
```

#### 检索参数（第103-107行）
- `VectorThreshold`：向量阈值
- `KeywordThreshold`：关键词阈值
- `EmbeddingTopK`：向量检索 TopK
- `RerankTopK`：重排序 TopK
- `RerankThreshold`：重排序阈值

#### CustomAgent 配置
**文件：** `internal/types/custom_agent.go:64-196`

- `EnableQueryExpansion`：查询扩展
- `EnableQueryIntentExplore`：查询意图探索
- `EnableRewrite`：查询重写
- `FallbackStrategy`：回退策略（`fixed_reply` 或 `model_generate`）
- `RerankTopK`、`VectorThreshold` 等检索参数

---

## 关键设计模式

### 1. 事件驱动管道
使用插件模式和事件总线解耦各处理阶段。每个阶段作为独立的插件，通过事件触发执行。

### 2. 动态管道组装
根据条件（`needsRAG`、`hasKB`、`webSearchEnabled` 等）动态组装处理流程：
- **RAG 模式**：启用完整的检索增强生成流程
- **纯聊天模式**：仅使用对话历史进行聊天

### 3. SSE 流式响应
通过 EventBus 和 StreamManager 实现实时流式输出，前端可以实时看到：
- 查询改写过程
- 检索进度
- 流式生成答案

### 4. 并行处理
`CHUNK_SEARCH_PARALLEL` 阶段并行执行：
- 向量检索（语义相似度）
- 实体检索（知识图谱实体匹配）

提高检索效率，降低延迟。

### 5. 降级策略
当检索结果为空时（`ErrSearchNothing`），根据 `FallbackStrategy` 执行：
- `fixed_reply`：返回固定回复（如"抱歉，未找到相关信息"）
- `model_generate`：使用模型生成回复（不依赖检索结果）

### 6. 多模态支持
通过 VLM 模型支持图片分析：
- 检查模型能力（第60-69行）
- 如果有图片，选择 VLM 模型进行图片分析
- 生成图片描述，融入查询上下文

---

## 配置示例

### config.yaml 中的相关配置

```yaml
conversation:
  summary:
    prompt: "你是一个智能助手..."  # 系统提示词
    contextTemplate: "{{.History}}\n\n{{.Context}}"  # 上下文模板
    temperature: 0.7
    maxCompletionTokens: 2000
    thinking: false

knowledge_qa:
  fallbackStrategy: "fixed_reply"  # 或 "model_generate"
  defaultRerankTopK: 10
  defaultVectorThreshold: 0.7
  defaultKeywordThreshold: 0.5
```

---

## 错误处理

### 常见错误场景

1. **检索结果为空**：触发降级策略
2. **模型调用失败**：返回错误信息，通过 `EventError` 通知前端
3. **会话不存在**：返回 404 错误
4. **知识库不存在**：返回 400 错误
5. **SSRF 攻击**：清除客户端提供的 URL/Caption

### 错误事件

- `EventError`：通用错误事件
- `ErrSearchNothing`：检索结果为空（内部错误，非致命）

---

## 性能优化

### 并行检索
向量检索和实体检索并行执行，显著减少检索时间。

### 上下文管理
- 对话历史限制（避免超出模型上下文窗口）
- 检索结果 TopK 过滤（减少无关信息）

### 异步处理
- 标题生成异步执行（不阻塞主流程）
- Q&A 对索引异步执行（不阻塞响应）

---

## 总结

KnowledgeQA 是一个设计良好的 RAG 系统，具有以下优势：

1. **模块化**：各处理阶段独立，易于扩展和维护
2. **灵活性**：支持动态管道组装，适应不同场景
3. **实时性**：SSE 流式响应，用户体验好
4. **健壮性**：降级策略、错误处理完善
5. **多模态**：支持图片分析，功能丰富

适用场景：
- 知识库问答
- 文档检索
- 数据分析（简单场景）
- 多模态问答（图片 + 文本）
