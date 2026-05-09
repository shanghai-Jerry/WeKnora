# Data Source Selection in Chat

## Original Requirement
在 `/platform/creatChat` 页面添加数据源选择功能，允许用户从已添加的 Query Data Sources 中选择数据源，并将其作为参数传递给会话创建流程。

## Scope
### In Scope
- 在 Input-field 组件的控制栏添加 "+" 按钮
- 点击 "+" 按钮显示下拉菜单，包含 "Upload File"、"Select Data Source"、"Select Knowledge Base" 选项
- 点击 "Select Data Source" 打开模态框，显示当前租户的 Query Data Sources 列表
- 用户可选择一个或多个数据源
- 选中的 data_source_ids 存储在 settingsStore 中
- 在 createSessions API 调用中传递 data_source_ids 参数

### Out of Scope
- 创建新数据源的完整流程（仅提供 "Add New Data Source" 跳转）
- 数据源的编辑/删除功能
- 数据源的同步状态显示

## Key Decisions & Tradeoffs
- 使用 Query Data Sources API (`/api/v1/query-data-sources`) 获取数据源列表
- 选中的数据源存储在 settingsStore 的 `selectedDataSources` 字段中
- 数据源选择与知识库选择逻辑分离，互不影响

## Affected Components
- `frontend/src/components/Input-field.vue` - 添加 "+" 按钮和菜单
- `frontend/src/components/DataSourceSelector.vue` - 新增数据源选择模态框组件
- `frontend/src/stores/settings.ts` - 添加 selectedDataSources 状态管理
- `frontend/src/views/creatChat/creatChat.vue` - 在 createSessions 中传递 data_source_ids
- `frontend/src/api/query-datasource/index.ts` - 已有 API，无需修改

## Change History
| Date | Change | Reason |
|------|--------|--------|
| 2026-05-08 | Initial creation | Original requirement |
| 2026-05-08 | Implementation completed | Added "+" menu, DataSourceSelector component, and data source state management |

## Related Links
- Plan: `plans/data-source-selection-20260508.md`
- API: `frontend/src/api/query-datasource/index.ts`
- Backend: `internal/handler/query_data_source.go`
