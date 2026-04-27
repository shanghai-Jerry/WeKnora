# 检索即生成 (Retrieve-then-Generate)

## 概述

检索即生成是一种迭代式检索增强生成（RAG）模式。与传统 RAG（先检索再生成）不同，检索即生成让 LLM **先尝试回答**，根据回答内容决定是否需要检索更多知识。这种模式特别适合**需要多步推理且知识分散在不同文档中**的场景。

**核心思想**：模型自主判断"我已经知道足够回答这个问题"还是"我需要查阅更多资料"。

## 与传统 RAG 的区别

| 特性 | 传统 RAG (quick-answer) | 检索即生成 (retrieve-then-generate) |
|------|--------------------------|--------------------------------------|
| 执行顺序 | 检索 → 生成 | 生成 → 判断 → 检索（可选）→ 生成 |
| 迭代次数 | 单次 | 多次（默认 3 轮，可配置） |
| 检索时机 | 问题入口即检索 | LLM 主动请求时检索 |
| 上下文来源 | 初始检索的全部结果 | 逐轮累积，仅保留相关分片 |
| 适用场景 | 简单问答 | 复杂推理、需要综合多份文档 |

## 设计架构

### 三种智能体模式

```
                    ┌─────────────────────────┐
                    │      CustomAgent        │
                    └───────────┬─────────────┘
                                │
           ┌────────────────────┼────────────────────┐
           │                    │                    │
           ▼                    ▼                    ▼
    ┌──────────────┐    ┌──────────────────┐   ┌──────────────────┐
    │ quick-answer │    │ smart-reasoning  │   │ retrieve-then-   │
    │   (RAG)      │    │   (ReAct)        │   │ generate         │
    └──────────────┘    └──────────────────┘   └──────────────────┘
```

- **quick-answer**：传统 RAG pipeline，检索→重排序→生成，一次完成
- **smart-reasoning**：ReAct Agent 模式，通过工具调用自主探索
- **retrieve-then-generate**：本文档描述的模式

### Pipeline 架构

```
用户问题
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│                   LOAD_HISTORY                          │
│              加载多轮对话历史                             │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                     RAG_ITERATE                          │
│    ┌──────────────────────────────────────────────┐     │
│    │  for round in 1..max_rounds:                 │     │
│    │    1. 构建 Prompt（含问题+中间推理+参考文本）│     │
│    │    2. 调用 LLM，获取 JSON 格式回复            │     │
│    │    3. 解析 Action: "answer" | "retrieve"     │     │
│    │    4. 如果 answer → 完成，输出最终答案        │     │
│    │    5. 如果 retrieve → HybridSearch + Rerank  │     │
│    │    6. 累积参考文本，进入下一轮                │     │
│    └──────────────────────────────────────────────┘     │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
              发送 final_answer 事件（流式）
              发送 references 事件（累积的所有分片引用）
```

### LLM 回复格式

LLM 必须以 JSON 格式回复，由系统解析决定下一步操作：

```json
// 情况1：模型认为已掌握足够信息
{"action": "answer", "content": "完整答案内容"}

// 情况2：模型认为需要检索更多内容
{"action": "retrieve", "content": "当前推理过程...", "query": "待检索的查询语句"}
```

**与 Python 原实现的区别**：

| 特性 | Python 原实现 | Go 实现 |
|------|--------------|---------|
| 解析方式 | 标签解析 `[ANSWER]...[SOLVED]` | JSON 结构化输出 |
| 检索方式 | 固定 Elasticsearch | 复用 RetrieveEngine（支持 Postgres/ES/Qdrant/Milvus 等） |
| 重排序 | 无（固定 BM25 top-3） | 完整 Reranker 流程（清洗→批量打分→阈值过滤→MMR） |
| 上下文累积 | 替换式（每轮覆盖） | 累积式（所有轮次去重合并） |

## 后端实现

### 类型定义

**`internal/types/custom_agent.go`**

```go
// AgentMode 常量
const (
    AgentModeQuickAnswer             = "quick-answer"
    AgentModeSmartReasoning          = "smart-reasoning"
    AgentModeRetrieveThenGenerate    = "retrieve-then-generate"  // 新增
)

// CustomAgentConfig 新增字段
type CustomAgentConfig struct {
    // 检索即生成配置
    RAGMaxRounds        int    // 最大迭代轮数，默认 3
    RAGRetrievalPrompt  string // 自定义 system prompt（可选）
    // ... 其他字段
}

// 内置智能体 ID
const BuiltinRetrieveThenGenerateID = "builtin-retrieve-then-generate"
```

