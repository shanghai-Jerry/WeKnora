# WeKnora 项目结构说明

## 项目概述

**WeKnora** 是一个基于 LLM 的智能知识管理与问答框架，专为企业级文档理解和语义检索设计。提供两种问答模式：
- **快速问答**（基于 RAG）
- **智能推理**（ReACT Agent），支持多源综合和复杂任务处理

- **版本**: 0.3.6
- **许可证**: MIT
- **官网**: https://weknora.weixin.qq.com
- **模块路径**: github.com/Tencent/WeKnora

---

## 技术栈

### 后端 (Go)
- **Go 版本**: 1.24.11
- **Web 框架**: Gin
- **ORM**: GORM (PostgreSQL)
- **依赖注入**: uber/dig
- **gRPC**: google.golang.org/grpc (与 docreader 通信)
- **认证**: JWT, API Key
- **任务队列**: Asynq (基于 Redis)
- **WebSocket**: gorilla/websocket
- **链路追踪**: OpenTelemetry + Jaeger
- **数据库迁移**: golang-migrate
- **配置管理**: Viper
- **API 文档**: Swagger (swag)

### 前端 (Vue 3)
- **框架**: Vue 3.5 + Composition API
- **构建工具**: Vite 7.2
- **UI 组件库**: TDesign Vue Next 1.17.2
- **状态管理**: Pinia 3.0.1
- **路由**: Vue Router 4.5
- **国际化**: Vue I18n 11.1 (支持 en, zh-CN, ja, ko)
- **Markdown 渲染**: Marked + highlight.js
- **图表**: Mermaid 11.4
- **HTTP 客户端**: Axios 1.8.4

### 数据库与存储
- **主数据库**: PostgreSQL 17 + pgvector (通过 ParadeDB)
- **缓存/会话**: Redis 7
- **知识图谱** (可选): Neo4j 6
- **对象存储**: 本地存储 / MinIO / AWS S3 / 腾讯云 COS / 火山引擎 TOS

### 向量数据库支持
1. PostgreSQL (pgvector，默认)
2. Elasticsearch v7/v8
3. Qdrant
4. Milvus v2
5. Weaviate v5

### LLM 提供商
OpenAI, DeepSeek, Qwen, 智谱 GLM, 腾讯 Hunyuan, Gemini, MiniMax, NVIDIA NIM, Novita AI, SiliconFlow, Ollama, OpenRouter, 火山引擎 Doubao 等 15+ 提供商

---

## 目录结构

