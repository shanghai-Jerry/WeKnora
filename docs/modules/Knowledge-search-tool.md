# Knowledge Search Tool 知识检索工具分析

## 概述

`internal/agent/tools/knowledge_search.go` 实现了基于语义/向量搜索的知识检索工具。该工具通过混合搜索（Hybrid Search）、重排序（Rerank）、最大边际相关性（MMR）等策略，从知识库中检索与用户查询相关的文本块（chunk）。

## 一、相关性判断策略

### 1. Hybrid Search（混合搜索）

**位置**: `concurrentSearchByTargets` 方法（第462行）

混合搜索同时进行两种检索，并使用 **RRF（Reciprocal Rank Fusion）** 进行结果融合：

- **向量搜索**: 基于 embedding 的语义相似度匹配
- **关键词搜索**: 基于关键词的文本匹配

**搜索参数**:
| 参数 | 说明 | 默认值 |
|------|------|--------|
| `topK` (MatchCount) | 召回的最大 chunk 数量 | 5 |
| `vectorThreshold` | 向量相似度最低阈值 | 0.6 |
| `keywordThreshold` | 关键词匹配最低阈值 | 0.5 |

**关键逻辑**: 阈值过滤在 HybridSearch 内部、RRF 融合**之前**完成。RRF 融合后的分数范围为 `[0, ~0.033]`（当两个检索都排第1时：`2/(60+1)`）。

### 2. 去重（Deduplication）

**位置**: `deduplicateResults` 方法（第991行）

去重在 rerank 之前和之后各执行一次，使用多种维度判断重复：

- **ID 去重**: 基于 chunk ID
- **Parent Chunk ID 去重**: 基于父块 ID（格式：`parent:<id>`）
- **Knowledge+Index 去重**: 基于 `kb:<knowledge_id>#<chunk_index>`
- **内容签名去重**: 通过 `searchutil.BuildContentSignature` 生成内容签名，检测近似重复内容

重复项保留最高分数版本。

### 3. Rerank（重排序）

**位置**: `rerankResults` 方法（第595行）

Rerank 对搜索结果进行重新打分和排序，策略如下：

#### 3.1 Rerank 模型选择优先级

1. **LLM-based Rerank（优先）**: 使用 `chatModel` 进行基于提示词的重排序
   - 位置: `rerankWithLLM` 方法（第714行）
   - 批量处理：每批 15 个结果
   - 评分范围：0.0 - 1.0
   - 评估因素：查询匹配度、信息完整性、语义相关性、关键词覆盖、信息准确性

2. **Rerank Model（备选）**: 使用专用 rerank 模型
   - 位置: `rerankWithModel` 方法（第952行）
   - Fallback: 如果 rerank model 失败或返回空结果，回退到 chatModel

3. **无 Rerank**: 如果不配置任何 rerank 模型，跳过此步骤

#### 3.2 FAQ 特殊处理

FAQ 类型的知识库结果**跳过 rerank**，保留原始分数，因为它们是显式匹配的问答对。

#### 3.3 Composite Scoring（复合评分）

**位置**: `compositeScore` 方法（第1384行）

Rerank 后对结果应用复合评分：

```
composite = (0.6 × modelScore + 0.3 × baseScore + 0.1 × sourceWeight) × positionPrior
```

- `modelScore`: rerank 模型给出的分数
- `baseScore`: rerank 前的原始分数
- `sourceWeight`: 来源权重（web_search 来源为 0.95，其他为 1.0）
- `positionPrior`: 位置先验，对文档中靠前位置的 chunk 给予轻微偏好

### 4. MMR（Maximal Marginal Relevance）

**位置**: `applyMMR` 方法（第1423行）

MMR 用于减少结果冗余，提高多样性：

- **lambda = 0.7**: 平衡相关性和多样性（0=纯多样性，1=纯相关性）
- **冗余计算**: 使用 Jaccard 相似度计算 token 集合的重叠度
- **选择策略**: 每次选择 `MMR = λ×相关性 - (1-λ)×冗余度` 最高的候选

### 5. 最终排序

所有处理后的结果按分数降序排列，相同分数时按 KnowledgeID 排序。

## 二、Fallback 机制

### 检索 Fallback 链

```
Tenant ConversationConfig → Global Config → 硬编码默认值
```

### Rerank Fallback 链

```
chatModel (LLM-based) → rerankModel → 原始结果（无 rerank）
```

具体逻辑（第626-651行）：
1. 如果配置了 `rerankModel`，先尝试使用它
2. 如果 `rerankModel` 失败或返回空结果，且配置了 `chatModel`，则 fallback 到 `chatModel`
3. 如果两者都未配置，使用原始结果

### 搜索目标 Fallback

- 如果用户指定了 `knowledge_base_ids`，则过滤搜索目标到指定 KB
- 否则使用预计算的 `searchTargets`
- 如果最终没有搜索目标，返回错误

