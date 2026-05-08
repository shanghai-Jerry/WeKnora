# 多模型对比功能

## 功能概述

多模型对比功能允许用户同时使用多个大语言模型回答同一个问题，并对比它们的回答效果。

## 访问方式

点击左侧菜单栏的"模型对比"图标，或直接访问 `/platform/model-compare` 路由。

## 功能特点

### 1. 智能体选择
- 支持选择不同的智能体（推理智能体 / 快问智能体）
- 根据选择的智能体类型自动调用对应的对话接口

### 2. 模型选择
- 支持选择最多 3 个模型进行对比
- 从可用的 KnowledgeQA 类型模型中选择
- 可以随时添加或移除模型

### 3. 知识库选择
- 通过 @ 符号快速选择知识库
- 支持多知识库同时使用
- 可选功能，不选择知识库时仅使用模型能力

### 4. 多轮对话
- 每个模型维护独立的 session
- 支持连续对话，保持上下文
- 实时显示每个模型的回答

### 5. 对话记录
- 展示完整的对话历史
- 每个模型的对话记录独立显示
- 支持清空对话记录

### 6. 回答对比
- 并排展示多个模型的回答
- 显示响应时间
- 展示引用的知识库文档
- 支持 Markdown 格式渲染

## 使用流程

1. **选择智能体**：在顶部选择要使用的智能体类型
2. **选择模型**：点击"添加模型"按钮，选择要对比的模型（最多3个）
3. **选择知识库**（可选）：点击 @ 按钮选择要使用的知识库
4. **输入问题**：在底部输入框输入问题
5. **发送提问**：点击发送按钮，所有选中的模型会同时回答
6. **查看对比**：在多栏视图中查看各模型的回答
7. **继续对话**：可以继续输入问题，进行多轮对话
8. **清空对话**：点击右上角的"清空对话"按钮，重新开始

## 技术实现

### 前端组件
- **位置**：`frontend/src/views/chat/ModelCompare.vue`
- **依赖**：Vue 3, TDesign Vue Next, Pinia

### 核心功能

#### 1. Session 管理
```typescript
// 每个模型维护独立的 session
columnSessions[modelId] = {
  id: sessionId,
  responseTime: 0
}
```

#### 2. 流式响应处理
```typescript
const { startStream, stopStream, onChunk } = useStream();

onChunk((data) => {
  if (data.response_type === 'answer' || data.response_type === 'complete') {
    // 实时更新回答内容
  } else if (data.response_type === 'references') {
    // 处理引用文档
  }
});
```

#### 3. 智能体类型判断
```typescript
const agentMode = agent?.config?.agent_mode || 'quick-answer';
const endpoint = agentMode === 'smart-reasoning' 
  ? '/api/v1/agent-chat'
  : '/api/v1/knowledge-chat';
```

### API 接口

#### 创建 Session
```
POST /api/v1/sessions
```

#### 对话接口
```
POST /api/v1/agent-chat/{session_id}  # 推理智能体
POST /api/v1/knowledge-chat/{session_id}  # 快问智能体
```

### 翻译文本

中文翻译位置：`frontend/src/i18n/locales/zh-CN.ts`
英文翻译位置：`frontend/src/i18n/locales/en-US.ts`

相关翻译键：
- `menu.modelCompare`: 菜单项
- `chat.modelCompareTitle`: 页面标题
- `chat.modelCompareDesc`: 页面描述
- `chat.selectAgent`: 选择智能体
- `chat.selectKnowledge`: 选择知识库
- `chat.addModel`: 添加模型
- `chat.clearConversation`: 清空对话
- `chat.conversationCleared`: 对话已清空
- `chat.compareGenerating`: 生成中...
- `chat.compareReferences`: 引用文档 ({count} 篇)

## 注意事项

1. **模型限制**：最多同时对比 3 个模型
2. **Session 独立**：每个模型的 session 是独立的，互不干扰
3. **资源消耗**：同时使用多个模型会消耗更多资源
4. **清空对话**：清空对话会删除所有模型的 session
5. **智能体配置**：不同的智能体可能有不同的配置限制

## 后续优化建议

1. **性能优化**：添加模型响应时间的可视化图表
2. **导出功能**：支持导出对比结果为 PDF 或 Markdown
3. **评分系统**：用户可以对模型回答进行评分
4. **历史记录**：保存历史对比记录，方便回顾
5. **批量对比**：支持批量问题对比