```
WeKnora/
├── cmd/                          # 应用程序入口
│   ├── server/main.go            # 主后端服务器 (Gin HTTP)
│   ├── pipline/main.go           # 文档处理管道 (解析→分块→实体提取)
│   ├── extractor_entity/main.go  # 实体提取工具
│   └── download/duckdb/          # DuckDB 下载工具
│
├── internal/                     # 核心业务逻辑 (私有 Go 包)
│   ├── agent/                    # Agent 系统 (ReACT), 工具, Token 管理, 技能
│   │   ├── token/               # Token 估算与压缩
│   │   ├── tools/               # 内置工具 (知识查询, 网页抓取, MCP 等)
│   │   ├── memory/              # Agent 记忆管理
│   │   └── skills/              # Agent 技能执行
│   ├── application/              # 应用服务与仓储层
│   │   ├── repository/          # 数据访问层 (检索器, 记忆存储)
│   │   │   ├── retriever/      # 向量数据库检索器
│   │   │   └── memory/         # 知识图谱记忆 (Neo4j, SQLite)
│   │   └── service/            # 业务逻辑服务
│   │       ├── chat_pipeline/   # 聊天处理管道
│   │       ├── file/            # 文件处理服务
│   │       ├── llmcontext/      # LLM 上下文管理
│   │       └── memory/          # 记忆图谱服务
│   ├── handler/                 # HTTP 请求处理器 (Gin)
│   │   └── session/            # 聊天会话处理器 (流式, QA, agent)
│   ├── models/                  # AI 模型集成
│   │   ├── chat/               # 聊天模型客户端
│   │   ├── embedding/           # 嵌入模型客户端
│   │   ├── provider/            # LLM 提供商实现
│   │   ├── vlm/                # 视觉语言模型
│   │   ├── asr/                # 自动语音识别
│   │   └── rerank/             # 重排序模型
│   ├── im/                      # 即时通讯集成
│   │   ├── wecom/              # 企业微信适配器
│   │   ├── feishu/             # 飞书适配器
│   │   ├── slack/              # Slack 适配器
│   │   ├── telegram/           # Telegram 适配器
│   │   ├── dingtalk/           # 钉钉适配器
│   │   └── mattermost/         # Mattermost 适配器
│   ├── datasource/              # 外部数据源集成
│   │   └── connector/feishu/   # 飞书 Wiki/云盘连接器
│   ├── infrastructure/          # 基础设施组件
│   │   ├── chunker/            # 文本分块策略
│   │   ├── docparser/          # 文档解析 (Go 端)
│   │   ├── web_fetch/          # 网页内容抓取
│   │   └── web_search/         # 网页搜索提供商
│   ├── config/                  # 配置加载
│   ├── container/               # 依赖注入 (uber/dig)
│   ├── database/                # 数据库连接与迁移
│   ├── mcp/                     # MCP (模型上下文协议) 客户端/管理器
│   ├── middleware/               # HTTP 中间件 (认证, CORS, 日志, 追踪)
│   ├── types/                   # 共享类型定义与接口
│   ├── sandbox/                 # 代码沙箱执行环境
│   ├── stream/                  # 流式响应管理
│   ├── searchutil/              # 搜索工具
│   ├── logger/                  # 日志工具
│   ├── tracing/                 # OpenTelemetry 追踪
│   ├── router/                  # HTTP 路由定义
│   ├── runtime/                 # 运行时环境信息
│   ├── event/                   # 事件处理
│   ├── utils/                   # 通用工具
│   ├── common/                  # 公共共享代码
│   ├── assets/                  # 静态资源
│   └── errors/                  # 错误定义
│
├── frontend/                     # Vue 3 + Vite 前端
│   ├── src/
│   │   ├── views/               # 页面组件
│   │   │   ├── chat/           # 聊天界面
│   │   │   ├── knowledge/      # 知识库管理
│   │   │   ├── agent/          # Agent 配置
│   │   │   ├── auth/           # 认证页面
│   │   │   ├── settings/       # 系统设置
│   │   │   ├── organization/   # 组织管理
│   │   │   └── platform/       # 平台管理
│   │   ├── components/          # 可复用 Vue 组件
│   │   ├── api/                 # API 客户端模块
│   │   ├── stores/              # Pinia 状态管理
│   │   ├── composables/         # Vue composables
│   │   ├── hooks/               # 自定义 hooks
│   │   ├── types/               # TypeScript 类型定义
│   │   ├── utils/               # 前端工具
│   │   ├── i18n/                # 国际化
│   │   ├── router/              # Vue Router 配置
│   │   └── assets/              # 静态资源
│   ├── public/                  # 公共静态文件
│   ├── packages/                # 本地包 (xlsx)
│   ├── package.json             # Node.js 依赖
│   ├── vite.config.ts           # Vite 配置
│   └── tsconfig.json            # TypeScript 配置
│
├── docreader/                    # Python gRPC 文档解析服务
│   ├── main.py                  # gRPC 服务器入口
│   ├── parser/                  # 文档解析器
│   │   ├── doc_parser.py       # PDF 解析
│   │   ├── docx_parser.py      # Word 文档解析
│   │   ├── markdown_parser.py  # Markdown 解析
│   │   ├── excel_parser.py     # Excel 解析
│   │   ├── web_parser.py       # 网页解析
│   │   ├── image_parser.py     # 图片解析
│   │   └── registry.py         # 解析器注册表
│   ├── proto/                   # Protobuf 定义
│   ├── models/                  # 文档理解 ML 模型
│   ├── ocr/                     # OCR 功能
│   ├── splitter/                # 文本分割工具
│   └── config.py                # Docreader 配置
│
├── client/                       # WeKnora API 的 Go 客户端库
│   ├── agent.go                 # Agent 管理
│   ├── knowledge.go             # 知识库操作
│   ├── knowledgebase.go         # 知识库管理
│   ├── session.go               # 聊天会话管理
│   ├── faq.go                   # FAQ 操作
│   ├── model.go                 # 模型配置
│   ├── organization.go          # 组织操作
│   ├── message.go               # 消息处理
│   ├── chunk.go                 # 分块操作
│   └── tenant.go                # 租户管理
│
├── config/                       # 配置文件
│   ├── config.yaml              # 主配置文件
│   ├── config-org.yaml          # 组织特定配置
│   ├── builtin_agents.yaml      # 内置 Agent 定义
│   └── prompt_templates/        # LLM 提示词模板
│
├── migrations/                    # 数据库迁移
│   ├── mysql/                   # MySQL 特定迁移
│   ├── paradedb/                # ParadeDB (PostgreSQL + 向量搜索)
│   ├── sqlite/                  # SQLite 迁移
│   └── versioned/               # 版本化 SQL 迁移文件 (68+ 版本)
│
├── docker/                       # Docker 配置
│   ├── Dockerfile.app           # 主应用容器
│   ├── Dockerfile.docreader     # Docreader 服务容器
│   ├── Dockerfile.rerank        # 重排序服务容器
│   └── Dockerfile.sandbox      # 沙箱执行容器
│
├── scripts/                      # 开发与部署脚本
│   ├── dev.sh                   # 开发环境管理
│   ├── build_images.sh          # Docker 镜像构建
│   ├── migrate.sh               # 数据库迁移运行器
│   ├── quick-dev.sh             # 快速开发设置
│   ├── start_all.sh             # 启动所有服务
│   ├── check-env.sh            # 环境验证
│   └── get_version.sh           # 版本信息提取
│
├── mcp-server/                   # Python MCP (模型上下文协议) 服务器
│   ├── main.py                  # MCP 服务器入口
│   ├── weknora_mcp_server.py    # MCP 服务器实现
│   └── run.py                   # 服务器运行器
│
├── python-scripts/               # 附加 Python 脚本
│   ├── prompt/                  # 提示词模板与工具
│   ├── agent/                   # Agent 相关脚本
│   ├── event/                   # 事件处理脚本
│   ├── sql/                     # SQL 工具
│   ├── tools/                   # 工具脚本
│   ├── rerank_server_bge.py     # BGE 重排序服务器
│   └── split_pdf.py             # PDF 分割工具
│
├── skills/                       # 预加载的 Agent 技能
│   └── preloaded/
│       ├── citation-generator/   # 引用生成技能
│       ├── data-processor/       # 数据处理技能
│       ├── doc-coauthoring/      # 文档协作技能
│       └── document-analyzer/    # 文档分析技能
│
├── helm/                         # Kubernetes Helm 图表
│   ├── Chart.yaml               # Helm 图表定义
│   ├── values.yaml               # 默认配置值
│   └── templates/               # Kubernetes 清单
│       ├── app.yaml             # 应用部署
│       ├── docreader.yaml       # Docreader 部署
│       ├── frontend.yaml        # 前端部署
│       ├── postgres.yaml        # PostgreSQL 部署
│       ├── redis.yaml           # Redis 部署
│       ├── neo4j.yaml           # Neo4j 部署
│       ├── ingress.yaml         # Ingress 配置
│       └── secrets.yaml         # 密钥管理
│
├── docs/                         # 文档与图片
├── .env.example                  # 环境变量模板
├── docker-compose.yml            # 主 Docker Compose 配置
├── docker-compose.dev.yml       # 开发 Docker Compose
├── docker-compose.local.yml      # 本地部署配置
├── Makefile                      # 构建与任务自动化
├── go.mod                        # Go 模块依赖
├── go.sum                        # Go 依赖校验和
├── README.md                     # 主文档 (英文)
├── README_CN.md                  # 中文文档
├── README_JA.md                  # 日文文档
├── README_KO.md                  # 韩文文档
├── CHANGELOG.md                  # 版本变更日志
├── AGENTS.md                     # Agent 特定文档
├── SECURITY.md                   # 安全策略
├── LICENSE                       # MIT 许可证
├── .air.toml                     # Air (热重载) 配置
├── .golangci.yml                 # GolangCI-Lint 配置
└── VERSION                       # 版本文件 (0.3.6)
```