## 三、阈值设定位置和获取优先级

### 优先级（从高到低）

| 优先级 | 来源 | 位置 |
|--------|------|------|
| 1（最高） | Tenant ConversationConfig | `internal/types/tenant.go:179` |
| 2 | Global Config（`config.Conversation`） | `internal/config/config.go:74` |
| 3（最低） | 硬编码默认值 | `knowledge_search.go` 第269-280行 |

### 1. Tenant ConversationConfig

**定义位置**: `internal/types/tenant.go:179-193`

```go
type ConversationConfig struct {
    EmbeddingTopK    int     `json:"embedding_top_k"`
    KeywordThreshold float64 `json:"keyword_threshold"`
    VectorThreshold  float64 `json:"vector_threshold"`
    // ... 其他字段
}
```

**获取方式**: 通过 context 中的 `types.TenantInfoContextKey` 获取当前租户信息（第240行）

```go
if tenantVal := ctx.Value(types.TenantInfoContextKey); tenantVal != nil {
    if tenant, ok := tenantVal.(*types.Tenant); ok && tenant != nil && tenant.ConversationConfig != nil {
        cc := tenant.ConversationConfig
        // 使用 cc.EmbeddingTopK, cc.VectorThreshold, cc.KeywordThreshold
    }
}
```

> **注意**: `ConversationConfig` 在 tenant 中已被标记为 Deprecated，推荐使⽤ CustomAgent 配置。

### 2. Global Config

**定义位置**: `internal/config/config.go:73-78`

```go
type ConversationConfig struct {
    KeywordThreshold float64 `yaml:"keyword_threshold"`
    EmbeddingTopK    int    `yaml:"embedding_top_k"`
    VectorThreshold  float64 `yaml:"vector_threshold"`
    // ... 其他字段
}
```

**YAML 配置示例**:
```yaml
conversation:
  embedding_top_k: 10
  vector_threshold: 0.6
  keyword_threshold: 0.5
```

**获取方式**: 通过 `t.config.Conversation` 访问（第258-266行）

### 3. 硬编码默认值

**位置**: `knowledge_search.go` 第269-280行

| 参数 | 默认值 | 代码行 |
|------|--------|--------|
| `topK` | 5 | 第270行 |
| `vectorThreshold` | 0.6 | 第273行 |
| `keywordThreshold` | 0.5 | 第276行 |
| `minScore` | 0.3 | 第279行 |

> **注意**: `minScore` 实际未被使用，因为 RRF 分数范围与 `[0,1]` 不同，阈值过滤已在 HybridSearch 内部完成。

### 4. 其他相关配置

#### CustomAgent Config

**位置**: `internal/types/custom_agent.go:164-168`

```go
type Config struct {
    EmbeddingTopK    int     `yaml:"embedding_top_k"`
    KeywordThreshold float64 `yaml:"keyword_threshold"`
    VectorThreshold  float64 `yaml:"vector_threshold"`
}
```

默认值（第252-259行）:
- `EmbeddingTopK`: 10
- `KeywordThreshold`: 0.3
- `VectorThreshold`: 0.5

#### RetrievalConfig

**位置**: `internal/types/retrieval_config.go`

这是一个较新的配置结构，用于集中管理检索相关配置：

```go
type RetrievalConfig struct {
    EmbeddingTopK    int     `json:"embedding_top_k"`
    VectorThreshold  float64 `json:"vector_threshold"`    // 默认 0.15
    KeywordThreshold float64 `json:"keyword_threshold"`  // 默认 0.3
}
```

提供带 fallback 的获取方法：
- `GetEffectiveEmbeddingTopK()`
- `GetEffectiveVectorThreshold()`
- `GetEffectiveKeywordThreshold()`

## 四、总结

### 相关性判断流程

```
用户输入 queries
    ↓
Hybrid Search (向量 + 关键词，RRF 融合)
    ↓ (应用 vectorThreshold 和 keywordThreshold 过滤)
去重 (ID / ParentID / Knowledge+Index / 内容签名)
    ↓
Rerank (LLM → rerankModel → 原始) + Composite Scoring
    ↓
MMR (lambda=0.7，提高多样性)
    ↓
最终去重 + 按分数排序
    ↓
返回结果
```

### 阈值来源总结

| 阈值 | Tenant Config | Global Config | 硬编码默认 |
|------|:-------------:|:-------------:|:----------:|
| topK | ConversationConfig.EmbeddingTopK | config.Conversation.EmbeddingTopK | 5 |
| vectorThreshold | ConversationConfig.VectorThreshold | config.Conversation.VectorThreshold | 0.6 |
| keywordThreshold | ConversationConfig.KeywordThreshold | config.Conversation.KeywordThreshold | 0.5 |
