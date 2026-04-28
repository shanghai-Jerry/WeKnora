# Go Database Migration 实现说明

## 概述

本项目使用 `golang-migrate/migrate/v4` 库实现数据库 schema 版本管理，通过 `migrations/` 目录下的版本化 SQL 文件控制表结构变更，支持 PostgreSQL（主库，生产环境）和 SQLite3（可选，本地开发）两种数据库，集成于应用启动流程与开发工具链中。

### 版本信息

| 项目 | 版本/路径 |
|------|-----------|
| golang-migrate 核心库 | `github.com/golang-migrate/migrate/v4` |
| golang-migrate CLI 工具 | `github.com/golang-migrate/migrate/v4/cmd/migrate` |
| PostgreSQL 驱动 | `github.com/golang-migrate/migrate/v4/database/postgres` |
| SQLite3 驱动 | `github.com/golang-migrate/migrate/v4/database/sqlite3` |
| 文件源驱动 | `github.com/golang-migrate/migrate/v4/source/file` |
| 迁移文件目录 | `migrations/versioned/`（PostgreSQL）、`migrations/sqlite/`（SQLite3） |
| 当前最新版本 | `000032`（截至文档编写时） |

---

## 工具选型：golang-migrate

| 项目 | 说明 |
|------|------|
| 官方库 | `github.com/golang-migrate/migrate/v4` |
| 支持的数据库驱动 | PostgreSQL（`_ "github.com/golang-migrate/migrate/v4/database/postgres"`）、SQLite3（`_ "github.com/golang-migrate/migrate/v4/database/sqlite3"`） |
| 迁移源 | 本地文件系统（`_ "github.com/golang-migrate/migrate/v4/source/file"`，使用 `file://` 协议） |
| 核心能力 | 版本化迁移、脏状态检测、正向/回滚迁移、版本跳转、多数据库适配 |

---

## 迁移文件结构

### 目录规划

```
migrations/
├── versioned/          # PostgreSQL 迁移文件（主目录，默认使用）
│   ├── 000000_init.up.sql
│   ├── 000000_init.down.sql
│   ├── 000001_agent.up.sql
│   ├── 000001_agent.down.sql
│   └── ...（按版本号递增，当前最新约 000032）
├── sqlite/             # SQLite 迁移文件（当 DSN 前缀为 sqlite3:// 时生效）
│   ├── 000000_init.up.sql
│   └── ...
├── paradedb/          # ParadeDB 专用迁移（向量搜索扩展场景）
└── mysql/             # MySQL 初始化文件（未启用）
```

### 命名规范

- **版本号**：6 位数字，从 `000000` 开始递增，保持前导零
- **文件后缀**：
  - `.up.sql`：正向迁移脚本（升级用）
  - `.down.sql`：回滚迁移脚本（降级用）
- **示例**：`000032_add_message_pipeline_stages.up.sql`

---

## 核心实现：internal/database/migration.go

### 文件基本信息

- 文件路径：`internal/database/migration.go`
- 总行数：248 行
- 职责：封装 `golang-migrate` 逻辑，提供迁移执行、状态查询、脏状态恢复能力

### 关键函数与结构体

| 函数/结构体 | 行号 | 说明 |
|------------|------|------|
| `CachedMigrationVersion() (uint, bool, bool)` | 26-28 | 获取应用启动时缓存的迁移版本（version, dirty, ok） |
| `RunMigrations(dsn string) error` | 40-42 | 基础迁移入口，默认不启用自动脏状态恢复 |
| `MigrationOptions` | 45-49 | 迁移配置选项（当前仅支持 `AutoRecoverDirty`） |
| `RunMigrationsWithOptions(dsn string, opts MigrationOptions) error` | 52-180 | 带自定义选项的迁移执行核心函数 |
| `recoverFromDirtyState(ctx, m, dirtyVersion) error` | 184-221 | 脏状态自动恢复函数 |
| `GetMigrationVersion() (uint, bool, error)` | 224-248 | 主动查询当前数据库迁移版本（重新创建 migrate 实例） |

