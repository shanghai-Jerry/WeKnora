# 文档解析处理流程详解

本文档详细描述了用户上传文档到知识库后，从文档解析、分块、问题生成、摘要总结、向量化到入库存储的完整流程。

## 整体流程图

```
用户上传文件
    ↓
[API层] 创建知识记录
    ↓
[Asynq任务] 异步文档处理
    ↓
[文档解析] 调用DocReader服务
    ↓
[图片处理] 解析和存储图片
    ↓
[文本分块] 按配置分块
    ↓
┌───────────┬───────────────┬──────────────┐
│           │               │              │
↓           ↓               ↓              ↓
[向量化]  [问题生成]    [摘要生成]   [多模态处理]
(同步)    (异步)        (异步)      (异步)
│           │               │              │
└───────────┴───────────────┴──────────────┘
    ↓
[入库存储] 保存分块和向量索引
    ↓
[状态更新] 完成处理
```

## 1. 文档上传（API层）

### 入口位置
- **文件**: `internal/handler/knowledge.go`
- **函数**: `CreateKnowledgeFromFile` (第205行)
- **URL**: `POST /api/v1/knowledge-bases/{id}/knowledge/file`

### 处理流程

```go
// 1. 验证知识库访问权限 (第210行)
// 2. 获取上传文件 (第224行)
// 3. 验证文件大小 (第231-237行)
// 4. 解析元数据 (第253-262行)
// 5. 调用服务层创建知识条目 (第286行)
```

### 服务层逻辑
- **文件**: `internal/application/service/knowledge.go`
- **函数**: `CreateKnowledgeFromFile` (第189行)

**核心步骤**:
1. **验证配置**: 检查多模态/ASR配置 (第216-269行)
2. **验证文件类型**: 确保支持的文件格式 (第273行)
3. **计算文件哈希**: 用于去重 (第280行)
4. **检查存储配额**: 防止超出限制 (第311行)
5. **创建知识记录**: 状态设为 `"pending"` (第336-353行)
6. **保存文件**: 存储到配置的存储系统 (第362行)
7. **入队任务**: 创建Asynq异步处理任务 (第416-417行)

```go
// 入队Asynq任务
task := tasks.NewDocumentProcessTask(knowledge.ID, knowledge.KnowledgeBaseID)
_, err = s.taskClient.Enqueue(task)
```

## 2. 文档解析（DocReader服务）

### Asynq任务入口
- **文件**: `internal/application/service/knowledge.go`
- **函数**: `ProcessDocument` (第7569行)
- **任务类型**: `TypeDocumentProcess`

### 解析流程核心函数
- **函数**: `convert` (第7945行)

```go
// Step 1: 确定解析引擎 (第7967-7970行)
// 引擎类型: "simple", "mineru", "mineru_cloud", "builtin"
```

### 解析器选择逻辑
- **函数**: `resolveDocReader` (第8023-8041行)

| 解析器 | 适用场景 | 实现位置 |
|--------|----------|----------|
| `SimpleFormatReader` | 简单格式（txt, md等） | HTTP解析器 |
| `MinerUReader` | 复杂PDF解析（本地） | 本地MinerU |
| `MinerUCloudReader` | 云端MinerU服务 | HTTP API |
| `GRPCDocumentReader` | gRPC调用docreader服务 | `internal/infrastructure/docparser/grpc_parser.go` (第105行) |

### gRPC解析器实现
- **文件**: `internal/infrastructure/docparser/grpc_parser.go`
- **函数**: `Read` (第105行)
- **功能**: 通过gRPC调用docreader服务（端口50051）

**返回结果**:
```go
type ReadResult struct {
    MarkdownContent string           // 解析后的Markdown内容
    Images         []ImageInfo      // 提取的图片信息
    Metadata       map[string]any   // 文档元数据
}
```

## 3. 图片处理

### 处理位置
- **文件**: `internal/application/service/knowledge.go`
- **位置**: `ProcessDocument` 函数内 (第7851-7879行)

### 处理流程

