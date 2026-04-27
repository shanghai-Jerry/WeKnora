# Wiki Ingest: Entity Page & Issue Construction Pipeline

本文档总结用户导入文档后，系统如何通过 LLM 驱动管道自动构建 wiki 实体/概念页面和问题的完整流程。

---

## 1. 整体架构概述

Wiki ingest 是一个异步、分批的 **Map-Reduce** 管道，在用户上传文档到知识库后自动触发。核心入口：

- `internal/application/service/wiki_ingest.go` — 主服务和协调逻辑
- `internal/application/service/wiki_ingest_batch.go` — 批处理 (Map-Reduce)
- `internal/application/service/wiki_ingest_cite.go` — Chunk 引用管道 (Pass 0 + Pass 1..N)
- `internal/application/service/wiki_ingest_dedup.go` — 预过滤去重
- `internal/agent/prompts_wiki.go` — 所有 LLM 提示词模板

支持的页面类型（`internal/types/wiki_page.go:11-31`）：

| 类型 | Slug 前缀 | 来源 |
|------|-----------|------|
| Summary | `summary/...` | 导入自动创建（文档摘要页） |
| Entity | `entity/...` | 导入自动创建（实体：人、组织、产品、地点等） |
| Concept | `concept/...` | 导入自动创建（概念：主题、方法论、理论等） |
| Index | `index` | 系统页面，自动维护 |
| Log | `log` | 系统页面，记录操作日志 |
| Synthesis | `synthesis/...` | Agent 手动创建（跨文档分析） |
| Comparison | `comparison/...` | Agent 手动创建（对比分析） |

---

## 2. 异步任务排队机制

### 2.1 入队触发

用户上传文档后，`EnqueueWikiIngest` (`wiki_ingest.go:185`) 被调用：

```
文件上传 → EnqueueWikiIngest → Redis RPUSH wiki:pending:{kbID} → asynq.Enqueue(延迟 30s)
```

- 每个文档上传将其 `knowledgeID` push 到 Redis 待处理列表
- 调度一个 **延迟 30 秒** 的 asynq 任务（`wikiIngestDelay`）
- 30 秒内的多次上传共享同一个待处理列表，由第一个到达的任务批量处理

### 2.2 并发控制

使用 Redis `wiki:active:{kbID}` 锁（SETNX + TTL = 60s）防止同一 KB 并发批处理。锁通过后台 goroutine 每 20 秒续期一次，防止崩溃后孤儿锁长时间阻塞。

### 2.3 Lite 模式回退

无 Redis 时直接执行单文档处理（Lite mode），payload 中携带操作列表。

---

## 3. Map-Reduce 批处理流程 (`wiki_ingest_batch.go`)

```
ProcessWikiIngest
  ├── 1. 获取 Redis 待处理列表 (peekPendingList)
  ├── 2. 预加载所有现有页面 (ListAllPages)
  ├── 3. MAP 阶段（并行，上限 10 并发）
  │     ├── Ingest 操作：mapOneDocument()
  │     └── Retract 操作：生成 retract SlugUpdate
  ├── 4. REDUCE 阶段（并行，上限 10 并发）
  │     └── reduceSlugUpdates() — 按 slug 分组执行 LLM 修改
  ├── 5. 后处理
  │     ├── appendLogEntry（操作日志）
  │     ├── rebuildIndexPage（重建索引页）
  │     ├── cleanDeadLinks（清理死链）
  │     ├── injectCrossLinks（注入交叉链接）
  │     └── publishDraftPages（发布草稿页面）
  └── 6. scheduleFollowUp（调度后续批处理任务）
```

### 3.1 MAP 阶段：`mapOneDocument()` (`wiki_ingest_batch.go:422`)

对每个文档执行以下步骤：

```
mapOneDocument
  ├── 1. 加载文档 chunks（ListChunksByKnowledgeID）
  ├── 2. 重建富文本内容（reconstructEnrichedContent — 含图片 OCR）
  ├── 3. Pass 0：候选 Slug 提取（extractCandidateSlugs）【并行1】
  │     ├── 调用 WikiCandidateSlugPrompt
  │     ├── 解析 JSON → entities + concepts
  │     └── 去重（deduplicateExtractedBatch）
  ├── 4. 并行执行【并行2】：
  │     ├── Summary 生成（WikiSummaryPrompt → 文档摘要页内容）
  │     └── Chunk Citation 分类（classifyChunkCitations）
  │           ├── 将 chunks 分批次（splitChunksIntoCitationBatches）
  │           ├── 对每批次调用 WikiChunkCitationPrompt（并行 4 并发）
  │           └── 合并各批次结果
  ├── 5. 合并引用（mergeCitationsIntoItems — 将 chunk ID 回填到 extractedItem）
  ├── 6. 老的 slug 与新提取结果协调（三种情况）：
  │     ├── (a) 旧 slug ∉ 新结果：生成 "retractStale" 更新
  │     ├── (b) 旧 slug ∈ 新结果且为 entity/concept：同时生成 retract + addition
  │     │      → Replace-not-append 语义
  │     └── (c) summary slug：跳过（summary 分支总是全量覆写）
  └── 7. 返回 SlugUpdate 列表 + docIngestResult
```

