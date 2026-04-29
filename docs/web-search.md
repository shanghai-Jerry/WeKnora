# Web Search 网络搜索

## 架构说明

WeKnora 的网络搜索功能采用**双轨架构**，支持新旧两种配置方式：

### 新架构（推荐）

使用独立的 `WebSearchProviderEntity` 实体管理搜索引擎配置：

```
web_search_providers 表
├── id (UUID, 主键)
├── tenant_id (租户隔离)
├── name (用户友好的名称，如 "Production Bing")
├── provider (类型: bing/google/duckduckgo/tavily)
├── parameters (加密的 JSON: api_key, engine_id 等)
├── is_default (是否为租户默认提供商)
└── description
```

- CRUD 接口：`/api/v1/web-search-providers`
- 获取类型列表：`GET /api/v1/web-search-providers/types`
- 每个租户可创建多个 provider 实例，通过 `is_default=true` 标记默认提供商

### 旧架构（已弃用）

使用 `Tenant` 表中的 `web_search_config` jsonb 字段：

```json
{
  "provider": "bing",        // Deprecated: 使用 WebSearchProviderEntity
  "api_key": "xxx",          // Deprecated: 使用 WebSearchProviderEntity.Parameters.APIKey
  "max_results": 10,
  "include_date": false,
  "compression_method": "none",
  "blacklist": []
}
```

- 读写接口：`/api/v1/tenants/kv/web-search-config`

### 后端判断逻辑

后端在执行搜索时优先使用新架构：

```go
// internal/application/service/web_search.go
if defaultProvider, err := s.webSearchProviderRepo.GetDefault(ctx, tenantInfo.ID); err == nil && defaultProvider != nil {
    // 使用 WebSearchProviderEntity
}
// fallback 到旧的 WebSearchConfig.Provider
```

---

## API 接口

| 方法 | 路径 | 描述 |
| ---- | ---- | ---- |
| GET | `/web-search/providers` | 获取网络搜索服务商类型列表（旧） |
| GET | `/web-search-providers/types` | 获取支持的服务商类型（新） |
| GET | `/web-search-providers` | 列出当前租户的所有 provider 实例 |
| POST | `/web-search-providers` | 创建新的 provider 实例 |
| PUT | `/web-search-providers/{id}` | 更新 provider 实例 |
| DELETE | `/web-search-providers/{id}` | 删除 provider 实例 |
| POST | `/web-search-providers/{id}/test` | 测试 provider 连接 |
| GET | `/tenants/kv/web-search-config` | 获取租户 web search 配置（旧） |
| PUT | `/tenants/kv/web-search-config` | 更新租户 web search 配置（旧） |

---

## 前端判断逻辑

前端 `Input-field.vue` 通过 `loadWebSearchConfig()` 判断搜索是否已配置：

```typescript
// 检查新旧两种配置方式
const [configRes, providersRes] = await Promise.all([
  getTenantWebSearchConfig(),      // 旧: 检查 config.provider
  listWebSearchProviders(),         // 新: 检查是否有 provider 实体
]);
const configured = !!(config && config.provider) || providers.length > 0;
```

**注意**：如果只在新架构下配置了 provider，旧的 `web-search-config` 接口会返回空结构体（而非 null），但前端会通过检查 `providers` 列表来判断是否已配置。

---

## 常见问题排查

### 问题：web-search-providers 有数据，但 /api/v1/tenants/kv/web-search-config 返回 null

**根因**：这是新旧架构差异导致的正常现象。

- `web-search-providers` 接口返回的是 `WebSearchProviderEntity` 列表（新架构）
- `web-search-config` 接口返回的是 `Tenant.WebSearchConfig`（旧架构字段）

如果只使用新架构配置 provider，`web-search-config` 的 `WebSearchConfig` 字段在数据库中为 null，接口会返回空结构体：

```json
{
  "success": true,
  "data": {
    "max_results": 0,
    "include_date": false,
    "compression_method": "",
    "blacklist": []
  }
}
```

**修复**：前端已修改 `loadWebSearchConfig()` 同时检查新旧两种配置方式。

### 问题：前端提示"未配置网络搜索引擎"

**排查步骤**：

1. 检查是否有 provider 实例：
   ```bash
   curl http://localhost:8080/api/v1/web-search-providers \
     -H 'X-API-Key: your_key'
   ```

2. 如果返回空数组，需要先在设置页面添加 provider：
   - 进入 设置 → 网络搜索
   - 点击"添加搜索引擎"
   - 选择类型、填写 API Key，并勾选"设为默认"

3. 如果有 provider 但前端仍提示未配置，检查该 provider 是否设置了 `is_default=true`

---

## 添加新搜索引擎类型

参见 [添加新的网络搜索引擎.md](./添加新的网络搜索引擎.md)