```go
// Step 2: Store images and update markdown references
if s.imageResolver != nil && convertResult != nil {
    // 解析和存储图片 (第7857行)
    updatedMarkdown, images, _ := s.imageResolver.ResolveAndStore(...)
    
    // 解析远程HTTP图片 (第7868行)
    updatedContent, remoteImages, _ := s.imageResolver.ResolveRemoteImages(...)
}
```

### 图片解析器
- **文件**: `internal/infrastructure/docparser/image_resolver.go`
- **功能**: 提取、存储图片，更新Markdown中的图片引用

## 4. 文本分块（Chunking）

### 分块位置
- **文件**: `internal/application/service/knowledge.go`
- **位置**: `ProcessDocument` 函数内 (第7881-7936行)

### 分块配置
来自知识库的 `ChunkingConfig`:
- **ChunkSize**: 块大小（默认512字符）
- **ChunkOverlap**: 重叠大小（默认128字符）
- **Separators**: 分隔符（默认 `["\n\n", "\n", "。"]`）
- **EnableParentChild**: 是否启用父子分块模式

### 分块实现
- **文件**: `internal/infrastructure/chunker/splitter.go`

#### 普通分块
- **函数**: `SplitText` (第143行)

**特性**:
1. **保护模式**: 防止特殊内容被分割（第46-54行）
   - LaTeX公式
   - 表格
   - 代码块
2. **递归分割**: 按分隔符递归（第109行 `splitBySeparators`）
3. **合并单元**: 带重叠合并（第263行 `mergeUnits`）
4. **最大块限制**: 绝对最大7500字符（第268行）

#### 父子分块
- **函数**: `SplitTextParentChild` (第516行)

**流程**:
1. 先分成大父块（使用 `ParentChunkSize`）
2. 再将每个父块分成小子块（使用 `ChildChunkSize`）
3. 子块携带 `ParentIndex` 引用父块

**优势**: 检索时使用子块，展示时使用父块，提高准确性。

## 5. 问题生成（Question Generation）

### 触发条件
知识库配置了 `QuestionGenerationConfig.Enabled = true`

### 配置结构
- **文件**: `internal/types/knowledgebase.go`
- **位置**: 第348行 `QuestionGenerationConfig`

```go
type QuestionGenerationConfig struct {
    Enabled       bool  // 是否启用
    QuestionCount int   // 每个块生成的问题数量（默认3，最大10）
}
```

### 任务入队
- **位置**: `processChunks` 函数内 (第1897-1907行)

```go
if options.EnableQuestionGeneration && len(textChunks) > 0 && !isImage {
    s.enqueueQuestionGenerationTask(ctx, knowledge.KnowledgeBaseID, knowledge.ID, questionCount)
}
```

### 任务处理
- **函数**: `ProcessQuestionGeneration` (第2339行)
- **核心逻辑**: `generateQuestionsWithContext` (第2518-2587行)

### 生成流程

```go
// 1. 获取聊天模型 (第2403行)
chatModel, err := s.modelService.GetChatModel(ctx, kb.SummaryModelID)

// 2. 对每个文本块生成问题 (第2439-2504行)
for _, chunk := range textChunks {
    // 构建上下文（前一个和后一个块的内容）
    context := buildContext(chunk, prevChunk, nextChunk)
    
    // 调用LLM生成问题
    questions := generateQuestions(chunk.Content, context)
}

// 3. 使用提示词模板 (第2526行)
prompt = s.config.Conversation.GenerateQuestionsPrompt

// 4. 解析生成的问题（每行一个）(第2569-2584行)

// 5. 更新chunk元数据 (第2476-2482行)
chunk.SetDocumentMetadata(meta) // 包含生成的问题

// 6. 索引问题 (第2508行)
retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfoList)
// 问题也被向量化，用于语义检索
```

### 提示词模板
配置在 `config.yaml` 中的 `conversation.generate_questions_prompt`

## 6. 摘要生成（Summary Generation）

### 触发条件
有文本块且非图片类型

### 任务入队
- **位置**: `processChunks` 函数内 (第1909-1912行)

