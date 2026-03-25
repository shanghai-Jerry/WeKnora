# WeKnora 项目说明文档

> 版本：v0.3.3 | 协议：MIT License | 语言：Go / Python / TypeScript

---

## 一、项目简介

**WeKnora** 是腾讯开源的企业级 LLM 驱动文档理解与语义检索框架，采用 RAG（检索增强生成）范式，为复杂异构文档提供深度理解和智能问答能力。

- **官方网站**：https://weknora.weixin.qq.com
- **GitHub**：https://github.com/Tencent/WeKnora
- **微信对话开放平台**：https://chatbot.weixin.qq.com
- **当前版本**：v0.3.3
- **开源协议**：MIT License

---

## 二、核心特性

### 🤖 Agent 模式
支持 ReACT Agent，可调用内置知识库检索工具、MCP 外部工具和网络搜索引擎（DuckDuckGo / Bing / Google），通过多轮迭代和反思生成综合报告。

### 🔍 精准理解
从 PDF、Word、图片等多种格式中提取结构化内容，转换为统一语义视图，支持 OCR 文字识别和多模态图像理解。

### 📚 多类型知识库
支持 **FAQ 知识库**和**文档知识库**两种类型，提供拖拽上传、文件夹导入、URL 导入、标签管理和在线录入等多种方式。

### 🔧 灵活扩展
从解析、Embedding 到检索、生成，所有组件均解耦，支持插拔替换。向量数据库支持 5 种，对象存储支持 5 种。

### ⚡ 高效混合检索
结合关键词（BM25）、向量检索和知识图谱（GraphRAG），支持跨知识库检索和父子分块策略。

### 🌐 网络搜索
支持多种搜索引擎扩展，内置 DuckDuckGo，可扩展接入 Bing、Google 等。

### 🔌 MCP 工具集成
通过 MCP（Model Context Protocol）协议扩展 Agent 能力，支持 stdio、HTTP Streamable、SSE 三种传输方式。

### 🔒 安全可控
支持本地私有化部署，数据不出企业网络，内置 SSRF 防护、SQL 注入防护和 AES-256 加密。

### 🏢 共享空间
支持创建组织级共享空间，成员邀请管理，跨成员共享知识库和 Agent，且保持租户数据隔离。

### 🧩 Agent Skills
预置技能库 + Docker 沙箱隔离执行，安全地扩展 Agent 的代码执行和分析能力。

---

## 三、功能矩阵

| 功能模块 | 状态 | 说明 |
|----------|------|------|
| Agent 模式（ReACT） | ✅ | 支持工具调用、多轮迭代、反思推理 |
| 知识库类型 | ✅ | FAQ 知识库 + 文档知识库 |
| 文档格式 | ✅ | PDF / Word / TXT / Markdown / 图片（OCR/Caption） |
| 模型管理 | ✅ | 集中配置，多租户共享内置模型 |
| Embedding 模型 | ✅ | BGE / GTE 等本地和云端 API |
| 向量数据库 | ✅ | PostgreSQL(pgvector) / ES / Qdrant / Milvus / Weaviate |
| 检索策略 | ✅ | BM25 / 向量检索 / GraphRAG |
| LLM 支持 | ✅ | Qwen / DeepSeek / 任意 OpenAI 兼容 API |
| 思考模式 | ✅ | 支持 LLM 思维链，自动过滤 `<think>` 标记 |
| 对话策略 | ✅ | 模型选择、检索阈值、Prompt 配置 |
| 网络搜索 | ✅ | DuckDuckGo / Google / Bing |
| MCP 工具 | ✅ | uvx/npx 启动器，三种传输方式 |
| 问题改写 | ✅ | 结合历史上下文自动改写 |
| 端到端测试 | ✅ | 检索命中率 / BLEU / ROUGE 指标评估 |
| 共享空间 | ✅ | 成员邀请 + 跨成员共享知识库 |
| Agent Skills | ✅ | 沙箱隔离执行（Docker/Local/Disabled） |
| Helm 部署 | ✅ | Kubernetes 完整 Helm Charts |
| 数据库自动迁移 | ✅ | 版本升级自动执行 schema 迁移 |
| 国际化 | ✅ | 中文 / English / 日本語 / 한국어 |
| 知识图谱 | ✅ | Neo4j GraphRAG（可选） |
| 知识搜索入口 | ✅ | 语义检索，可将结果引入对话 |
| 文档预览 | ✅ | 内嵌文档预览，支持原始文件查看 |
| Mermaid 渲染 | ✅ | 对话中渲染图表，支持全屏/缩放/导出 |
| 批量会话管理 | ✅ | 批量删除会话 |
| 父子分块 | ✅ | 层级化分块策略，提升检索精度 |

---

## 四、应用场景