### 核心执行流程（`RunMigrationsWithOptions`）

#### 1. 初始化迁移实例（行 57-66）

```go
migrationsPath := "file://migrations/versioned"
if strings.HasPrefix(dsn, "sqlite3://") {
    migrationsPath = "file://migrations/sqlite"
}
m, err := migrate.New(migrationsPath, dsn)
```

根据 DSN 前缀自动选择对应数据库的迁移目录，创建 `migrate.Migrate` 实例。

#### 2. 检查当前迁移状态（行 70-80）

```go
oldVersion, oldDirty, versionErr := m.Version()
if versionErr == migrate.ErrNilVersion {
    logger.Infof(ctx, "Database has no migration history, will start from version 0")
} else {
    logger.Infof(ctx, "Current migration version: %d, dirty: %v", oldVersion, oldDirty)
}
```

- `ErrNilVersion`：数据库无迁移历史（首次初始化）
- `dirty=true`：上次迁移中断，数据库处于不一致状态

#### 3. 脏状态处理（行 83-113）

```go
if oldDirty {
    if opts.AutoRecoverDirty {
        // 自动恢复：强制回退到上一版本，清除脏标记后重试
        recoverFromDirtyState(ctx, m, oldVersion)
    } else {
        // 返回错误，提示用户手动执行 force 命令
        return fmt.Errorf("database is in dirty state at version %d...", oldVersion)
    }
}
```

#### 4. 执行正向迁移（行 117）

```go
if err := m.Up(); err != nil && err != migrate.ErrNoChange {
    // 处理迁移失败逻辑（含迁移过程中变为脏状态的处理）
}
```

`m.Up()` 按版本号顺序执行所有待处理的 `.up.sql` 文件；`ErrNoChange` 表示数据库已是最新版本，属于正常情况。

#### 5. 缓存版本信息（行 167）

```go
setMigrationVersion(version, dirty)
```

使用 `sync.Once` 确保版本信息仅缓存一次，供全局通过 `CachedMigrationVersion()` 读取，避免重复查询数据库。

### 脏状态恢复逻辑（`recoverFromDirtyState`）

| 场景 | 处理逻辑 |
|------|----------|
| 脏状态版本为 `0`（初始迁移失败） | 强制设置版本为 `-1`，清除迁移状态，允许重新执行初始迁移（要求初始迁移脚本使用 `IF NOT EXISTS` 等幂等语法） |
| 脏状态版本 > `0` | 强制回退到 `dirtyVersion - 1`，清除脏标记，后续重新执行失败的迁移 |

---

## 项目集成方式

### 1. 应用启动时自动执行

入口文件：`cmd/server/main.go`

```go
// 默认启动方式：不自动恢复脏状态，出现脏状态则报错退出
database.RunMigrations(dsn)

// 可选：启用自动脏状态恢复（适合开发环境）
database.RunMigrationsWithOptions(dsn, database.MigrationOptions{AutoRecoverDirty: true})
```

应用启动时优先执行迁移，完成后再初始化其他组件，确保数据库 schema 与代码版本匹配。

### 2. 开发工具链集成（Makefile）

文件路径：`Makefile`（行 165-197），所有迁移命令封装为 Make 目标：

```makefile
# 执行所有待处理迁移
migrate-up:
	./scripts/migrate.sh up

# 回滚1个版本
migrate-down:
	./scripts/migrate.sh down

# 查看当前迁移版本
migrate-version:
	./scripts/migrate.sh version

# 创建新迁移文件（需指定 name 参数）
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required"; \
		echo "Usage: make migrate-create name=your_migration_name"; \
		exit 1; \
	fi
	./scripts/migrate.sh create $(name)

# 强制设置迁移版本（用于恢复脏状态）
migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-force version=4"; \
		exit 1; \
	fi
	./scripts/migrate.sh force $(version)

# 跳转到指定版本
migrate-goto:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-goto version=3"; \
		exit 1; \
	fi
	./scripts/migrate.sh goto $(version)
```