```go
if len(textChunks) > 0 && !isImage {
    s.enqueueSummaryGenerationTask(ctx, knowledge.KnowledgeBaseID, knowledge.ID)
}
```

### 任务处理
- **函数**: `ProcessSummaryGeneration` (第2150行)
- **核心逻辑**: `getSummary` (第1930行)

### 摘要生成流程

```go
// 1. 确定最大输入字符数 (第1940行)
maxInputChars = 16384 // 默认
// 可从配置覆盖: s.config.Conversation.Summary.MaxInputChars

// 2. 按StartAt排序所有文本块 (第1950-1954行)

// 3. 拼接所有块内容 (第1957-1965行)
// 注意：这里拼接的是完整内容，不是采样

// 4. 移除Markdown图片语法 (第1973-1975行)

// 5. 添加图片注释（如果有）(第1977-1991行)
// Image Caption 和 OCR Text

// 6. 对长内容采样 (第1994行) - 关键处理
chunkContents = sampleLongContent(chunkContents, maxInputChars)
// 采样策略：保留开头、中间、结尾的关键部分

// 7. 调用LLM生成摘要 (第2026-2029行)
summary, err := summaryModel.Chat(ctx, []chat.Message{
    {Role: "system", Content: summaryPrompt},
    {Role: "user", Content: contentWithMetadata},
})
```

### 长文档处理策略（sampleLongContent）

当文档内容超过 `maxInputChars`（默认16384字符）时，采用采样策略：

1. **保留开头**: 前 20% 的内容
2. **保留结尾**: 后 20% 的内容
3. **保留中间**: 从中间部分均匀采样 60% 的内容

这种策略确保摘要生成时能够覆盖文档的关键部分，同时不超过模型的输入限制。

### 摘要后处理
- **位置**: `ProcessSummaryGeneration` 函数内 (第2256-2321行)

```go
// 1. 更新知识描述
knowledge.Description = summary
knowledge.SummaryStatus = types.SummaryStatusCompleted

// 2. 创建摘要块 (第2275-2287行)
summaryChunk := &types.Chunk{
    ChunkType: types.ChunkTypeSummary,  // 特殊类型
    Content:   fmt.Sprintf("# Document\n%s\n\n# Summary\n%s", ...),
    ParentChunkID: textChunks[0].ID,  // 关联到第一个文本块
}

// 3. 保存摘要块并索引 (第2292-2321行)
```

### 摘要状态
- **文件**: `internal/types/knowledge.go`

```go
SummaryStatusNone      = "none"      // 无摘要
SummaryStatusPending   = "pending"   // 等待生成
SummaryStatusProcessing = "processing" // 生成中
SummaryStatusCompleted = "completed"  // 已完成
SummaryStatusFailed   = "failed"    // 生成失败
```

## 7. 向量化（Embedding）

### 处理位置
- **函数**: `processChunks` (第1512行开始)

### 向量化流程

```go
// Step: 获取嵌入模型 (第1541行)
embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)

// 准备索引信息 (第1744-1760行)
for _, chunk := range textChunks {
    indexContent := titlePrefix + chunk.Content  // 前缀文档标题
    indexInfoList = append(indexInfoList, &types.IndexInfo{
        Content:    indexContent,
        SourceID:   chunk.ID,
        ChunkID:    chunk.ID,
        // ...
    })
}

// 批量向量化并索引 (第1819行)
err = retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfoList)
```

### 复合检索引擎
- **文件**: `internal/application/service/retriever/composite.go`
- **函数**: `BatchIndex` (第221行)

**功能**: 批量索引到所有注册的向量存储

### 支持的向量数据库
位于 `internal/application/repository/retriever/`:

| 向量数据库 | 实现文件 |
|-----------|----------|
| Qdrant | `qdrant/repository.go` |
| Milvus | `milvus/repository.go` |
| Elasticsearch | `elasticsearch/v7/` 或 `v8/` |
| PostgreSQL (pgvector) | `postgres/repository.go` |
| Weaviate | `weaviate/repository.go` |
| Neo4j (知识图谱) | `neo4j/repository.go` |