#### 3.1.1 Pass 0 失败回退

如果 Pass 0 失败，回退到旧的 legacy 提取器 `extractEntitiesAndConceptsNoUpsert`，它使用 `WikiKnowledgeExtractPrompt` 在一次 LLM 调用中提取 entities + concepts（但没有 chunk 级别的引用）。

### 3.2 REDUCE 阶段：`reduceSlugUpdates()` (`wiki_ingest_batch.go:802`)

对每个 slug 分组的所有更新进行合并处理：

```
reduceSlugUpdates
  ├── filterLiveUpdates（过滤已删除文档的更新）
  ├── 按类型分类更新：summary / retract / entity / concept
  ├── 情况 A：有 summary 更新
  │     └── 直接覆写页面（title = DocTitle + " - Summary"）
  ├── 情况 B：有 retract / addition 更新
  │     ├── 解析 retract 引用（删除文档的源引用）
  │     ├── resolveCitedChunks（批量加载真实 chunk 内容）
  │     ├── 构建 <new_information> / <deleted_documents> 块
  │     └── 调用 WikiPageModifyPrompt → LLM 增量编辑页面
  └── 持久化（CreatePage 或 UpdatePage）
```

---

## 4. Chunk Citation 管道（核心创新）

这是实体/概念页面质量的关键——避免 LLM 复述失真，直接用源 chunk 原文字段构建页面内容。

### 4.1 Pass 0：候选 Slug 提取

**Prompt**: `WikiCandidateSlugPrompt` (`prompts_wiki.go:121`)

- 输入：完整文档文本 + 粒度配置
- 输出：轻量级 skeleton（name, slug, aliases, description + 简短 fallback details）
- 粒度控制（三个级别）：

| 级别 | 描述 | 适用场景 |
|------|------|----------|
| `focused` | 仅提取文档主要主题（3-7 项） | 简历、公告等单主题文档 |
| `standard` | 主要主题 + 实质性讨论的次要主题 | 默认，平衡 |
| `exhaustive` | 所有命名实体/概念 | 技术词汇表用途 |

粒度由 KB 的 `WikiConfig.ExtractionGranularity` 配置。

**Slug 连续性规则**：如果文档之前已被提取过，LLM 必须复用已有的 slug（由 `PreviousSlugs` 提供），保证跨文档更新的 slug 稳定性。

### 4.2 Pass 1..N：Chunk 引用分类

**Prompt**: `WikiChunkCitationPrompt` (`prompts_wiki.go:203`)

- 将文档的 text chunks 按 12000 rune 预算分批（`splitChunksIntoCitationBatches`）
- 每批次提交给 LLM，要求对每个候选 slug 列出**实质性讨论它的 chunk ID**
- 错误容忍：批次失败不中断其他批次
- LLM 还可以发现 Pass 0 遗漏的新 slug（`new_slugs` 数组）

### 4.3 内容合并

`mergeCitationsIntoItems` 将 chunk 引用回填到 `extractedItem.SourceChunks` 字段。未被任何 chunk 引用的 item 保留 fallback Details 作为降级内容。

### 4.4 内容物化

在 Reduce 阶段，`resolveCitedChunks` 批量加载真实 chunk 内容 → `<new_information>` 块携带原始 chunk 文本（而非 LLM 复述），`WikiPageModifyPrompt` 编译时保持尽可能接近源措辞。

---

## 5. 实体/概念页面的构建（完整链路）

```
文档 chunks
  │
  ├── [LLM] WikiCandidateSlugPrompt (Pass 0)
  │     → {entities: [{name, slug, aliases, description, details}], concepts: [...]}
  │
  ├── [LLM] WikiDeduplicationPrompt
  │     → 检测新 item 与已有页面的去重映射 (merges: {new_slug → existing_slug})
  │
  ├── [LLM + Chunks] WikiChunkCitationPrompt (Pass 1..N)
  │     → {citations: {slug → [chunk_alias, ...]}, new_slugs: [...]}
  │
  ├── [Code] mergeCitationsIntoItems
  │     → extractedItem.SourceChunks 被填充
  │
  ├── [Code + DB] resolveCitedChunks
  │     → chunk ID 翻译为真实文本内容
  │
  └── [LLM] WikiPageModifyPrompt (Reduce)
        → 合并新/删除信息到页面 markdown 内容
```

