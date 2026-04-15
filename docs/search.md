# KnowledgeQA Pipeline Search 流程

## 目录结构

```
internal/application/service/
├── session_knowledge_qa.go       # KnowledgeQA 入口
├── chat_pipeline/
│   ├── chat_pipeline.go         # EventManager 核心
│   ├── common.go               # 公共工具函数
│   ├── search.go               # 核心搜索插件 (CHUNK_SEARCH)
│   ├── search_parallel.go       # 并行搜索插件 (CHUNK_SEARCH_PARALLEL)
│   ├── search_entity.go         # 实体搜索插件 (ENTITY_SEARCH)
│   ├── query_understand.go     # 查询理解插件 (QUERY_UNDERSTAND)
│   ├── query_expansion.go       # 查询扩展
│   ├── rerank.go               # 重排序插件 (CHUNK_RERANK)
│   ├── merge.go                # 合并插件 (CHUNK_MERGE)
│   ├── filter_top_k.go         # TopK 过滤插件
│   ├── web_fetch.go            # 网页内容抓取
│   ├── data_analysis.go        # 数据分析
│   └── chat_completion_stream.go # LLM 流式生成
```

## 入口流程

### 主入口函数

| 函数名 | 位置 | 功能 |
|--------|------|------|
| `KnowledgeQA()` | session_knowledge_qa.go:20-217 | 主入口，处理请求参数、构建 pipeline |
| `KnowledgeQAByEvent()` | session_knowledge_qa.go:484-536 | 事件驱动执行器，遍历 eventList 触发各阶段事件 |
| `SearchKnowledge()` | session_knowledge_qa.go:541-645 | 纯搜索接口，不经过 LLM 生成 |
| `buildSearchTargets()` | session_knowledge_qa.go:387-481 | 解析 KBID 和 KnowledgeID 为统一搜索目标 |

### Pipeline 动态组装 (session_knowledge_qa.go:142-179)

**纯聊天模式**（无 KB、无网络搜索）：
```go
pipeline = types.NewPipelineBuilder()
    .AddIf(hasHistory, types.LOAD_HISTORY)
    .AddIf(chatManage.EnableMemory, types.MEMORY_RETRIEVAL)
    .Add(types.CHAT_COMPLETION_STREAM)
    .AddIf(chatManage.EnableMemory, types.MEMORY_STORAGE)
    .Build()
```

**RAG 模式**（有 KB 或网络搜索）：
```go
pipeline = types.NewPipelineBuilder()
    .Add(types.LOAD_HISTORY)
    .Add(types.QUERY_UNDERSTAND)
    .Add(types.CHUNK_SEARCH_PARALLEL)     // 搜索开始
    .Add(types.CHUNK_RERANK)              // 重排序
    .AddIf(req.WebSearchEnabled, types.WEB_FETCH)
    .Add(types.CHUNK_MERGE)              // 合并、去重
    .Add(types.FILTER_TOP_K)             // TopK 过滤
    .Add(types.DATA_ANALYSIS)
    .Add(types.INTO_CHAT_MESSAGE)         // 组装 LLM prompt
    .Add(types.CHAT_COMPLETION_STREAM)   // LLM 生成
    .Build()
```

## 数据流走向

