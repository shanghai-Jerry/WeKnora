# Query Intent Explore 插件处理流程文档

## 文件信息
- 文件路径：`internal/application/service/chat_pipeline/query_intent_explore.go`
- 包名：`chatpipeline`
- 核心结构体：`PluginQueryIntentExplore`，依赖模型服务、配置、搜索插件

## 核心功能
该插件作为聊天流水线的一环，负责将用户原始查询拆解为多个分析路径，生成针对性搜索查询，并行执行检索并汇聚结果，为后续对话生成提供更丰富的上下文。

## 处理流程
1. **事件触发与开关检查**
   - 监听 `QUERY_INTENT_EXPLORE` 类型事件
   - 检查 `chatManage.EnableQueryIntentExplore` 开关，若关闭则跳过流程，调用 `next()` 继续后续插件

2. **模型与提示词准备**
   - 根据 `chatManage.ChatModelID` 获取聊天模型
   - 读取配置中的意图探索提示词 `config.Conversation.IntentExplorePrompt`，若为空则跳过
   - 构建用户内容：优先使用配置的 `IntentExplorePromptUser`，若存在 `{{query}}` 占位符则替换为重写后的查询 `chatManage.RewriteQuery`，否则直接使用重写查询

3. **流式调用LLM**
   - 构造系统提示（提示词）和用户消息，设置温度0.3、最大生成token数65536
   - 调用模型的 `ChatStream` 方法获取流式响应，实时收集answer类型的文本内容，忽略thinking、complete等类型，记录错误

4. **响应解析**
   - 拼接流式收集到的完整文本内容
   - 调用 `parseOutput` 方法解析JSON：提取首尾`{}`之间的内容，反序列化为 `intentExploreOutput` 结构
   - 解析结果需包含非空 `FinalSearchQueries`，否则判定为解析失败，跳过后续流程

5. **数据写入上下文**
   - 将解析结果写入 `chatManage.IntentExploreData`，包含原始查询、分析路径列表、最终搜索查询列表
   - 分析路径包含实体、维度、合并搜索字符串、关系路径相关字段（源实体、目标实体等）

6. **并行搜索执行**
   - 调用 `searchMultiplePaths` 方法，对每个最终搜索查询启动独立goroutine并行搜索
   - 单路径搜索逻辑：`Clone` 聊天上下文，替换查询为当前路径查询，复用 `PluginSearch` 插件执行检索，去重后返回结果
   - 所有路径搜索完成后，合并结果并去重，写入 `chatManage.SearchResult`

7. **事件通知与流程传递**
   - 通过 `chatManage.EventBus` 发送 `EventQueryIntentExplore` 事件，携带分析路径、最终查询、搜索结果总数等信息
   - 调用 `next()` 将控制权传递给流水线下一个插件

## 关键方法说明
- `parseOutput`：从LLM响应中提取并解析JSON结构，校验有效性
- `searchMultiplePaths`：并行执行多路径搜索，汇总去重结果
- `searchSinglePath`：单路径搜索实现，复用搜索插件逻辑

## 前后端开关控制

### 后端配置层
1. **全局配置**（`internal/config/config.go:85`）
   - 配置文件字段：`conversation.enable_query_intent_explore`（bool）
   - 由 `Config.Conversation.EnableQueryIntentExplore` 读取，作为全局默认值

2. **智能体自定义配置**（`internal/types/custom_agent.go:179`）
   - 智能体配置支持 `enable_query_intent_explore` 字段（`*bool` 类型，支持 nil 表示不覆盖）
   - 优先级：智能体配置 > 全局配置

3. **生效逻辑**（`internal/application/service/session_qa_helpers.go:171-173`）
   ```go
   if customAgent.Config.EnableQueryIntentExplore != nil {
       cm.EnableQueryIntentExplore = *customAgent.Config.EnableQueryIntentExplore
   }
   ```
   - 普通模式：使用 `session_knowledge_qa.go:115` 中的全局配置 `s.cfg.Conversation.EnableQueryIntentExplore`
   - 智能体模式：若智能体配置中该字段非 nil，则覆盖全局配置

### 前端控制层
1. **智能体编辑器**（`frontend/src/views/agent/AgentEditorModal.vue:1010`）
   - 使用 `<t-switch>` 组件绑定 `formData.config.enable_query_intent_explore`
   - 标签：`agent.editor.enableQueryIntentExplore`，描述：`agentEditor.desc.queryIntentExplore`

2. **API 接口定义**（`frontend/src/api/agent/index.ts:65`）
   ```typescript
   enable_query_intent_explore?: boolean; // 是否启用意图探索
   ```
   - 位于 `AgentConfig` 接口中，作为可选字段

3. **普通模式**
   - 前端不单独控制，使用后端全局配置决定的行为

## SSE 事件流处理

### 后端事件发送
1. **事件触发**（`query_intent_explore.go:223-232`）
   - 插件通过 `chatManage.EventBus.Emit()` 发送 `EventQueryIntentExplore` 事件
   - 事件数据结构 `event.QueryIntentExploreData`：
     - `OriginalQuery`：原始查询
     - `AnalysisPaths`：分析路径列表（含实体、维度、关系路径等字段）
     - `FinalSearchQueries`：最终搜索查询列表
     - `TotalSearchCount`：搜索结果总数

2. **SSE 处理器**（`internal/handler/session/agent_stream_handler.go:704-739`）
   - 注册：`h.eventBus.On(event.EventQueryIntentExplore, h.handleQueryIntentExplore)`
   - 处理逻辑：
     - 将 `AnalysisPaths` 转换为 `map[string]interface{}` 数组
     - 调用 `streamManager.AppendEvent()` 推送 SSE 事件
   - 事件结构：
     ```go
     interfaces.StreamEvent{
         ID:        evt.ID,
         Type:      types.ResponseTypeQueryIntentExplore, // "query_intent_explore"
         Content:   "",
         Done:      true,
         Timestamp: time.Now(),
         Data: map[string]interface{}{
             "original_query":       data.OriginalQuery,
             "analysis_paths":       pathsData,
             "final_search_queries": data.FinalSearchQueries,
             "total_search_count":   data.TotalSearchCount,
         },
     }
     ```

3. **响应类型定义**（`internal/types/chat.go:73`）
   ```go
   ResponseTypeQueryIntentExplore ResponseType = "query_intent_explore"
   ```

### 前端 SSE 接收处理
1. **事件监听**（`frontend/src/views/chat/index.vue:625, 675-682`）
   - 判断条件：`data.response_type === 'query_intent_explore'`
   - 存储位置：`existingMessage.pipeline_stages.intentExplore`

2. **数据结构**
   ```typescript
   existingMessage.pipeline_stages.intentExplore = {
       originalQuery: data.data?.original_query || '',
       analysisPaths: data.data?.analysis_paths || [],
       finalSearchQueries: data.data?.final_search_queries || [],
       totalSearchCount: data.data?.total_search_count || 0
   };
   ```

3. **展示层**
   - `intentExplore` 数据可在聊天界面展示分析路径、搜索查询等信息
   - 属于 `pipeline_stages` 的一部分，与 `queryRewritten`、`retrievalQuery` 等并列展示

## 依赖说明
- 依赖 `PluginSearch` 插件执行实际的检索逻辑
- 依赖模型服务调用LLM进行意图分析
- 依赖事件总线进行结果通知
- 依赖 SSE 流管理器（`streamManager`）向前端推送实时事件