### 接口定义
- **文件**: `internal/types/interfaces/retriever.go`
- **函数**: `BatchIndex` (第87行)

## 8. 入库存储

### 分块存储
- **位置**: `processChunks` 函数内 (第1798行)

```go
// 保存chunks到数据库
if err := s.chunkService.CreateChunks(ctx, insertChunks); err != nil {
    // 失败后清理向量索引
}
```

### 向量索引存储
- **位置**: (第1819行)

```go
// 批量向量化并索引
err = retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfoList)
```

### 知识状态更新
- **位置**: (第1876-1895行)

```go
knowledge.ParseStatus = types.ParseStatusCompleted  // "completed"
knowledge.EnableStatus = "enabled"
knowledge.StorageSize = totalStorageSize
knowledge.ProcessedAt = &now
knowledge.SummaryStatus = types.SummaryStatusPending  // 或 Completed

s.repo.UpdateKnowledge(ctx, knowledge)
```

### 存储配额更新
- **位置**: (第1920-1923行)

```go
tenantInfo.StorageUsed += totalStorageSize
s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, totalStorageSize)
```

## 关键配置说明

### 1. 分块配置
`knowledge_bases` 表的 `chunking_config` 字段:
```json
{
  "chunk_size": 512,
  "chunk_overlap": 128,
  "separators": ["\n\n", "\n", "。"],
  "enable_parent_child": false,
  "parent_chunk_size": 2048,
  "child_chunk_size": 512
}
```

### 2. 问题生成配置
`knowledge_bases` 表的 `question_generation_config` 字段:
```json
{
  "enabled": true,
  "question_count": 3
}
```

### 3. 模型配置
- **摘要模型**: `knowledge_bases` 表的 `summary_model_id` 字段
- **嵌入模型**: `knowledge_bases` 表的 `embedding_model_id` 字段

### 4. 提示词模板
`config.yaml` 中的 `conversation` 配置:
- `generate_questions_prompt`: 问题生成提示词
- `generate_summary_prompt`: 摘要生成提示词
- `summary.max_input_chars`: 长文档采样阈值（默认16384）

## 异步任务汇总

| 任务类型 | 任务函数 | 说明 |
|---------|---------|------|
| 文档处理 | `ProcessDocument` | 主任务：解析、分块、向量化 |
| 问题生成 | `ProcessQuestionGeneration` | 异步生成问题 |
| 摘要生成 | `ProcessSummaryGeneration` | 异步生成摘要 |
| 多模态处理 | `ProcessImageMultimodal` | 异步处理图片（如果有） |

## 状态流转

```
上传 → pending → processing → completed
                    ↓
                  failed
```

- **ParseStatus**: none → pending → processing → completed/failed
- **SummaryStatus**: none → pending → processing → completed/failed
- **EnableStatus**: disabled → enabled (处理完成后)

## 错误处理

1. **解析失败**: 更新 `ParseStatus` 为 `failed`，记录错误信息
2. **向量化失败**: 回滚已保存的chunks，更新状态为 `failed`
3. **图片处理失败**: 记录警告，继续处理文本
4. **配额超限**: 拒绝上传，返回错误

## 相关文件索引

| 功能模块 | 文件路径 |
|---------|---------|
| API处理 | `internal/handler/knowledge.go` |
| 服务层 | `internal/application/service/knowledge.go` |
| 分块实现 | `internal/infrastructure/chunker/splitter.go` |
| gRPC解析器 | `internal/infrastructure/docparser/grpc_parser.go` |
| HTTP解析器 | `internal/infrastructure/docparser/http_parser.go` |
| 图片解析 | `internal/infrastructure/docparser/image_resolver.go` |
| 复合检索 | `internal/application/service/retriever/composite.go` |
| 类型定义 | `internal/types/knowledge.go`, `internal/types/chunk.go` |
| 接口定义 | `internal/types/interfaces/retriever.go` |
| 向量存储 | `internal/application/repository/retriever/` |
