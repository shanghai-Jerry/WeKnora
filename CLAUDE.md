# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WeKnora is an LLM-powered knowledge management and Q&A framework. It provides two modes: Quick Q&A (RAG-based retrieval) and Intelligent Reasoning (ReACT Agent with tool calling, MCP, and web search). The monorepo contains a Go backend, a Vue 3 frontend, a Python document parsing service (docreader), and a Python MCP server.

## Technology Stack

- **Backend**: Go 1.24.11, Gin, GORM (PostgreSQL), uber/dig (DI), golang-migrate, Asynq (Redis task queue), OpenTelemetry
- **Frontend**: Vue 3.5 + Vite 7.2, TDesign Vue Next, Pinia, Vue I18n, Mermaid, Marked
- **Docreader**: Python 3.10+ gRPC service for document parsing, OCR, and splitting (uv-managed)
- **MCP Server**: Python MCP server (`mcp-server/`) exposing WeKnora as an MCP tool
- **Databases**: PostgreSQL 17 + pgvector (default), with optional vector stores (Elasticsearch, Qdrant, Milvus, Weaviate) and Neo4j for knowledge graphs
- **Object Storage**: Local, MinIO, AWS S3, Tencent COS, Volcengine TOS

## Common Commands

### Backend (Go)

```bash
# Build the server binary
make build

# Run the server (builds first)
make run

# Run all Go tests
make test

# Run a specific test package or single test
go test -v ./internal/agent/...
go test -v ./internal/agent/ -run TestEngine

# Format and lint
make fmt
make lint          # requires golangci-lint

# Install Go dependencies
make deps

# Generate Swagger API docs
make docs          # requires swag (make install-swagger)
```

### Frontend

```bash
cd frontend
npm install
npm run dev        # Vite dev server
npm run build      # Production build
npm run type-check # Vue TSC type check
```

### Docreader (Python)

```bash
cd docreader
uv sync            # install dependencies from uv.lock
uv run python main.py --config config.yaml
```

### Development Environment

```bash
# Start infrastructure dependencies only (Postgres, Redis, MinIO, etc.)
make dev-start

# Start backend locally with hot reload (requires dev-start first)
make dev-app       # uses Air; see .air.toml

# Start frontend locally (requires dev-start first)
make dev-frontend

# Stop dev infrastructure
make dev-stop

# View dev logs
make dev-logs
```

### Database Migrations

```bash
# Apply all up migrations
make migrate-up

# Rollback one migration
make migrate-down

# Create a new migration
make migrate-create name=add_new_column

# Force a specific version
make migrate-force version=4
```

### Docker

```bash
# Build all images from source
make build-images

# Build individual images
make build-images-app
make build-images-docreader
make build-images-frontend

# Start all services via Docker Compose
make start-all

# Stop all services
make stop-all
```

## Architecture

### Application Entry Points

- `cmd/server/main.go` — Main HTTP server (Gin) with graceful shutdown, SO_REUSEPORT for hot reload, and OpenTelemetry tracing.
- `cmd/download/duckdb/duckdb.go` — DuckDB binary download utility.
- `docreader/main.py` — gRPC document parsing service (port 50051 by default).
- `mcp-server/main.py` — MCP server exposing WeKnora capabilities.

### Dependency Injection

The backend uses `go.uber.org/dig` for dependency injection. All services, repositories, and handlers are wired in `internal/container/container.go`. When adding new services or repositories, register their constructors in `BuildContainer`. The container exposes a `ResourceCleaner` interface for graceful shutdown.

### Layer Structure

- **`internal/handler/`** — HTTP handlers (Gin). Organized by domain (auth, knowledge, tenant, session, etc.). `session/` contains chat/QA/agent streaming handlers.
- **`internal/application/service/`** — Business logic services. Key sub-packages:
  - `chat_pipeline/` — Chat request processing pipeline.
  - `llmcontext/` — LLM context window management.
  - `memory/` — Knowledge graph memory operations.
  - `file/` — File processing.
