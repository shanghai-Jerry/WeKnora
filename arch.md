# WeKnora 架构文档

> 版本：v0.3.3 | 协议：MIT | 官网：https://weknora.weixin.qq.com

---

## 一、系统总览

WeKnora 是腾讯开源的企业级 LLM 驱动文档理解与检索框架，核心采用 **RAG（Retrieval-Augmented Generation，检索增强生成）** 范式，将文档解析、向量索引、智能检索与大模型推理四大模块解耦为可独立部署的微服务，支持私有化部署。

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           WeKnora 系统架构                                │
│                                                                          │
│  ┌─────────────┐    ┌──────────────────────────────────────────────┐     │
│  │  Vue3 前端   │    │                   Go 后端 (Gin)               │     │
│  │  (Nginx:80) │◄──►│                   (App:8080)                  │     │
│  └─────────────┘    │  ┌─────────┐ ┌──────────┐ ┌──────────────┐  │     │
│                     │  │ Handler │ │ Agent    │ │ MCP 集成     │  │     │
│                     │  │ (REST)  │ │ (ReACT)  │ │ (工具扩展)   │  │     │
│                     │  └────┬────┘ └──────────┘ └──────────────┘  │     │
│                     │       │                                       │     │
│                     │  ┌────▼────────────────────────────────┐     │     │
│                     │  │        RAG Pipeline                  │     │     │
│                     │  │  rewrite→preprocess→search→rerank   │     │     │
│                     │  │  →merge→filter→generate→stream      │     │     │
│                     │  └────────────────────────────────────-┘     │     │
│                     └─────────┬────────────────────────────────────┘     │
│                               │                                          │
│  ┌──────────────────┐  ┌──────▼──────┐  ┌──────────────────────────┐    │
│  │  DocReader       │  │  存储层      │  │  向量数据库               │    │
│  │  (Python/gRPC)   │  │ PostgreSQL  │  │  pgvector / ES /          │    │
│  │  - PDF/Word/Img  │  │ Redis       │  │  Qdrant / Milvus /        │    │
│  │  - OCR/PaddleOCR │  │ MinIO/COS   │  │  Weaviate                 │    │
│  │  - 文本分块      │  └─────────────┘  └──────────────────────────┘    │
│  └──────────────────┘                                                    │
│                                                                          │
│  ┌──────────────────┐  ┌──────────────┐  ┌──────────────────────────┐   │
│  │  LLM 服务        │  │  图数据库    │  │  可观测性                 │   │
│  │  Ollama/OpenAI   │  │  Neo4j       │  │  Jaeger(OTel) / Logrus   │   │
│  │  兼容 API        │  │  (GraphRAG)  │  │                          │   │
│  └──────────────────┘  └──────────────┘  └──────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 二、服务组件清单

| 服务名 | 技术栈 | 默认端口 | 说明 |
|--------|--------|----------|------|
| **frontend** | Vue3 + Nginx | 80 | Web UI 前端，Nginx 反代后端 |
| **app** | Go 1.24 + Gin | 8080 | 核心后端，REST API + 异步任务 |
| **docreader** | Python + gRPC | 50051 | 文档解析微服务 |
| **postgres** | ParadeDB v0.21 (pg17) | 5432 | 主数据库 + pgvector |
| **redis** | Redis 7.0 | 6379 | 缓存 + 消息队列 (Asynq) |
| **minio** | MinIO 2025 | 9000/9001 | 对象存储（可选） |
| **neo4j** | Neo4j 2025 | 7474/7687 | 图数据库，GraphRAG（可选） |
| **qdrant** | Qdrant v1.16 | 6333/6334 | 向量数据库（可选） |
| **milvus** | Milvus v2.6 | 19530 | 向量数据库（可选） |
| **weaviate** | Weaviate 1.28 | 9035/50052 | 向量数据库（可选） |
| **rerank** | Python BGE-M3 | 8889 | 重排序服务（可选） |
| **ollama** | Ollama | 11434 | 本地 LLM 服务（可选） |
| **sglang** | SGLang | 30012 | 高性能 LLM 推理（可选） |
| **jaeger** | Jaeger 1.76 | 16686 | 链路追踪（可选） |
| **sandbox** | Docker | — | Agent Skills 隔离执行环境 |

---

## 三、目录结构