### 3. 迁移脚本详解（scripts/migrate.sh）

文件：`scripts/migrate.sh`（共 122 行）

#### 核心逻辑

| 部分 | 行号 | 说明 |
|------|------|------|
| 加载环境变量 | 8-14 | 从项目根目录 `.env` 文件加载数据库配置 |
| 默认参数 | 17-24 | `DB_HOST=localhost`、`DB_PORT=5432`、`DB_USER=postgres`、`DB_PASSWORD=postgres`、`DB_NAME=WeKnora`，迁移目录默认为 `migrations/versioned` |
| 检查 migrate 工具 | 27-31 | 检查 `migrate` CLI 是否安装，未安装则提示安装命令 |
| 构建数据库连接 URL | 33-60 | 优先使用 `DB_URL` 环境变量，否则从组件拼接；自动处理 `sslmode` 和特殊字符编码 |
| 命令分发 | 63-120 | 根据第一个参数执行 `up`、`down`、`create`、`version`、`force`、`goto` |

#### 关键特性

1. **环境变量支持**（行 36-60）：
   - 如果已设置 `DB_URL`，直接使用（自动确保 `sslmode=disable`）
   - 否则从 `DB_HOST`、`DB_PORT` 等组件构建
   - 密码中的特殊字符通过 Python `urllib.parse.quote` 编码

2. **创建迁移文件**（行 79-90）：
   ```bash
   migrate create -ext sql -dir ${MIGRATIONS_DIR} -seq $2
   ```
   使用 `-seq` 参数生成顺序版本号（6 位数字，如 `000033`）

3. **强制版本处理**（行 95-106）：
   ```bash
   env migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" force -- "$VERSION"
   ```
   使用 `env` 命令避免 shell 将 `-1` 解析为参数

#### 使用示例

```bash
# 查看帮助
./scripts/migrate.sh

# 执行迁移
./scripts/migrate.sh up

# 查看版本
./scripts/migrate.sh version

# 创建迁移（生成 000033_xxx.up.sql 和 000033_xxx.down.sql）
./scripts/migrate.sh create add_user_email

# 强制设置版本（恢复脏状态）
./scripts/migrate.sh force 4

# 跳转到指定版本
./scripts/migrate.sh goto 3
```

---

## 迁移文件编写规范

### SQL 文件结构示例

#### 正向迁移（.up.sql）

**简单字段添加**（如 `000032_add_message_pipeline_stages.up.sql`）：
```sql
-- Add pipeline_stages column to messages table
ALTER TABLE messages ADD COLUMN pipeline_stages JSONB DEFAULT '{}';
```

**创建表**（如 `000000_init.up.sql` 片段）：
```sql
-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create table with IF NOT EXISTS for idempotency
CREATE TABLE IF NOT EXISTS tenants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(64) NOT NULL,
    retriever_engines JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_tenants_api_key ON tenants(api_key);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
```

**添加注释和日志**（推荐做法）：
```sql
-- Migration: 000032_add_message_pipeline_stages
-- Description: Add pipeline_stages column to messages table
DO $$ BEGIN RAISE NOTICE '[Migration 000032] Starting...'; END $$;

ALTER TABLE messages ADD COLUMN pipeline_stages JSONB DEFAULT '{}';

DO $$ BEGIN RAISE NOTICE '[Migration 000032] Completed successfully'; END $$;
```

#### 回滚迁移（.down.sql）

**字段回滚**：
```sql
ALTER TABLE messages DROP COLUMN IF EXISTS pipeline_stages;
```

**表回滚**：
```sql
DROP TABLE IF EXISTS tenants;
```

### 编写要点