### 5.1 首次创建

- 页面以 `draft` 状态创建
- 批处理完成后由 `publishDraftPages` 转为 `published`

### 5.2 更新（重新导入同一文档）

- Slug 通过 `PreviousSlugs` 机制保持稳定
- 使用 **retract + addition**（类型 c 重新解析）实现 replace-not-append 语义
- `WikiPageModifyPrompt` 中的冲突检查：新信息如果不属于当前页面主题则被拒绝

### 5.3 去重

`deduplicateExtractedBatch` (`wiki_ingest.go:826`)
1. **预过滤**（`selectDedupCandidatePages`）：基于 Jaccard 相似度过滤候选页（`wiki_ingest_dedup.go`）
2. **LLM 去重**（`WikiDeduplicationPrompt`）：严格判断是否同一事物
3. **代码校验**（`validMerge`）：验证目标 slug 存在且类型匹配

---

## 6. Issue 的构建机制

### 6.1 两种 Issue 来源

#### A. Wiki Lint 自动检测 (`wiki_lint.go`)

`WikiLintService.RunLint()` 扫描整个 wiki 生成结构化问题报告：

| Issue 类型 | 严重度 | 自动修复 |
|-----------|--------|---------|
| `orphan_page` — 无入链页面 | warning | 否 |
| `broken_link` — 死链接 | error | 是 (移除 `[[]]`) |
| `stale_ref` — 引用已删除知识 | error | 是 (移除源引用) |
| `missing_cross_ref` — 未建立链接的关联 | info | 否 |
| `empty_content` — 内容过少 | warning | 是 (归档) |
| `duplicate_slug` — 重复 slug | - | - |

健康分计算：100 分起步，按问题扣分 → 最终 0-100。

#### B. Agent 手动标记 (`wiki_flag_issue.go`)

通过 `wiki_flag_issue` 工具由 Agent 在对话中标记 wiki 页面问题：

```
参数: {slug, issue_type, description, suspected_knowledge_ids}
issue_type 枚举: mixed_entities | contradictory_facts | out_of_date | other
```

Agent 工具提交的 Issue 以 `reported_by: "wiki-researcher-agent"` 记录，状态为 `pending`。

### 6.2 Issue 数据模型 (`types/wiki_page.go:235-249`)

```go
type WikiPageIssue struct {
    ID                    string       // UUID 主键
    TenantID              uint64
    KnowledgeBaseID       string
    Slug                  string       // 关联的 wiki 页面 slug
    IssueType             string       // mixed_entities, contradictory_facts, out_of_date, other
    Description           string       // 问题描述
    SuspectedKnowledgeIDs StringArray  // 疑似污染源的知识 ID
    Status                string       // pending (默认)
    ReportedBy            string       // "wiki-researcher-agent" 或系统
    CreatedAt / UpdatedAt time.Time
}
```

存储表：`wiki_page_issues`

### 6.3 Issue 生命周期

```
创建 (CreateIssue)
  ├── Lint 扫描 → 发现即时报告（不稳定，下次重新扫描）
  └── Agent 标记 → 持久化 pending Issue

展示：WikiStats.PendingIssues = count of pending issues

修复：
  ├── Lint AutoFix → 自动处理可修复问题
  └── UpdateIssueStatus → Agent 手动更新状态
```

---

## 7. 相关 Prompt 模板汇总

| Prompt 常量 | 文件行号 | 用途 |
|---|---|---|
| `WikiSummaryPrompt` | `prompts_wiki.go:8` | 生成文档摘要页 |
| `WikiKnowledgeExtractPrompt` | `prompts_wiki.go:40` | Legacy 提取实体+概念（Pass 0 回退） |
| `WikiCandidateSlugPrompt` | `prompts_wiki.go:121` | Pass 0 候选 slug 提取 |
| `WikiChunkCitationPrompt` | `prompts_wiki.go:203` | Pass 1..N chunk 引用分类 |
| `WikiPageModifyPrompt` | `prompts_wiki.go:262` | 增量更新已有 wiki 页面 |
| `WikiDeduplicationPrompt` | `prompts_wiki.go:376` | 新 item 与已有页面的去重 |
| `WikiIndexIntroPrompt` | `prompts_wiki.go:326` | 生成索引页引言 |
| `WikiIndexIntroUpdatePrompt` | `prompts_wiki.go:342` | 增量更新索引页引言 |
| `WikiLogEntryTemplate` | `prompts_wiki.go:368` | 操作日志模板（非 LLM 生成） |

---

