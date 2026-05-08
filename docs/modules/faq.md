# FAQ 知识库处理流程与问题排查

## 概述

本文档说明知识库类型为"问答(FAQ)"时，在不同智能体模式下的处理流程差异，以及常见问题不命中的原因和排查方法。

---

## 知识库类型定义

**文件：** `internal/types/knowledgebase.go:12-16`

```go
const (
    // KnowledgeBaseTypeDocument represents the document knowledge base type
    KnowledgeBaseTypeDocument = "document"
    KnowledgeBaseTypeFAQ      = "faq"
)
```

FAQ 知识库的特点：
- 存储 Q&A 对（标准问题 + 答案 + 相似问题）
- Chunk 类型为 `ChunkTypeFAQ`（`internal/types/chunk.go:30`）
- 索引时使用 `KnowledgeTypeFAQ` 标识

---

## 不同智能体处理流程差异

### 1. KnowledgeQA（快速问答模式）

**核心文件：**
- `internal/application/service/session_knowledge_qa.go` - 服务层
- `internal/application/service/chat_pipeline/` - 管道插件
- `internal/application/service/knowledgebase_search.go` - 搜索逻辑

**处理特点：**
- 使用**静态 RAG 管道**，预定义处理阶段顺序执行
- 管道阶段：LOAD_HISTORY → QUERY_UNDERSTAND → CHUNK_SEARCH_PARALLEL → CHUNK_RERANK → ...

**FAQ 特殊处理：**

1. **跳过关键词检索**（`knowledgebase_search.go:268-270`）
   ```go
   // Add keyword retrieval params if supported and not FAQ
   if retrieveEngine.SupportRetriever(types.KeywordsRetrieverType) && 
      !params.DisableKeywordsMatch &&
      kb.Type != types.KnowledgeBaseTypeFAQ {  // FAQ 类型跳过
       // ... 不添加关键词检索参数
   }
   ```

2. **向量检索使用 FAQ 索引**（`knowledgebase_search.go:259-262`）
   ```go
   // For FAQ knowledge base, use FAQ index
   if kb.Type == types.KnowledgeBaseTypeFAQ {
       vectorParams.KnowledgeType = types.KnowledgeTypeFAQ
   }
   ```

3. **跳过 Rerank**（`internal/agent/tools/knowledge_search.go:594-596`）
   ```go
   // Skip reranking for FAQ results (they are explicitly matched Q&A pairs)
   if result.KnowledgeBaseType == types.KnowledgeBaseTypeFAQ {
       faqResults = append(faqResults, result)
   } else {
       rerankCandidates = append(rerankCandidates, result)
   }
   ```

4. **FAQ 后处理**（`knowledgebase_search.go:165-166`）
   ```go
   // FAQ-specific post-processing: iterative retrieval or negative question filtering
   deduplicatedChunks = s.applyFAQPostProcessing(ctx, kb, deduplicatedChunks, 
                                                   vectorResults, retrieveEngine, 
                                                   retrieveParams, params, matchCount)
   ```

---

### 2. AgentQA（智能体模式）

**核心文件：**
- `internal/application/service/session_agent_qa.go` - 服务层
- `internal/agent/engine.go` - Agent 引擎（ReAct 循环）
- `internal/agent/tools/knowledge_search.go` - 知识搜索工具

**处理特点：**
- 使用 **ReAct 模式**（Reasoning + Acting + Observation）
- LLM 动态决策工具调用，支持多轮推理
- 通过 `KnowledgeSearchTool` 工具主动搜索知识库

**FAQ 处理流程：**

1. **工具注册**：Agent 引擎根据配置注册 `KnowledgeSearchTool`
2. **LLM 决策**：LLM 根据上下文决定是否调用搜索工具
3. **执行搜索**：工具内部调用 `HybridSearch`，同样识别 FAQ 类型
4. **结果处理**：FAQ 结果同样跳过 reranking

**与 KnowledgeQA 的关键差异：**