| 要点 | 说明 | 示例 |
|------|------|------|
| **幂等性** | 使用 `IF NOT EXISTS`、`IF EXISTS` 等语法 | `CREATE TABLE IF NOT EXISTS` |
| **默认值** | 新增字段设置合理的默认值 | `ADD COLUMN xxx JSONB DEFAULT '{}'` |
| **索引** | 单独创建索引，不要内联在 CREATE TABLE 中 | `CREATE INDEX IF NOT EXISTS` |
| **日志** | 使用 `RAISE NOTICE` 输出迁移进度 | `DO $$ BEGIN RAISE NOTICE '...'; END $$;` |
| **事务** | 单个迁移文件在同一事务中执行（除非使用 DDL 不支持事务的语句） | 无需手动 BEGIN/COMMIT |
| **回滚对应** | `.down.sql` 必须能完全撤销 `.up.sql` 的变更 | 添加字段 ↔ 删除字段 |

### 初始化迁移的特殊处理

`000000_init.up.sql` 是数据库初始化脚本，需要特别注意：

1. **使用 `IF NOT EXISTS`**：确保重复执行不报错
2. **序列初始化**：如果需要设置序列起始值，需检查当前值（见 `000000_init.up.sql:28-43`）
3. **扩展创建**：使用 `CREATE EXTENSION IF NOT EXISTS`

---

## 数据库版本控制机制

### schema_migrations 表

`golang-migrate` 使用 `schema_migrations` 表（在目标数据库中自动创建）记录迁移状态：

```sql
-- 表结构
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL,
    PRIMARY KEY (version)
);
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `version` | bigint | 当前已应用的迁移版本号 |
| `dirty` | boolean | 是否处于脏状态（上次迁移中断） |

### 版本管理流程

```
┌─────────────────┐
│  应用启动       │
│  RunMigrations  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 读取 schema_   │
│ migrations 表  │
│ 获取当前版本   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 扫描 migrations │
│ /versioned/     │
│ 目录下的文件    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 按顺序执行      │
│ 待处理的 .up.sql │
│ (version > current) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 更新            │
│ schema_migrations│
│ version = N     │
│ dirty = false   │
└─────────────────┘
```

### Dirty 状态说明

当迁移过程中发生错误（如 SQL 语法错误、连接中断等），`schema_migrations` 表的 `dirty` 字段会被设置为 `true`。

**影响**：
- 此后所有迁移操作都会失败，直到清除 dirty 状态
- 需要手动检查数据库状态，确认哪些变更已应用、哪些未应用

**恢复方法**：
```bash
# 方法1：强制回退到上一版本（推荐）
make migrate-force version=<previous_version>

# 方法2：启用自动恢复（开发环境）
# 修改 internal/database/migration.go 调用方式
database.RunMigrationsWithOptions(dsn, database.MigrationOptions{AutoRecoverDirty: true})
```

---

## 常用操作工作流

### 1. 创建新迁移

```bash
# 格式：make migrate-create name=迁移名称（下划线分隔）
make migrate-create name=add_user_email

# 生成文件：
# migrations/versioned/000033_add_user_email.up.sql
# migrations/versioned/000033_add_user_email.down.sql
```

编辑生成的 SQL 文件，编写正向变更与回滚逻辑。

### 2. 执行待处理迁移

```bash
# 手动触发（开发环境）
make migrate-up

# 生产环境：启动应用时自动执行
./WeKnora
```

### 3. 查看当前版本

```bash
make migrate-version
# 输出示例：Current migration version: 32, dirty: false
```

### 4. 处理脏状态

当迁移中断导致数据库处于脏状态时：

```bash
# 1. 查看当前版本与脏状态
make migrate-version

# 2. 强制回退到上一版本（如当前脏版本为5，回退到4）
make migrate-force version=4

# 3. 重新启动应用执行迁移
make dev-app
```

### 5. 回滚迁移

```bash
# 回滚1个版本
make migrate-down