## 8. 数据流总图

```
用户上传文档
    │
    ├── 文档解析 → chunks 写入 DB
    └── EnqueueWikiIngest(knowledgeID)
            │
            ▼  (30s 延迟后)
    ProcessWikiIngest 批处理
            │
    ┌───────┴───────┐
    ▼               ▼
  Ingest         Retract
    │               │
    ├──────────────────────────────┐
    │ Map Phase (mapOneDocument)   │
    │  ├── 提取候选 slug (Pass 0)  │
    │  ├── 生成摘要                │
    │  ├── Chunk 引用分类          │
    │  └── 协调旧 slug             │
    │      → []SlugUpdate          │
    └──────────────────────────────┘
            │
    ┌───────┴───────────┐
    ▼                   ▼
  Reduce Phase    (按 slug 分组)
    │
    ├── Summary?  → 直接覆写
    ├── Entity?   → WikiPageModifyPrompt (增量编辑)
    └── Concept?  → WikiPageModifyPrompt (增量编辑)
            │
            ▼
  Post-processing
    ├── 操作日志 → log.md
    ├── 索引页重建 → index.md (LLM intro + code directory)
    ├── 死链清理
    ├── 交叉链接注入 (linkifyContent)
    └── 发布草稿 → published
```

---

## 9. 长文档截断与 Pipeline 影响

系统对单个文档传入 LLM 的内容做了硬性限制：

| 阶段 | 截断阈值 | 文件位置 |
|------|---------|---------|
| **Pass 0 + Summary 生成** | `maxContentForWiki = 32768 rune` | `wiki_ingest.go:35` |
| **Pass 1..N Chunk Citation** | `maxRunesPerCitationBatch = 12000 rune/batch` | `wiki_ingest_cite.go:24` |

### 9.1 Pass 0 + Summary 的截断

```go
// wiki_ingest_batch.go:452-456
content := reconstructEnrichedContent(ctx, s.chunkRepo, payload.TenantID, chunks)
rawRuneCount := len([]rune(content))
if len([]rune(content)) > maxContentForWiki {
    content = string([]rune(content)[:maxContentForWiki])
}
```

`mapOneDocument` 中，文档内容重建后超过 **32768 rune**（约 3.2 万字符）即被**头部截断**。被截断掉的后半部分文档不会进入 Pass 0 的候选 slug 提取，也不会进入 Summary 生成。这意味着：

- **实体/概念提取不完整**：长文档后半部分的实体/概念不会被识别
- **Summary 缺失后半部分信息**：摘要页只覆盖了文档前 32768 rune 的内容
- **截断行为无感知**：`"(truncated)"` 标记只出现在日志中（`content_len(raw=X,truncated=Y)`），但 Summary 页本身不会标注"截断"状态

**典型场景**：一本 200 页的 PDF，前 80 页讲 A 公司，后 120 页讲 B 公司 —— Pass 0 和 Summary 都只看前 80 页，B 公司相关内容完全被忽略。

### 9.2 Chunk Citation 的截断

即使 Pass 0 成功识别了大量候选 slug，Pass 1..N 中的 chunk citation 仍面临第二个截断：

- 每个批次最多 `12000 rune`（约 1.2 万字符）
- 超出后自动开新批次，最多 **4 并发** 执行（`maxCitationBatchConcurrency = 4`）
- 每个 chunk 独立截断：超大批次会独占一个 batch，不会被静默丢弃

这保证了**长文档不会漏掉任何 chunk**，但代价是：
- LLM 调用次数随文档长度线性增长
- 每个 batch 的成本和耗时累加
- 当文档 chunk 极多时，整体处理时间可能超过 asynq 的 `Timeout(60 * time.Minute)` 上限

### 9.3 截断的连锁影响

```
长文档 (e.g. 500+ 页 PDF)
    │
    ├── Pass 0 + Summary — 头部截断至 32768 rune
    │     ├── 后半部分实体/概念 → 完全丢失
    │     └── Summary → 不完整
    │
    └── Chunk Citation — 按 12000 rune/batch 分批
          ├── 批次数量多 → LLM 调用成本高
          ├── 4 并发上限 → 串行等待 → 处理时间长
          └── 超时风险 → asynq Timeout 60min
```

### 9.4 当前设计限制

- **无 Chunk 级预抽样**：截断发生在 `reconstructEnrichedContent` 之后，LLM 看到的是被截断的连续文本，不是均匀采样的 chunk 子集。越靠后的 chunk 被引用的概率越低。
- **截断后无法恢复**：没有机制将长文档分散到多个"虚拟文档"中并行处理
- **没有文档级截断提示**：Summary 页不标注截断状态，用户无法判断摘要是否完整