**`internal/types/chat_manage.go`**

```go
// PipelineRequest 新增字段
type PipelineRequest struct {
    EnableRetrieveThenGenerate bool   // 是否启用检索即生成
    RAGMaxRounds              int    // 最大迭代轮数
    RAGRetrievalPrompt        string // 自定义 prompt
    // ... 其他字段
}

// PipelineState 新增字段
type PipelineState struct {
    RAGIterationState *RAGIterationState // 迭代状态
    // ... 其他字段
}

// 迭代状态结构
type RAGIterationState struct {
    CurrentRound    int                     // 当前轮次
    MaxRounds       int                     // 最大轮次
    IsCompleted     bool                    // 是否完成
    FinalAnswer     string                  // 最终答案
    Intermediary    string                  // 中间推理
    ReferenceText   string                  // 累积的参考文本
    AllReferences   []*SearchResult         // 所有引用的分片
    IterationSteps  []RAGIterationStep      // 每轮详情（前端展示用）
}

type RAGIterationStep struct {
    Round           int             // 轮次
    LLMAction       string          // "answer" | "retrieve"
    Content         string          // 回答内容或中间推理
    RetrieveQuery   string          // 检索查询
    RetrievedChunks []*SearchResult // 本轮检索结果
}
```

### 事件类型

**`internal/event/event.go`**

```go
// RAG 迭代事件
const EventRAGIteration EventType = "rag_iteration"
```

**`internal/event/event_data.go`**

```go
type RAGIterationData struct {
    Round         int    `json:"round"`
    Action        string `json:"action"`         // "answer" | "retrieve"
    Content       string `json:"content"`        // 回答内容或中间推理
    RetrieveQuery string `json:"retrieve_query"` // 检索查询
    ChunkCount    int    `json:"chunk_count"`    // 本轮检索到的分片数
    Done          bool   `json:"done"`
}
```

### Pipeline 插件 (`rag_iterate.go`)

核心插件 `PluginRAGIterate` 实现了完整的迭代循环：

1. **Prompt 构建**：问题 + 当前中间推理 + 累积的参考文本
2. **LLM 调用**：非流式调用，获取完整 JSON 回复
3. **响应解析**：解析 `action` 字段判断下一步
4. **检索流程**：调用 `HybridSearch` + `Rerank`
5. **引用累积**：去重后追加到 `AllReferences`
6. **事件发射**：每轮发出 `EventRAGIteration`，完成后发出 `final_answer`

**检索复用说明**：
- 使用 `knowledgeBaseService.HybridSearch()` 进行检索，支持多 KB 并行
- 使用 `modelService.GetRerankModel()` 获取重排模型
- 复用 `cleanPassageForRerank` 和 `getEnrichedPassage` 进行文本清洗
- 支持所有后端检索引擎（Postgres/ES/Qdrant/Milvus/Weaviate 等）

### Pipeline 组装

**`internal/application/service/session_knowledge_qa.go`**

```go
if chatManage.EnableRetrieveThenGenerate && needsRAG {
    pipeline = types.NewPipelineBuilder().
        AddIf(hasHistory, types.LOAD_HISTORY).
        Add(types.RAG_ITERATE).
        Build()
} else if !needsRAG {
    // 纯聊天
    ...
} else {
    // 传统 RAG
    ...
}
```

### 智能体覆盖应用

**`internal/application/service/session_qa_helpers.go`**

```go
if customAgent.Config.AgentMode == types.AgentModeRetrieveThenGenerate {
    cm.EnableRetrieveThenGenerate = true
    if customAgent.Config.RAGMaxRounds > 0 {
        cm.RAGMaxRounds = customAgent.Config.RAGMaxRounds
    } else {
        cm.RAGMaxRounds = 3
    }
    cm.RAGRetrievalPrompt = customAgent.Config.RAGRetrievalPrompt
}
```

### 内置智能体

**`config/builtin_agents.yaml`**

```yaml
- id: "builtin-retrieve-then-generate"
  avatar: "🔄"
  is_builtin: true
  i18n:
    default:
      name: "Retrieve then Generate"
      description: "Iterative retrieval-augmented generation..."
  config:
    agent_mode: "retrieve-then-generate"
    temperature: 0.7
    max_completion_tokens: 4096
    rag_max_rounds: 3
    kb_selection_mode: "all"
    # ... 其他配置
```

