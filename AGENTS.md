# AGENTS.md

## Build Commands

```bash
# Start infrastructure + dev servers (fast dev mode - recommended)
make dev-start      # Start dependencies (postgres, redis, neo4j, docreader, etc.)
make dev-app        # Start backend (hot reload with Air)
make dev-frontend  # Start frontend (Vite hot reload)

# Or use the quick script
./scripts/quick-dev.sh
```

## Run Tests

```bash
go test -v ./...
```

## Database Migrations

```bash
make migrate-up                  # Apply migrations
make migrate-down                # Rollback
make migrate-create name=xxx       # Create new migration
make migrate-goto version=3      # Go to specific version
```

## Docker Services

```bash
# Start all services
docker compose up -d
# With profiles
docker-compose --profile full up -d        # All features
docker-compose --profile neo4j up -d    # With Neo4j knowledge graph
docker-compose --profile minio up -d    # With MinIO storage
```

## Code Quality

```bash
make fmt          # go fmt
make lint         # golangci-lint run
make docs         # Generate Swagger API docs (swag init)
```

## Key Entry Points

- Backend: `cmd/server/main.go`
- Docreader: `cmd/docreader/` (separate gRPC service)
- Frontend: `frontend/` (Vue 3 + Vite)
- Migrations: `internal/database/migration.go`

## Dependencies

- PostgreSQL (paradedb v0.21.4-pg17 for vector search)
- Redis (session/stream management)
- Neo4j (knowledge graph - optional)
- gRPC docreader (document parsing)
- Multiple vector DB options: Elasticsearch, Qdrant, Milvus, Weaviate

## Environment

Copy `.env.example` to `.env` and configure. Required variables are documented in the comments.

## Project Structure

```
WeKnora/
├── cmd/           # Entry points (server, docreader, pipeline)
├── client/        # Go client library
├── config/        # YAML config
├── docker/        # Dockerfiles
├── docreader/     # Document parsing service (gRPC)
├── frontend/      # Vue 3 frontend
├── internal/     # Core business logic
├── migrations/    # DB migrations
├── skills/        # Agent skills
└── scripts/      # Dev/deploy scripts
```

## Notable Conventions

- Uses Air for backend hot reload (configured in `.air.toml`)
- Go 1.24.11 with CGO required
- Docreader runs as separate gRPC service on port 50051
- Uses `golang-migrate` for database migrations
- Skills run in Docker sandbox by default for security isolation