# 跳转到指定版本（如跳转到版本3）
make migrate-goto version=3
```

---

## 最佳实践

1. **迁移脚本幂等性**：尽量使用 `CREATE TABLE IF NOT EXISTS`、`ALTER TABLE IF NOT EXISTS`、`DROP TABLE IF EXISTS` 等语法，避免重复执行报错
2. **小步变更**：每次迁移只做单一变更（如新增一个字段、一张表），便于回滚和问题定位
3. **配套回滚脚本**：每个 `.up.sql` 必须对应可执行的 `.down.sql`，确保可回滚
4. **测试迁移**：在开发环境验证迁移脚本后再提交，避免生产环境脏状态
5. **不修改已提交迁移**：已提交的迁移文件不要修改，新增变更只需递增版本号

---

## 故障排查

### 问题1：迁移失败提示脏状态（dirty state）

**错误信息示例**：
```
Error: database is in dirty state at version 5. This usually means a migration failed partway through.
```

**原因**：
- 上次迁移中断（进程被杀死、数据库连接断开等）
- `schema_migrations` 表的 `dirty` 字段被设置为 `true`
- 数据库可能处于部分迁移状态

**诊断步骤**：
```bash
# 1. 查看当前状态
make migrate-version
# 输出示例：Current migration version: 5, dirty: true

# 2. 检查数据库实际状态
# 连接到数据库，查看相关表是否存在
psql -U postgres -d WeKnora -c "\dt"  # 列出所有表
```

**解决方案**：

方案A - 手动恢复（推荐）：
```bash
# 1. 检查版本5的迁移内容（migrations/versioned/000005_xxx.up.sql）
# 2. 手动修复已部分应用的变更（如删除不完整的表、字段等）
# 3. 强制回退到上一版本
make migrate-force version=4
# 4. 重新启动应用
make dev-app
```

方案B - 自动恢复（仅开发环境）：
```go
// 修改 internal/database/migration.go 的调用
database.RunMigrationsWithOptions(dsn, database.MigrationOptions{AutoRecoverDirty: true})
```

### 问题2：迁移执行后无变化

**现象**：执行 `make migrate-up` 后提示无变化

**原因**：
- 数据库已是最新版本（返回 `migrate.ErrNoChange`）
- 所有待处理迁移已执行

**验证**：
```bash
make migrate-version
# 输出示例：Current migration version: 32, dirty: false