| 场景 | 应用示例 | 核心价值 |
|------|----------|----------|
| **企业知识管理** | 内部文档检索、政策问答、操作手册搜索 | 提升知识发现效率，降低培训成本 |
| **学术研究分析** | 论文检索、研究报告分析、文献整理 | 加速文献综述，辅助研究决策 |
| **产品技术支持** | 产品手册问答、技术文档搜索、故障排查 | 提升客服质量，减轻支持负担 |
| **法律合规审查** | 合同条款检索、法规政策搜索、案例分析 | 提升合规效率，降低法律风险 |
| **医疗知识辅助** | 医学文献检索、诊疗指南搜索、病例分析 | 支持临床决策，提升诊断质量 |

---

## 五、技术栈概览

### 后端
- **语言**：Go 1.24
- **Web 框架**：Gin + Swagger
- **ORM**：GORM（PostgreSQL / SQLite）
- **任务队列**：Asynq（基于 Redis）
- **依赖注入**：uber/dig
- **配置**：Viper

### 前端
- **框架**：Vue 3.5 + TypeScript 5.8
- **构建工具**：Vite 7
- **UI 组件库**：TDesign Vue Next
- **状态管理**：Pinia
- **国际化**：Vue I18n（支持中/英/日/韩）

### 文档解析
- **通信**：Python gRPC
- **OCR**：PaddleOCR
- **多模态**：VLM 接口（OpenAI/Ollama 兼容）

### 基础设施
- **容器编排**：Docker Compose + Kubernetes Helm
- **可观测性**：Jaeger + OpenTelemetry
- **代理**：Nginx

---

## 六、快速开始

### 前置条件

