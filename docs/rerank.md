# KnowledgeQA Rerank Pipeline 详解

> 文档位置：`docs/rerank.md`
> 源码位置：`internal/application/service/chat_pipeline/rerank.go`

## 一、整体流程

Rerank 位于 KnowledgeQA Pipeline 的**检索阶段**，顺序如下：

```
CHUNK_SEARCH (向量搜索) → CHUNK_RERANK (重排) → CHUNK_MERGE (合并) → FILTER_TOP_K (过滤TopK)
```

触发时机：`sessionKnowledgeQA` 中会根据 `RetrievalConfig` 自动选择或使用第一个可用的 rerank 模型。

---

## 二、核心流程

### 2.1 输入

| 字段 | 类型 | 说明 |
|------|------|------|
| `chatManage.SearchResult` | `[]*SearchResult` | 向量搜索返回的候选结果 |
| `chatManage.RerankModelID` | `string` | 重排模型ID |
| `chatManage.RerankThreshold` | `float64` | 相似度阈值（默认0.2，范围-10~10） |
| `chatManage.RerankTopK` | `int` | 重排后保留结果数（默认10） |
| `chatManage.RewriteQuery` | `string` | 重写后的查询词 |

### 2.2 处理步骤

#### Step 1: 预处理候选文档

```go
// 源码位置：rerank.go 第73-97行
for _, result := range chatManage.SearchResult {
    // DirectLoad 类型直接保留，跳过重排
    if result.MatchType == types.MatchTypeDirectLoad {
        directLoadResults = append(directLoadResults, result)
        continue
    }
    // 合并Content、ImageInfo、GeneratedQuestions构建语义段落
    passage := getEnrichedPassage(ctx, result)
    // 跳过空段落
    if strings.TrimSpace(passage) == "" {
        continue
    }
    passages = append(passages, passage)
    candidatesToRerank = append(candidatesToRerank, result)
}
```

**关键点**：
- DirectLoad 类型（如FAQ）直接保留，不经过重排模型
- 段落内容经过 `getEnrichedPassage` 增强：合并 `Content` + `ImageInfo`(caption/ocr) + `GeneratedQuestions`

#### Step 2: 调用重排模型

```go
// 源码位置：rerank.go 第238行
rerankResp, err := rerankModel.Rerank(ctx, query, passages)
```

**支持的模型**（通过 Provider 自动识别）：

| Provider | BaseURL | 模型示例 |
|----------|--------|----------|
| Aliyun | `dashscope.aliyuncs.com` | qwen3-rerank |
| Zhipu | `open.bigmodel.cn` | - |
| Jina | `api.jina.ai` | jina-reranker |
| Nvidia | `ai.api.nvidia.com` | nv-embed-v1, rerank-qa-mistral-4b |
| OpenAI兼容 | 自定义 | - |

#### Step 3: 阈值过滤与Fallback

```go
// 源码位置：rerank.go 第272-303行
for _, result := range rerankResp {
    if result.RelevanceScore >= chatManage.RerankThreshold {
        rankFilter = append(rankFilter, result)
    }
}

// Fallback: 如果过滤后全部丢失，但最高分>=0.15，保留Top1
if len(rankFilter) == 0 && len(rerankResp) > 0 && rerankResp[0].RelevanceScore >= 0.15 {
    rankFilter = rerankResp[:1]
}
```

**关键点**：
- 默认阈值 `0.2`，低于此分数的结果被过滤
- Fallback 保护：即使全过滤，如果最高分 >= 0.15，仍保留1条
- 阈值自动降级：如原始阈值>0.3且无结果，会降至0.3重试（第114-129行）

#### Step 4: 分数合成

```go
// 源码位置：rerank.go 第322-343行
composite := 0.6*modelScore + 0.3*baseScore + 0.1*sourceWeight
```

**公式**：`综合分数 = 0.6 × 重排分数 + 0.3 × 向量分数 + 0.1 × 来源权重`

| 来源 | 权重 |
|------|------|
| WebSearch | 0.95 |
| 其他 | 1.0 |

#### Step 5: MMR多样性重排序

```go
// 源码位置：rerank.go 第346-426行
final := applyMMR(ctx, reranked, chatManage, min(len(reranked), max(1, chatManage.RerankTopK)), 0.7)
```