---

## 架构概览

```
┌─────────────────────────────────────────────────────────┐
│                   前端 (Vue 3)                          │
│              http://localhost:5173                      │
└────────────────────────────┬────────────────────────────┘
                             │ HTTP/REST API
                             ▼
┌─────────────────────────────────────────────────────────┐
│                后端 (Gin + Go)                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │  处理器层 (HTTP 端点, 中间件, 认证)              │  │
│  └──────────────────────┬───────────────────────────┘  │
│                         │                               │
│  ┌──────────────────────▼───────────────────────────┐  │
│  │  服务层 (聊天, 文件, agent, 记忆)               │  │
│  └──────────────────────┬───────────────────────────┘  │
│                         │                               │
│  ┌──────────────────────▼───────────────────────────┐  │
│  │  仓储层 (检索器, 记忆存储)                       │  │
│  └──────────────────────┬───────────────────────────┘  │
└─────────────────────────┼───────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
┌──────────────┐  ┌─────────────┐  ┌──────────────┐
│  PostgreSQL  │  │   Redis     │  │  向量数据库  │
│  (pgvector)  │  │  (Asynq)   │  │(Qdrant/Milvus│
│              │  │             │  │   /Weaviate) │
└──────────────┘  └─────────────┘  └──────────────┘
        │
        ▼
┌──────────────────────────────────────────────────────────┐
│              文档处理管道                                 │
│  ┌──────────┐    ┌──────────┐    ┌──────────────────┐  │
│  │  上传    │───▶│ Docreader│───▶│分块 + 嵌入向量  │  │
│  │  (HTTP)  │    │ (gRPC)   │    │  (向量数据库)   │  │
│  └──────────┘    └──────────┘    └──────────────────┘  │
└──────────────────────────────────────────────────────────┘
        │
        ▼
┌──────────────────────────────────────────────────────────┐
│              Agent 系统 (ReACT)                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐ │
│  │  工具    │  │  MCP     │  │  知识检索           │ │
│  │(内置)    │  │  服务器  │  │ (GraphRAG, BM25)   │ │
│  └──────────┘  └──────────┘  └──────────────────────┘ │
│  ┌─────────────────────────────────────────────────┐    │
│  │  技能 (沙箱执行)                                │    │
│  └─────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────┘
```