- [Docker](https://www.docker.com/) 已安装
- [Docker Compose](https://docs.docker.com/compose/) 已安装
- [Git](https://git-scm.com/) 已安装

### 标准部署

```bash
# 1. 克隆仓库
git clone https://github.com/Tencent/WeKnora.git
cd WeKnora

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env，修改数据库密码、AES 密钥等必要配置

# 3. 启动所有服务
./scripts/start_all.sh
# 或
make start-all
```

### 按需启动

```bash
# 最小化核心服务
docker compose up -d

# 全功能（含 MinIO、Neo4j、Jaeger、Ollama 等）
docker compose --profile full up -d

# 按需组合
docker compose --profile neo4j --profile minio up -d
```

### 访问地址

| 服务 | 地址 |
|------|------|
| Web UI | http://localhost |
| 后端 API | http://localhost:8080 |
| Swagger 文档 | http://localhost:8080/swagger/index.html |
| Jaeger 追踪 | http://localhost:16686 |
| MinIO 控制台 | http://localhost:9001 |
| Neo4j Browser | http://localhost:7474 |

首次访问会自动跳转注册/登录页，注册账号后创建知识库即可开始使用。

---

## 七、开发指南

### 快速开发模式（推荐）

无需重建 Docker 镜像，基础设施运行于 Docker，代码在本地运行：

```bash
# 终端 1：启动基础设施（PostgreSQL、Redis、DocReader 等）
make dev-start

# 终端 2：启动 Go 后端（支持 Air 热重载）
make dev-app

# 终端 3：启动前端（Vite HMR 热重载）
make dev-frontend
```

开发环境地址：
- 前端开发服务器：http://localhost:5173
- 后端 API：http://localhost:8080

### 常用命令

```bash
make build           # 构建 Go 二进制
make test            # 运行测试
make lint            # 代码检查（golangci-lint）
make fmt             # 格式化代码
make docs            # 生成 Swagger API 文档
make migrate-up      # 执行数据库迁移
make migrate-down    # 回滚数据库迁移
make docker-build-all  # 构建所有 Docker 镜像
```

### 代码规范

- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化代码
- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范

提交示例：
```
feat: 添加文档批量上传功能
fix: 修复向量检索精度问题
docs: 更新 API 文档
refactor: 重构文档解析模块
```

---

## 八、环境变量说明

以下为 `.env` 关键配置项：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `GIN_MODE` | `release` | Gin 运行模式（debug/release） |
| `DB_DRIVER` | `postgres` | 主数据库类型 |
| `DB_USER/PASSWORD/NAME` | — | 数据库连接信息 |
| `RETRIEVE_DRIVER` | `postgres` | 向量存储类型 |
| `STORAGE_TYPE` | `local` | 文件存储类型（local/minio/cos/tos/s3） |
| `STREAM_MANAGER_TYPE` | `redis` | 流处理后端（memory/redis） |
| `REDIS_PASSWORD` | — | Redis 密码 |
| `OLLAMA_BASE_URL` | `http://host.docker.internal:11434` | Ollama 服务地址 |
| `ENABLE_GRAPH_RAG` | `false` | 是否启用知识图谱（GraphRAG） |
| `JWT_SECRET` | — | JWT 签名密钥 |
| `SYSTEM_AES_KEY` | — | AES-256 加密密钥（32字节） |
| `TENANT_AES_KEY` | — | 租户 AES 加密密钥（32字节） |
| `DISABLE_REGISTRATION` | `false` | 禁止新用户注册 |
| `WEKNORA_SANDBOX_MODE` | `docker` | Agent Skills 沙箱模式 |
| `CONCURRENCY_POOL_SIZE` | `5` | Embedding 并发数 |
| `MAX_FILE_SIZE_MB` | `50` | 最大文件上传大小（MB） |

---

## 九、API 文档

WeKnora 提供完整的 RESTful API，通过 Swagger 生成文档：

- **Swagger UI**：启动服务后访问 `http://localhost:8080/swagger/index.html`
- **API 文档目录**：`docs/api/`
- **Swagger JSON**：`docs/swagger.json`
- **Swagger YAML**：`docs/swagger.yaml`

支持 API Key 认证（请求头：`x-api-key: sk-...`）。

---

## 十、MCP 服务器

WeKnora 提供独立的 MCP Server，允许外部 AI 助手通过 MCP 协议访问 WeKnora 知识库。

### 配置方式

```json
{
  "mcpServers": {
    "weknora": {
      "command": "python",
      "args": ["path/to/WeKnora/mcp-server/run_server.py"],
      "env": {
        "WEKNORA_API_KEY": "sk-your-api-key",
        "WEKNORA_BASE_URL": "http://your-weknora-address/api/v1"
      }
    }
  }
}
```

也可通过 pip 安装：

```bash
pip install weknora-mcp-server
python -m weknora-mcp-server
```

---

## 十一、Kubernetes 部署

使用 Helm Charts 部署到 Kubernetes：

```bash
# 安装
helm install weknora ./helm

# 自定义配置
helm install weknora ./helm -f my-values.yaml
```

Helm Charts 位于 `helm/` 目录，含完整的 `values.yaml` 配置文件。

---

## 十二、版本历史摘要

| 版本 | 主要特性 |
|------|----------|
| **v0.3.3** | 父子分块策略、知识库置顶、Fallback 响应、Docker 改进 |
| **v0.3.2** | 知识搜索入口、文档预览、Milvus 支持、Mermaid 渲染、Weaviate 支持 |
| **v0.3.0** | 共享空间、Agent Skills（沙箱）、自定义 Agent、Thinking 模式、Helm Chart |
| **v0.2.0** | Agent 模式、多类型知识库、MCP 工具集成、网络搜索、异步任务队列 |
| **v0.1.3** | 登录认证、安全加固 |

完整更新日志见 [CHANGELOG.md](./CHANGELOG.md)，产品规划见 [docs/ROADMAP.md](./docs/ROADMAP.md)。

---

## 十三、相关文档

| 文档 | 路径 |
|------|------|
| 架构文档 | [arch.md](./arch.md) |
| 开发指南 | [docs/开发指南.md](./docs/开发指南.md) |
| 知识图谱配置 | [docs/开启知识图谱功能.md](./docs/开启知识图谱功能.md) |
| 共享空间说明 | [docs/共享空间说明.md](./docs/共享空间说明.md) |
| Agent Skills | [docs/agent-skills.md](./docs/agent-skills.md) |
| MCP 功能说明 | [docs/MCP功能使用说明.md](./docs/MCP功能使用说明.md) |
| 其他向量数据库 | [docs/使用其他向量数据库.md](./docs/使用其他向量数据库.md) |
| 常见问题 | [docs/QA.md](./docs/QA.md) |
| API 文档 | [docs/api/](./docs/api/) |
| 产品路线图 | [docs/ROADMAP.md](./docs/ROADMAP.md) |

---

## 十四、贡献指南

欢迎社区贡献！以下方式均可参与：

- 🐛 **Bug 修复**：发现并修复系统缺陷
- ✨ **新功能**：提出并实现新能力
- 📚 **文档**：完善项目文档
- 🧪 **测试**：编写单元和集成测试
- 🎨 **UI/UX**：改善用户界面和体验

### 贡献流程

1. Fork 项目到个人账户
2. 创建功能分支：`git checkout -b feature/amazing-feature`
3. 提交代码：`git commit -m 'feat: Add amazing feature'`
4. 推送分支：`git push origin feature/amazing-feature`
5. 创建 Pull Request，详细描述改动内容

---

## 十五、安全说明

> ⚠️ **重要提示**：从 v0.1.3 起，WeKnora 包含登录认证功能。生产环境强烈建议：

- 在内网/私有网络中部署，避免直接暴露于公网
- 配置防火墙规则和访问控制
- 定期更新到最新版本以获取安全补丁
- 修改默认密码（数据库、Redis、MinIO 等）
- 使用强随机值设置 `JWT_SECRET`、`SYSTEM_AES_KEY`、`TENANT_AES_KEY`

如发现安全漏洞，请参考 [SECURITY.md](./SECURITY.md) 进行负责任披露。