```
WeKnora/
├── cmd/
│   └── server/          # Go 服务主入口 (main.go)
├── internal/            # 核心业务逻辑
│   ├── agent/           # ReACT Agent 实现
│   ├── application/     # 应用服务层（RAG Pipeline）
│   ├── handler/         # HTTP 处理器（17个）
│   │   ├── auth.go          # 认证授权
│   │   ├── knowledge.go     # 知识条目管理
│   │   ├── knowledgebase.go # 知识库管理
│   │   ├── faq.go           # FAQ 知识库
│   │   ├── tenant.go        # 租户/共享空间
│   │   ├── organization.go  # 组织管理
│   │   ├── model.go         # 模型配置
│   │   ├── system.go        # 系统设置
│   │   ├── mcp_service.go   # MCP 工具集成
│   │   ├── custom_agent.go  # 自定义 Agent
│   │   ├── skill_handler.go # Agent Skills
│   │   └── ...
│   ├── infrastructure/  # 基础设施层（向量库、存储适配器）
│   ├── models/          # GORM 数据模型
│   ├── mcp/             # MCP 协议集成
│   ├── sandbox/         # Skills 沙箱执行
│   ├── event/           # 事件系统
│   ├── stream/          # 流式输出
│   ├── router/          # 路由注册
│   ├── middleware/       # JWT、CORS、SSRF 防护等中间件
│   ├── config/          # 配置管理（Viper）
│   ├── container/       # 依赖注入（dig）
│   ├── tracing/         # OpenTelemetry 链路追踪
│   └── utils/           # 工具函数
├── frontend/            # Vue3 前端
│   └── src/
│       ├── api/         # Axios API 封装
│       ├── views/       # 页面视图
│       ├── components/  # 公共组件
│       ├── stores/      # Pinia 状态管理
│       ├── router/      # Vue Router
│       └── i18n/        # 国际化（中/英/日/韩）
├── docreader/           # 文档解析微服务（Python）
│   ├── main.py          # gRPC 服务入口
│   ├── parser/          # 文档解析器（PDF/Word/图片等）
│   ├── ocr/             # PaddleOCR 集成
│   ├── splitter/        # 文本分块策略
│   └── proto/           # gRPC 协议定义
├── mcp-server/          # MCP 协议服务器（Python）
├── migrations/          # 数据库迁移脚本（golang-migrate）
├── config/              # 配置文件模板
├── helm/                # Kubernetes Helm Charts
├── docker/              # Dockerfile 集合
│   ├── Dockerfile.app       # Go 后端镜像
│   ├── Dockerfile.docreader # 文档解析镜像
│   ├── Dockerfile.rerank    # 重排序服务镜像
│   ├── Dockerfile.embedding # Embedding 服务镜像
│   └── Dockerfile.sandbox   # Agent Skills 沙箱镜像
├── scripts/             # 运维脚本
├── skills/              # Agent Skills 预置技能
├── docs/                # 项目文档
└── docker-compose.yml   # 主编排文件
```

---

## 四、核心技术栈

### 4.1 后端（Go）

| 组件 | 技术选型 |
|------|----------|
| 语言/版本 | Go 1.24 |
| Web 框架 | Gin v1.11 |
| ORM | GORM v1.30 |
| 数据库驱动 | PostgreSQL (pgvector) / SQLite |
| 任务队列 | Asynq v0.25（基于 Redis） |
| 依赖注入 | uber/dig v1.18 |
| 配置管理 | Viper v1.20 |
| JWT 认证 | golang-jwt/jwt v5 |
| 数据迁移 | golang-migrate v4 |
| MCP 协议 | mark3labs/mcp-go v0.43 |
| 链路追踪 | OpenTelemetry v1.38 |
| 日志 | logrus v1.9 |
| 分词 | yanyiwu/gojieba |

### 4.2 前端（Vue3）

| 组件 | 技术选型 |
|------|----------|
| 框架 | Vue 3.5 + TypeScript 5.8 |
| 构建 | Vite 7 |
| UI 库 | TDesign Vue Next v1.17 |
| 状态管理 | Pinia v3 |
| 路由 | Vue Router v4 |
| 国际化 | Vue I18n v11 |
| HTTP 客户端 | Axios v1.8 |
| Markdown | Marked v5 + Highlight.js v11 |
| 图表 | Mermaid v11 |

### 4.3 文档解析（Python）

| 组件 | 技术选型 |
|------|----------|
| 通信协议 | gRPC |
| OCR | PaddleOCR |
| PDF 解析 | PyMuPDF 等 |
| 多模态 | VLM（OpenAI/Ollama API） |

### 4.4 存储层

| 类型 | 支持选项 |
|------|----------|
| 主数据库 | PostgreSQL / MySQL |
| 向量数据库 | PostgreSQL(pgvector) / Elasticsearch 7/8 / Qdrant / Milvus / Weaviate |
| 图数据库 | Neo4j（GraphRAG） |
| 对象存储 | 本地 / MinIO / 腾讯云COS / 火山引擎TOS / AWS S3 |
| 缓存/队列 | Redis 7 |

---

## 五、RAG 数据流

系统处理一次知识库问答的完整 Pipeline 包含 9 个串行事件：