```
用户 Query
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  1. QUERY_UNDERSTAND (query_understand.go)                 │
│     - 调用 LLM 重写查询 (RewriteQuery)                     │
│     - 意图分类 (Intent)                                    │
│     - 图片描述提取                                        │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  2. CHUNK_SEARCH_PARALLEL (search_parallel.go)           │
│     - 遍历 SearchTargets                                   │
│     - 并行调用 CHUNK_SEARCH                                │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  3. CHUNK_SEARCH (search.go)                               │
│     - 混合搜索: 向量搜索 + 关键词搜索                       │
│     - 结果合并、去重                                       │
│     - [可选] 查询扩展 (recall 不足时触发)                   │
│     - [可选] 网络搜索                                     │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  4. ENTITY_SEARCH (search_entity.go)                       │
│     - 知识图谱实体搜索 (Neo4j)                           │
│     - 图数据合并                                         │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  5. CHUNK_RERANK (rerank.go)                             │
│     - 调用 Rerank 模型                                   │
│     - 分数合成: 0.6*模型分 + 0.3*基础分 + 0.1*来源权重   │
│     - MMR 多样性重排序                                    │
│     - FAQ 分数提升                                       │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  6. WEB_FETCH (web_fetch.go)                              │
│     - 抓取 TopN 网页完整内容                              │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  7. CHUNK_MERGE (merge.go)                               │
│     - 去重 (ID + 内容签名)                               │
│     - 父子 Chunk 合并                                   │
│     - 按 KnowledgeID + ChunkType 分组                      │
│     - 重叠范围合并                                       │
│     - FAQ 答案填充                                       │
│     - 短上下文扩展                                       │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  8. FILTER_TOP_K (filter_top_k.go)                       │
│     - 保留 TopK 结果                                     │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  9. DATA_ANALYSIS (data_analysis.go)                    │
│     - 数据质量分析                                       │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  10. INTO_CHAT_MESSAGE (into_chat_message.go)          │
│     - 组装 LLM prompt (系统提示 + 上下文)               │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│  11. CHAT_COMPLETION_STREAM (chat_completion_stream.go)│
│     - 调用 LLM 生成流式回答                              │
│     - 通过 EventBus 发送 answer 事件                     │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
用户收到流式响应 + 参考文档引用
```

## 搜索核心插件

### 3.1 CHUNK_SEARCH (search.go)

| 函数/方法 | 行号 | 功能 |
|-----------|------|------|
| `PluginSearch` 结构体 | 14-20 | 搜索插件定义，包含 knowledgeBaseService、embeddingService、rerankService |
| `NewPluginSearch()` | 29-51 | 构造函数，注册插件 |
| `OnEvent()` | 60-639 | 主搜索逻辑，处理 CHUNK_SEARCH 事件 |
| `combinedKBSearch()` | 73-155 | 多 KB 并行搜索 |
| `singleKBSearch()` | 157-259 | 单 KB 搜索（向量+关键词混合） |
| `performVectorSearch()` | 261-298 | 向量搜索 |
| `performKeywordSearch()` | 300-362 | 关键词搜索 |
| `handleSearchResults()` | 364-426 | 结果合并与去重 |
| `webSearch()` | 428-620 | 网络搜索（可选） |
| `runQueryExpansion()` | 622-680 | 查询扩展（recall 不足时触发） |
| `performWebSearch()` | 682-752 | 执行网络搜索 |

### 3.2 CHUNK_SEARCH_PARALLEL (search_parallel.go)

| 函数/方法 | 行号 | 功能 |
|-----------|------|------|
| `PluginSearchParallel` 结构体 | 20-27 | 并行搜索插件 |
| `OnEvent()` | 83-175 | 处理 CHUNK_SEARCH_PARALLEL 事件 |
| `parallelKBSearch()` | 177-263 | 多 KB 并行搜索（goroutine） |

### 3.3 ENTITY_SEARCH (search_entity.go)

| 函数/方法 | 行号 | 功能 |
|-----------|------|------|
| `OnEvent()` | 42-193 | 处理 ENTITY_SEARCH 事件 |
| `filterSeenChunk()` | 196-215 | 过滤已见 chunk |
| `chunk2SearchResult()` | 218-239 | Chunk 转 SearchResult |

## 数据结构

### PipelineRequest (不可变配置)

```go
type PipelineRequest struct {
    Query, SessionID, UserID            // 查询信息
    KnowledgeBaseIDs, KnowledgeIDs      // 知识库/文档 ID
    SearchTargets     SearchTargets     // 统一搜索目标
    VectorThreshold   float64           // 向量相似度阈值
    KeywordThreshold  float64           // 关键词匹配阈值
    EmbeddingTopK     int               // 向量搜索 TopK
    RerankTopK       int               // 重排 TopK
    RerankThreshold   float64           // 重排阈值
    EnableRewrite, EnableQueryExpansion // 查询优化开关
    WebSearchEnabled, WebFetchEnabled  // 网络搜索开关
}
```