### SSE 事件流

检索即生成模式发送的 SSE 事件序列：

| 事件类型 | 内容 | 说明 |
|---------|------|------|
| `rag_iteration` | `{round, action, content, retrieve_query}` | 每轮 LLM 回复 |
| `final_answer` | 答案文本流 | 最终答案（流式） |
| `references` | `SearchResult[]` | 累积的所有引用分片 |

## 前端实现

### 智能体类型定义

**`frontend/src/api/agent/index.ts`**

```ts
export const AGENT_MODE_RETRIEVE_THEN_GENERATE = 'retrieve-then-generate';
export const BUILTIN_RETRIEVE_THEN_GENERATE_ID = 'builtin-retrieve-then-generate';

interface CustomAgentConfig {
    // ... 其他字段
    rag_max_rounds?: number;        // 最大迭代轮数
    rag_retrieval_prompt?: string;  // 自定义 prompt
}
```

### 智能体编辑器

**`frontend/src/views/agent/AgentEditorModal.vue`**

- Agent Mode 单选组新增第三个选项：`retrieve-then-generate`
- 当选择该模式时显示专属配置区：
  - **最大迭代轮数** (InputNumber, 1-6, 默认 3)
  - 复用现有检索策略配置（embedding_top_k, rerank_top_k 等）
  - 复用知识库选择配置
  - 不显示：Query Expansion、Query Intent Explore、Context Template

### 聊天界面展示

**`frontend/src/views/chat/components/RAGIterationDisplay.vue`**（新建）

折叠式时间线展示每轮迭代：

```
┌─ 第1轮 检索 ─────────────────────────┐
│  检索查询: "xxx"                     │
│  检索结果: 3 个分片                   │
│  ▶ 展开查看详情                       │
└─────────────────────────────────────┘
┌─ 第2轮 回答 ✓ ───────────────────────┐
│  [展开] 中间推理内容                   │
└─────────────────────────────────────┘
┌─ 最终答案 ───────────────────────────┐
│  这是最终的完整回答...                │
│  [1] 来源: 文档A                     │
│  [2] 来源: 文档B                     │
└─────────────────────────────────────┘
```

## 配置选项

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `agent_mode` | string | - | 必须是 `"retrieve-then-generate"` |
| `rag_max_rounds` | int | 3 | 最大迭代轮数，建议 1-6 |
| `rag_retrieval_prompt` | string | (默认 prompt) | 自定义 system prompt |
| `temperature` | float | 0.7 | LLM 温度 |
| `rerank_top_k` | int | 5 | 每轮检索后重排保留数量 |
| `rerank_threshold` | float | 0.3 | 重排分数阈值 |
| `kb_selection_mode` | string | "all" | 知识库选择模式 |
| `multi_turn_enabled` | bool | true | 自动启用多轮对话 |

## 使用示例

### 通过 API 创建检索即生成智能体

```bash
curl -X POST /api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "我的检索即生成助手",
    "config": {
      "agent_mode": "retrieve-then-generate",
      "rag_max_rounds": 3,
      "temperature": 0.7,
      "kb_selection_mode": "selected",
      "knowledge_bases": ["kb-uuid-1", "kb-uuid-2"]
    }
  }'
```

### 通过聊天接口使用

```bash
curl -X POST /api/v1/knowledge-chat/{session_id} \
  -H "Content-Type: application/json" \
  -d '{
    "query": "帮我综合分析这份报告中提到的所有风险因素",
    "agent_id": "agent-uuid-or-builtin-id"
  }'
```

## 适用场景

**适合使用检索即生成的场景**：

- 需要综合多份文档才能回答的复杂问题
- 答案分散在不同章节或文件中
- 需要 LLM 先理解问题再决定查阅哪些资料
- 传统 RAG 容易遗漏关键信息的情况

**不适合使用检索即生成的场景**：

- 简单的事实问答（直接用 quick-answer）
- 需要精确匹配的问题（FAQ 更合适）
- 模型无法准确判断何时需要检索的场景

## 参考资料

- Python 原型实现：`python-scripts/agent/agent-api.py`
- 论文：Retrieval as Generation: A Unified Framework with Self-Triggered Information Planning