```
用户提问
    │
    ▼
[1] rewrite_query        ← 结合历史消息改写问题（调用 LLM）
    │
    ▼
[2] preprocess_query     ← 分词预处理（gojieba）
    │
    ▼
[3] chunk_search         ← 混合检索（向量检索 + 关键词BM25）
    │                       两轮搜索（改写问题 + 关键词序列）
    ▼
[4] chunk_rerank         ← 精细排序（BGE ReRank，可选）
    │
    ▼
[5] chunk_merge          ← 合并相邻区块（父子分块策略）
    │
    ▼
[6] filter_top_k         ← 保留 Top-K 最相关结果
    │
    ▼
[7] into_chat_message    ← 构建含检索结果的 Prompt
    │
    ▼
[8] chat_completion_stream ← 调用 LLM 流式生成答案
    │
    ▼
[9] stream_filter        ← 过滤 <think> 等内部标记
    │
    ▼
流式返回给前端（含引用来源）
```

---

## 六、文档处理流程

文件上传后的异步处理流程（通过 Redis/Asynq 消息队列）：

```
文件上传
    │
    ▼
存储至对象存储（MinIO/COS/本地）
    │
    ▼
创建异步任务（Asynq → Redis）
    │
    ▼
DocReader gRPC 服务解析文件
    ├── PDF → PyMuPDF 提取文本/表格/图片
    ├── Word → 结构化解析
    ├── 图片 → PaddleOCR / VLM Caption
    └── 文本 → 直接读取
    │
    ▼
文本分块（多策略：固定长度/语义/父子分块）
    │
    ▼
Embedding 模型向量化（BGE-M3 等）
    │
    ▼
写入向量数据库 + PostgreSQL
    │
    ▼（可选）
构建知识图谱（Neo4j GraphRAG）
```

---

## 七、Agent 架构

```
用户请求（Agent 模式）
    │
    ▼
ReACT Agent Loop
    ├── [Thought] 分析问题
    ├── [Action] 选择工具
    │   ├── 知识库检索工具（内置）
    │   ├── 网络搜索工具（DuckDuckGo/Bing/Google）
    │   ├── MCP 工具（外部扩展，stdio/HTTP/SSE）
    │   ├── 数据分析工具（CSV/Excel - DataSchema）
    │   └── Agent Skills（沙箱执行脚本）
    ├── [Observation] 观察结果
    └── [循环直到完成]
    │
    ▼
汇总生成综合报告
```

### Agent Skills 沙箱

```
Agent Skills 请求
    │
    ▼
沙箱模式判断
    ├── docker 模式 → docker run weknora-sandbox（推荐，完全隔离）
    ├── local 模式  → 本地执行（开发用）
    └── disabled 模式 → 禁用 Skills
```

---

## 八、MCP 协议集成

WeKnora 支持 MCP（Model Context Protocol）协议，提供三种传输方式：

| 传输方式 | 场景 |
|----------|------|
| **stdio** | 本地工具（uvx/npx 启动） |
| **HTTP Streamable** | 远程 HTTP 服务 |
| **SSE** | 服务器推送事件流 |

内置启动器支持 `uvx`（Python）和 `npx`（Node.js）两种运行时。

---

## 九、多租户与安全

- **租户隔离**：所有数据按 `tenant_id` 物理隔离，检索时严格过滤
- **共享空间**：可创建跨成员共享的知识库和 Agent，保持租户边界
- **认证机制**：JWT Token + API Key 双认证
- **数据加密**：AES-256 加密数据库中 API Key 等敏感字段
- **SSRF 防护**：内置 SSRF-safe HTTP 客户端
- **SQL 注入防护**：增强型 SQL 语句验证
- **注册控制**：`DISABLE_REGISTRATION` 开关控制新用户注册

---

## 十、可观测性

| 组件 | 作用 |
|------|------|
| Jaeger | 分布式链路追踪（OTLP/gRPC 协议） |
| OpenTelemetry | 追踪数据采集与导出 |
| Logrus | 结构化日志（支持动态日志级别） |
| 健康检查 | 每个服务均配置 Docker healthcheck |

---

## 十一、部署架构

### 11.1 Docker Compose（推荐）

```
宿主机
├── Nginx (前端:80)
├── Go App (后端:8080)
├── DocReader (gRPC:50051)
├── PostgreSQL (:5432)
├── Redis (:6379)
└── [可选] MinIO / Neo4j / Qdrant / Milvus / Weaviate / Jaeger / Ollama
```

所有服务通过 `WeKnora-network` Bridge 网络互通，数据持久化至 `./data/` 目录。

### 11.2 Kubernetes（Helm）

提供完整 Helm Charts（`helm/`），支持 Kubernetes 生产级部署，含 Neo4j GraphRAG 支持。

### 11.3 开发模式（本地）

```bash
make dev-start      # 启动基础设施（Docker）
make dev-app        # 本地运行 Go 后端（支持 Air 热重载）
make dev-frontend   # 本地运行前端（Vite HMR）
```

---

## 十二、构建产物

| 镜像名 | 说明 |
|--------|------|
| `wechatopenai/weknora-app` | Go 后端 |
| `wechatopenai/weknora-ui` | Vue3 前端 |
| `wechatopenai/weknora-docreader` | 文档解析服务 |
| `wechatopenai/weknora-sandbox` | Agent Skills 沙箱 |