| 维度 | KnowledgeQA | AgentQA |
|------|-------------|---------|
| 决策方式 | 静态管道，自动执行 | LLM 动态决策，可能多次搜索 |
| 搜索触发 | 管道阶段自动触发 | LLM 调用工具时触发 |
| 搜索次数 | 1 次（并行检索） | 可能多次（迭代推理） |
| 上下文利用 | 使用历史消息 | LLM 可主动利用中间结果 |
| 工具生态 | 无 | 可使用 15+ 工具 + Skills |

---

## 不命中知识库答案的常见原因

### 1. 阈值设置过高

**默认阈值：**
```go
// internal/application/service/knowledge.go:5701-5702
if req.VectorThreshold <= 0 {
    req.VectorThreshold = 0.7  // 默认 0.7
}
```

**影响：**
- FAQ 完全依赖向量相似度匹配
- 用户问法与标准问题/相似问题差异大时，相似度可能低于 0.7
- 导致结果被过滤

**解决方案：**
- 降低 `vector_threshold` 到 0.5 或更低
- 在智能体配置中设置 `vector_threshold`
- 为 FAQ 知识库单独配置较低的阈值

---

### 2. FAQ 只依赖向量检索

**代码逻辑：**
```go
// knowledgebase_search.go:268-270
if kb.Type != types.KnowledgeBaseTypeFAQ {
    // 添加关键词检索参数
}
// FAQ 类型不会添加关键词检索
```

**影响：**
- 完全依赖向量相似度，无法利用关键词匹配
- 如果用户问题包含标准问题中的关键词但表述不同，可能不匹配

**解决方案：**
- 为 FAQ 条目生成足够多的**相似问题**（`faq.similar_questions`）
- 使用更好的 embedding 模型提高语义匹配能力
- 考虑在 FAQ 场景下也启用关键词检索（需修改代码）

---

### 3. Embedding 质量或模型不一致

**可能问题：**
- FAQ 索引时使用的 embedding 模型与查询时不一致
- Embedding 模型质量差，语义表示能力不足
- FAQ 条目未正确向量化

**检查代码：**
```go
// knowledgebase_search.go:20-40
func (s *knowledgeBaseService) GetQueryEmbedding(ctx context.Context, 
                                                kbID string, 
                                                queryText string) ([]float32, error) {
    kb, err := s.repo.GetKnowledgeBaseByID(ctx, kbID)
    // 使用知识库的 embedding_model_id 获取模型
    embeddingModel, err = s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
    return embeddingModel.Embed(ctx, queryText)
}
```

**排查步骤：**
1. 检查 FAQ 知识库的 `embedding_model_id` 配置
2. 确认查询时使用的是同一个模型
3. 检查 `chunks` 表中 FAQ 条目是否有向量数据（`embedding` 字段非空）
4. 测试 embedding 模型的质量

---

### 4. FAQ 后处理过滤

**相关代码：**
- `knowledgebase_search_faq.go` - FAQ 后处理逻辑
- `applyFAQPostProcessing` 函数

**可能的过滤逻辑：**
- 负样本问题过滤（排除不应该匹配的问题）
- 迭代检索（多次检索优化结果）
- 基于标签的优先级过滤

**排查方法：**
```bash
# 查看日志中的 FAQ 后处理信息
grep -i "faq\|applyFAQPostProcessing" logs/*.log
```

---

### 5. 搜索范围限制

**常见问题：**
- 请求未正确传递 `knowledge_base_ids` 或 `knowledge_ids`
- FAQ 条目未正确关联到知识库
- 搜索目标构建错误

**检查代码：**
```go
// types/search.go:158-161
type SearchParams struct {
    // KnowledgeBaseIDs overrides the single KB ID
    KnowledgeBaseIDs []string `json:"knowledge_base_ids,omitempty"`
    KnowledgeIDs     []string `json:"knowledge_ids"`
}
```

**排查步骤：**
1. 检查请求参数是否包含正确的知识库 ID
2. 查看日志中的搜索参数：
   ```
   Hybrid search parameters, knowledge base IDs: [kb-xxx], query text: ...
   ```