### PipelineState (可变中间状态)

```go
type PipelineState struct {
    RewriteQuery       string             // 重写后查询
    Intent             QueryIntent        // 意图分类
    History           []*History          // 对话历史
    
    SearchResult      []*SearchResult     // 搜索结果
    RerankResult      []*SearchResult     // 重排结果
    MergeResult       []*SearchResult     // 合并结果
    Entity            []string            // 提取实体
    GraphResult       *GraphData          // 图数据
}
```

### EventType 阶段定义

```go
const (
    LOAD_HISTORY           EventType = "load_history"
    QUERY_UNDERSTAND       EventType = "query_understand"
    CHUNK_SEARCH           EventType = "chunk_search"
    CHUNK_SEARCH_PARALLEL  EventType = "chunk_search_parallel"
    ENTITY_SEARCH          EventType = "entity_search"
    CHUNK_RERANK           EventType = "chunk_rerank"
    WEB_FETCH              EventType = "web_fetch"
    CHUNK_MERGE           EventType = "chunk_merge"
    FILTER_TOP_K          EventType = "filter_top_k"
    DATA_ANALYSIS         EventType = "data_analysis"
    INTO_CHAT_MESSAGE     EventType = "into_chat_message"
    CHAT_COMPLETION_STREAM EventType = "chat_completion_stream"
)
```

## 关键搜索逻辑

### 6.1 混合搜索 (search.go 行 157-259)

`singleKBSearch()` 执行单 KB 混合搜索：
1. 并行执行向量搜索 + 关键词搜索
2. 结果合并
3. 去重处理
4. [可选] 查询扩展 (recall 不足时)

### 6.2 查询扩展 (query_expansion.go)

当 `len(SearchResult) < EmbeddingTopK` 时触发 `runQueryExpansion()`：

```go
// 查询变体生成策略
1. 移除停用词
2. 提取引用短语
3. 分割长句
4. 移除疑���词

// 执行
- 并发执行扩展查询
- 结果合并
```

### 6.3 重排序 (rerank.go)

分数合成公式：
```go
func compositeScore(sr *types.SearchResult, modelScore, baseScore float64) float64 {
    sourceWeight := 1.0  // 来源权重
    if sr.KnowledgeSource == "web_search" { sourceWeight = 0.95 }
    positionPrior := 1.0  // 位置先验
    // 分数合成公式
    return 0.6*modelScore + 0.3*baseScore + 0.1*sourceWeight * positionPrior
}
```

MMR 多样性重排序：
```go
// 公式: MMR = λ*relevance - (1-λ)*redundancy
// 使用 Jaccard 相似度计算冗余
func applyMMR(results []*types.SearchResult, k int, lambda float64)
```

### 6.4 合并策略 (merge.go)

合并流程：
1. 初始去重 (ID + 内容签名)
2. 注入历史引用
3. 解析父子 Chunk
4. 按 KnowledgeID + ChunkType 分组
5. 组内重叠范围合并
6. FAQ 答案填充
7. 短上下文扩展
8. 二次重叠合并
9. 最终去重

## 总结

| 阶段 | 文件 | 核心功能 |
|------|------|----------|
| 查询理解 | query_understand.go | LLM 重写 + 意图分类 |
| 并行搜索 | search_parallel.go | 多 KB 并行调度 |
| 核心搜索 | search.go | 向量+关键词混合搜索、查询扩展 |
| 实体搜索 | search_entity.go | 知识图谱实体检索 |
| 重排序 | rerank.go | 模型重排 + MMR 多样性 |
| 网页抓取 | web_fetch.go | TopN 网页内容获取 |
| 结果合并 | merge.go | 去重、分组、重叠合并 |
| 过滤 | filter_top_k.go | TopK 筛选 |
| LLM 生成 | chat_completion_stream.go | 流式回答生成 |