**MMR (Maximal Marginal Relevance)** 算法：
- 目标：在相关性与多样性之间取得平衡
- 参数 `λ = 0.7`：越高越偏重相关性
- 实现：使用 Jaccard 相似度计算冗余度

```go
mmr := λ*relevance - (1.0-λ)*redundancy
```

---

## 三、段落清洗

> 源码位置：rerank.go 第428-530行

重排模型基于语义相似度，Markdown格式、URL等属于噪声，需要预处理清洗：

| 步骤 | 正则表达式 | 处理 |
|------|-----------|------|
| 1 | `` ```...``` `` | 删除代码块 |
| 2 | `` $$...$$ `` | 删除LaTeX块 |
| 3 | `` <[^>]*> `` | 删除HTML标签 |
| 4 | `` ![[^\]]*]\([^)]*\) `` | 删除Markdown图片 |
| 5 | `` [[^\]]+]\([^)]*\) `` | 保留链接文本，删除URL |
| 6 | `` https?://... `` | 删除纯URL |
| 7 | `` \|---\| `` | 删除表格分隔符 |
| 8 | `` ^#{1,6}\s+ `` | 删除标题标记 |
| 9 | `` ^>\s? `` | 删除引用标记 |
| 10 | `` \*{1,3}(.+?)\*{1,3} `` | 保留粗体/斜体文本 |
| 11 | `` ^[\t ]*[-*+]\s+ `` | 删除列表标记 |

---

## 四、输出

```go
chatManage.RerankResult = final
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `RerankResult` | `[]*SearchResult` | 重排后的最终结果 |
| `Score` | `float64` | 综合分数（0~1） |
| `Metadata.base_score` | `string` | 原始向量搜索分数 |
| `Metadata.faq_boosted` | `string` | FAQ-boosted标记 |

---

## 五、配置参数

### 5.1 环境变量

```yaml
# config.yaml
conversation:
  rerank_top_k: 10          # 重排后保留结果数
  rerank_threshold: 0.2      # 相似度阈值
  rerank_model_id: ""        # 模型ID（可选，不填则自动选择）
```

### 5.2 会话级配置

```json
{
  "retrieval_config": {
    "rerank_top_k": 10,
    "rerank_threshold": 0.3,
    "rerank_model_id": "qwen3-rerank"
  }
}
```

---

## 六、流程图

```
┌─────────────────────────────────────────────────────────────┐
│                    CHUNK_SEARCH                            │
│              (向量搜索 → SearchResult)                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    CHUNK_RERANK                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 1. 预处理候选文档                                      │ │
│  │    - DirectLoad 跳过                                     │ │
│  │    - 合并 Content + ImageInfo + GeneratedQuestions    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 2. 调用重排模型                                        │ │
│  │    - query: RewriteQuery                               │ │
│  │    - documents: 清洗后的段落                           │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 3. 阈值过滤                                            │ │
│  │    - RelevanceScore >= RerankThreshold                  │ │
│  │    - Fallback: 最高分>=0.15 时保留Top1                  │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 4. 分数合成                                            │ │
│  │    composite = 0.6*model + 0.3*base + 0.1*source     │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 5. MMR多样性重排                                        │ │
│  │    - λ=0.7, Jaccard冗余度计算                          │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  CHUNK_MERGE + FILTER_TOP_K                 │
└─────────────────────────────────────────────────────────────┘
```

---

## 七、FAQ增强

当启用 `FAQPriorityEnabled` 时，FAQ类型结果会获得分数提升：

```go
// 源码位置：rerank.go 第155-167行
if chatManage.FAQPriorityEnabled && chatManage.FAQScoreBoost > 1.0 &&
    sr.ChunkType == string(types.ChunkTypeFAQ) {
    originalScore := sr.Score
    sr.Score = math.Min(sr.Score*chatManage.FAQScoreBoost, 1.0)
    sr.Metadata["faq_boosted"] = "true"
}
```

- 默认 boost 因子：`1.5`
- 上限：`1.0`（不超过满分）

---

## 八、日志关键点

| 日志字段 | 说明 |
|----------|------|
| `candidate_cnt` | 候选文档数量 |
| `direct_cnt` | DirectLoad数量（跳过重排） |
| `threshold_degrade` | 阈值降级日志 |
| `mmr_start` / `mmr_done` | MMR起止 |
| `composite_top` | 综合分数Top3 |