3. 确认 FAQ 条目在数据库中状态正常（`chunks` 表 `chunk_type = 'faq'`）

---

### 6. Rerank 阈值过滤

**代码逻辑：**
```go
// chat_pipeline/rerank.go:116-130
if originalThreshold > 0.3 {
    degradedThreshold := originalThreshold * 0.7
    // 尝试降低阈值重新 rerank
}
```

**注意：** FAQ 结果默认跳过 reranking，但其他类型结果会经过 rerank。

---

## 调试方法与工具

### 1. 查看日志

**关键日志：**
```bash
# 查看混合搜索参数
grep "HybridSearch\|search parameters" <log_file>

# 查看检索结果
grep "vectorRetrieval\|keywordRetrieval" <log_file>

# 查看 FAQ 相关处理
grep -i "faq\|FAQ" <log_file>
```

**日志示例：**
```
INFO [HybridSearch] | Hybrid search parameters, knowledge base IDs: [kb-00001], query text: 如何退款
INFO [HybridSearch] | vectorRetrieval | index=0 chunk_id=xxx score=0.6542 match_type=vector
```

---

### 2. 数据库查询

**检查 FAQ 条目：**
```sql
-- 查看 FAQ 知识库
SELECT id, name, type, embedding_model_id 
FROM knowledge_bases 
WHERE type = 'faq' AND id = 'kb-xxx';

-- 查看 FAQ chunks
SELECT chunk_id, knowledge_id, chunk_type, content, 
       faq_standard_question, faq_answers, embedding IS NOT NULL as has_embedding
FROM chunks 
WHERE knowledge_base_id = 'kb-xxx' AND chunk_type = 'faq'
LIMIT 10;

-- 查看相似问题
SELECT chunk_id, json_extract(chunk_metadata, '$.similar_questions') as similar_questions
FROM chunks 
WHERE knowledge_base_id = 'kb-xxx' AND chunk_type = 'faq';
```

---

### 3. 手动测试搜索接口

**使用 Swagger/API 测试：**
```bash
POST /api/knowledge-bases/{id}/hybrid-search
Content-Type: application/json

{
  "query_text": "用户问题",
  "vector_threshold": 0.5,
  "match_count": 10,
  "knowledge_type": "faq"
}
```

---

### 4. 检查 Embedding 模型

**验证模型配置：**
```go
// 检查知识库使用的 embedding 模型
kb, _ := knowledgeBaseService.GetKnowledgeBaseByID(ctx, kbID)
fmt.Printf("Embedding Model ID: %s\n", kb.EmbeddingModelID)

// 测试 embedding 生成
model, _ := modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
embedding, _ := model.Embed(ctx, "测试问题")
fmt.Printf("Embedding length: %d\n", len(embedding))
```

---

## 优化建议

### 1. 降低阈值（快速见效）
```yaml
# config.yaml 或智能体配置
conversation:
  vector_threshold: 0.5  # 从 0.7 降低到 0.5
  keyword_threshold: 0.5
```

### 2. 丰富相似问题
- 使用系统生成的相似问题功能
- 手动添加常见的相似问法
- 覆盖同义词、语序变化、口语化表达

### 3. 优化 Embedding 模型
- 选择更适合中文的 embedding 模型
- 考虑微调 embedding 模型（如果有足够数据）

### 4. 启用 FAQ 关键词检索（需改代码）
考虑修改 `knowledgebase_search.go`，为 FAQ 也添加关键词检索支持。

### 5. 监控与告警
- 监控 FAQ 命中率（命中次数/总查询次数）
- 记录未命中的高频问题，用于优化 FAQ 库

---

## 相关文档

- [KnowledgeQA 处理流程](./KnowledgeQA.md)
- [AgentQA 处理流程](./AgentQA.md)
- [查询意图探索](./query-intent-explore.md)
- [知识库搜索结果处理](./knowledgebase_search_results.md)

---

**文档版本：** 1.0  
**更新时间：** 2026-04-28  
**维护者：** WeKnora Team