# 查看 migrations/versioned/ 目录中最新的版本号
ls migrations/versioned/*.up.sql | sort | tail -1
```

**解决**：无需处理，属于正常情况

### 问题3：SQLite 迁移不生效

**现象**：使用 SQLite 数据库时，迁移未执行

**原因**：
- DSN 前缀不是 `sqlite3://`，导致加载了 PostgreSQL 的迁移目录
- `migrations/sqlite/` 目录不存在或为空

**检查**：
```bash
# 查看 .env 中的数据库配置
cat .env | grep DB_URL

# 正确的 SQLite DSN 格式
DB_URL=sqlite3:///path/to/database.db
```

**解决**：
1. 确保 DSN 以 `sqlite3://` 开头
2. 确保 `migrations/sqlite/` 目录存在且有迁移文件

### 问题4：迁移脚本执行报错

**常见错误类型**：

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| `syntax error at or near...` | SQL 语法错误 | 检查 SQL 脚本语法，特别是 PostgreSQL 特有语法 |
| `relation "xxx" already exists` | 表已存在，但脚本未使用 `IF NOT EXISTS` | 修改脚本添加幂等性语法 |
| `password authentication failed` | 数据库连接失败 | 检查 `.env` 中的数据库用户名密码 |
| `connection refused` | 数据库服务未启动 | 启动数据库服务：`make dev-start` |
| `database "xxx" does not exist` | 数据库不存在 | 手动创建数据库：`createdb WeKnora` |

**调试技巧**：
```bash
# 1. 手动执行 SQL 脚本验证语法
psql -U postgres -d WeKnora -f migrations/versioned/000032_xxx.up.sql

# 2. 查看 migrate 工具的详细输出
migrate -path migrations/versioned -database "${DB_URL}" up -verbose

# 3. 检查数据库日志
docker logs weknora_postgres-1  # 如果使用 Docker
```

### 问题5：创建迁移文件时版本号不连续

**现象**：`make migrate-create name=xxx` 生成的版本号与预期不符

**原因**：
- `migrate create -seq` 会根据现有文件的最大版本号 +1
- 可能存在手动修改或删除的迁移文件

**解决**：
```bash
# 查看当前最大版本号
ls migrations/versioned/*.up.sql | sort | tail -1

# 如果需要指定版本号，手动创建文件
# 但不推荐，建议让 migrate 工具自动管理版本号
```

### 问题6：多个开发者迁移冲突

**场景**：团队开发中，多个开发者同时创建迁移，导致版本号冲突

**预防**：
1. 迁移文件按功能命名，避免重复
2. 提交前先 `git pull` 获取最新代码
3. 如果冲突，协商重新生成迁移文件

**解决**：
```bash
# 1. 如果本地版本号已被占用，重新创建迁移
git pull origin main
make migrate-create name=your_migration_name

# 2. 如果已提交冲突的迁移，需要回滚并重新创建
make migrate-down
# 删除冲突的迁移文件
# 重新创建
make migrate-create name=your_migration_name
```

---

## 高级用法

### 1. 手动控制迁移（不通过 Makefile）

```bash
# 设置环境变量
export DB_URL="postgres://postgres:postgres@localhost:5432/WeKnora?sslmode=disable"

# 直接调用 migrate 工具
migrate -path migrations/versioned -database "${DB_URL}" version
migrate -path migrations/versioned -database "${DB_URL}" up
migrate -path migrations/versioned -database "${DB_URL}" down 1
```

### 2. 重置数据库（开发环境）

```bash
# 警告：会删除所有数据！
# 1. 停止应用
# 2. 删除数据库
dropdb WeKnora
# 3. 创建新数据库
createdb WeKnora
# 4. 重新执行所有迁移
make migrate-up
```

### 3. 使用 Docker 执行迁移

```bash
# 在 Docker 容器中执行迁移
docker-compose exec app ./WeKnora  # 应用启动时会自动执行迁移

# 或者使用 migrate 工具的 Docker 镜像
docker run -v $(pwd)/migrations:/migrations \
  -e DB_URL="postgres://postgres:postgres@host.docker.internal:5432/WeKnora?sslmode=disable" \
  migrate/migrate:latest -path=/migrations/versioned -database="${DB_URL}" up
```

### 4. 迁移文件版本回填

如果项目初期没有使用迁移工具，后期需要补上迁移文件：

```bash
# 1. 创建初始迁移（包含当前所有表结构）
make migrate-create name=init_existing_schema
# 2. 编辑生成的 up.sql，包含所有现有表结构
# 3. 编辑 down.sql，删除所有表
# 4. 强制设置版本
make migrate-force version=0
# 5. 重启应用
```

---

## 参考资源

- **golang-migrate 官方文档**：https://github.com/golang-migrate/migrate
- **PostgreSQL DDL 语法**：https://www.postgresql.org/docs/current/sql-commands.html
- **项目 AGENTS.md**：查看项目特定的迁移命令和约定

---

## 总结

本项目使用 `golang-migrate` 实现数据库版本管理，关键要点：

1. **自动化**：应用启动时自动执行迁移，无需手动干预
2. **版本化**：每个迁移文件有唯一版本号，按序执行
3. **双向迁移**：支持正向升级和回滚降级
4. **多数据库**：支持 PostgreSQL 和 SQLite3，自动适配
5. **开发友好**：提供 Makefile 和脚本封装，简化日常操作
6. **健壮性**：脏状态检测与恢复机制，保证数据库一致性