---

## 主要功能

### 1. 智能对话
- **快速问答模式**: 基于 RAG 的检索，响应快速
- **智能推理模式**: ReACT Agent，支持多步推理
- **工具调用**: 内置工具, MCP 工具, 网页搜索集成
- **对话策略**: 在线提示词编辑, 检索阈值调整
- **建议问题**: 基于知识库内容自动生成
- **多模态支持**: 图像处理的视觉语言模型 (VLM)
- **语音识别**: 音频文件的自动语音识别 (ASR)
- **思考模式**: 支持 LLM 思考/推理过程展示

### 2. 知识管理
- **知识库类型**: FAQ 和文档知识库
- **文档格式**: 支持 10+ 格式 (PDF, Word, Excel, PPT, 图片等)
- **数据源导入**: 飞书 Wiki/云盘自动同步 (增量/全量)
- **检索策略**: BM25 稀疏检索, 稠密检索, GraphRAG, 父子分块
- **标签管理**: 使用标签组织知识
- **知识搜索**: 语义搜索与直接聊天集成
- **文档预览**: 嵌入原始文件预览
- **文档摘要**: AI 生成的摘要

### 3. 集成与扩展
- **LLM 支持**: 15+ LLM 提供商
- **嵌入模型**: Ollama, BGE, GTE, OpenAI 兼容 API
- **向量数据库**: 5 种支持 (PostgreSQL, Elasticsearch, Qdrant, Milvus, Weaviate)
- **存储**: 5 种选项 (本地, MinIO, S3, TOS, COS)
- **IM 渠道**: 6 种集成 (企业微信, 飞书, Slack, Telegram, 钉钉, Mattermost)
- **网页搜索**: 4 种提供商 (DuckDuckGo, Bing, Google, Tavily)
- **MCP 支持**: 模型上下文协议，用于扩展 Agent 能力

### 4. 平台能力
- **部署**: 本地, Docker, Kubernetes (Helm)，支持离线部署
- **UI**: Web UI, RESTful API, Chrome 扩展
- **多租户**: 组织和租户隔离
- **认证**: JWT, API Key, OIDC (OpenID Connect)
- **任务管理**: 基于 Asynq 的异步任务，自动数据库迁移
- **模型管理**: 集中配置，每个知识库可选择模型
- **安全**: SSRF 防护, API 密钥加密 (AES-256-GCM), 沙箱执行
- **可观测性**: OpenTelemetry 追踪 (Jaeger), 结构化日志
- **国际化**: 多语言支持 (英, 简中, 日, 韩)

---

## 快速开始

```bash
# 克隆仓库
git clone https://github.com/Tencent/WeKnora.git
cd WeKnora

# 配置环境
cp .env.example .env
# 编辑 .env 设置你的配置

# Docker 启动 (最小核心服务)
docker compose up -d

# 启动所有功能
docker compose --profile full up -d

# 带 Neo4j 知识图谱
docker compose --profile neo4j up -d

# 开发模式 (热重载)
make dev-start    # 启动依赖服务
make dev-app      # 启动后端 (另一个终端)
make dev-frontend  # 启动前端 (另一个终端)

# 运行测试
go test -v ./...

# 生产构建
make build-prod

# 数据库迁移
make migrate-up
```

---

## 入口点总结

| 入口点 | 位置 | 用途 |
|--------|------|------|
| **主服务器** | `cmd/server/main.go` | 主要后端 HTTP 服务器 (Gin)，处理所有 API 请求 |
| **管道工具** | `cmd/pipline/main.go` | 文档处理管道: 解析 → 分块 → 提取实体/关系 |
| **实体提取器** | `cmd/extractor_entity/main.go` | 实体提取工具 (当前为空) |
| **Docreader 服务** | `docreader/main.py` | Python gRPC 服务，用于文档解析 (10+ 格式) |
| **MCP 服务器** | `mcp-server/main.py` | Python MCP 服务器，用于扩展 Agent 能力 |
| **前端** | `frontend/` | Vue 3 SPA，生产环境由 Nginx 提供服务 |

---

## 文档版本
- 创建时间: 2026-04-28
- 基于版本: WeKnora v0.3.6
- 作者: opencode AI assistant