- **`internal/application/repository/`** — Data access layer.
  - `retriever/` — Vector DB retrievers (postgres, elasticsearch v7/v8, qdrant, milvus, weaviate, sqlite).
  - `memory/` — Graph memory stores (neo4j, sqlite).
- **`internal/models/`** — LLM client abstractions.
  - `chat/` — Chat model clients.
  - `embedding/` — Embedding model clients.
  - `provider/` — Provider-specific implementations (OpenAI, DeepSeek, Qwen, etc.).
  - `vlm/`, `asr/`, `rerank/` — Vision, speech, and reranking models.
- **`internal/agent/`** — ReACT agent engine, tool definitions, token management, skills, and memory.
  - `tools/` — Built-in tools (knowledge query, web fetch, final_answer, etc.).
  - `skills/` — Agent skill execution with sandboxing.
- **`internal/datasource/`** — Data source connectors and sync scheduler. Includes `connector/` for implementations like Feishu and MySQL.
- **`internal/im/`** — IM integrations: WeCom, Feishu, Slack, Telegram, DingTalk, Mattermost.
- **`internal/mcp/`** — MCP client integration for external tool servers.
- **`internal/stream/`** — Streaming abstraction over WebSocket and SSE.
- **`internal/router/`** — Route registration.
- **`internal/middleware/`** — Auth, logging, recovery, language, tracing.

### Configuration

Configuration is loaded from `config/config.yaml` (mounted in Docker) via `internal/config/config.go`. Environment variables in `.env` are consumed by Docker Compose. Key config sections: `conversation`, `server`, `knowledge_base`, `models`, `vector_database`, `docreader`, `stream_manager`, `web_search`, `im`, `oidc_auth`.

### Database & Migrations

- Uses `golang-migrate` for schema migrations. Migration files are in `migrations/versioned/`.
- GORM models are defined in `internal/types/`.
- ParadeDB (PostgreSQL + pgvector) is the default vector store.

### Agent & Skills

- Agent runs a ReACT loop in `internal/agent/engine.go`.
- Tools are registered in `internal/agent/tools/` and auto-discovered.
- Skills are stored in `skills/` (preloaded) and executed in sandboxed Docker containers by default.
- Agent memory (short-term) is stored in Neo4j or SQLite depending on configuration.

## Testing

- Go tests use `stretchr/testify` (`assert`, `require`).
- Tests are scattered alongside source files (`*_test.go`).
- To run tests for a specific package: `go test -v ./internal/agent/...`
- To run a single test: `go test -v ./internal/agent/ -run TestEngine`
- Docreader tests: `cd docreader && uv run pytest`

## Development Workflow (from AGENTS.md)

Before significant changes, follow the planning workflow documented in `AGENTS.md`:

1. Check `docs/requirements/` for an existing requirement document. If none exists, search the codebase, then create one.
2. Create a plan at `plans/{feature-name}-{date}.md` with affected components, implementation steps, and testing strategy.
3. Get user approval before executing.
4. After completion, update `docs/CHANGELOG.md` and `docs/ROADMAP.md` if applicable.

## Notable Conventions

- **CGO is required** due to SQLite, DuckDB, and tokenization bindings. Build flags include `-Wno-deprecated-declarations` and `-Wl,-no_warn_duplicate_libraries`.
- **Hot reload**: Air is used for backend development; see `.air.toml`.
- **gRPC**: The backend communicates with docreader over gRPC. Proto files are in `docreader/proto/`.
- **LLM provider compatibility**: New providers are added in `internal/models/provider/` by implementing the chat/embedding/VLM interfaces.
- **Vector store compatibility**: New retrievers are added under `internal/application/repository/retriever/` and wired in `internal/container/container.go`.
- **IM adapters**: New IM channels are added under `internal/im/{platform}/` with a handler registered in the router.
- **API docs**: Swagger annotations are in handler files. Run `make docs` to regenerate.
