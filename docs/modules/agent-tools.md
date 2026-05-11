# Agent Tools 文档

本文档总结了 `internal/agent/tools` 目录中所有 Agent Tool 的输入、输出和作用。

## 工具列表

### 1. thinking (思考工具)

**作用**: 动态和反思性的问题解决思考工具，帮助通过灵活的思考过程分析复杂问题，支持修订、分支和回溯。

**输入参数**:
- `thought` (string, 必需): 当前思考步骤，用自然语言描述，不要提及工具名称
- `next_thought_needed` (boolean, 必需): 是否需要更多思考
- `thought_number` (integer, 必需): 当前思考编号（从1开始）
- `total_thoughts` (integer, 必需): 预估需要的思考总数
- `is_revision` (boolean, 可选): 是否修订之前的思考
- `revises_thought` (integer, 可选): 正在重新考虑的思考编号
- `branch_from_thought` (integer, 可选): 分支点的思考编号
- `branch_id` (string, 可选): 分支标识符
- `needs_more_thoughts` (boolean, 可选): 是否需要更多思考

**输出**:
- `thought_number`: 当前思考编号
- `total_thoughts`: 总思考数
- `next_thought_needed`: 是否需要下一步思考
- `branches`: 分支列表
- `thought_history_length`: 思考历史长度
- `display_type`: "thinking"
- `thought`: 思考内容
- `incomplete_steps`: 是否有未完成的步骤

---

### 2. todo_write (制定计划工具)

**作用**: 创建和管理结构化的任务列表，用于跟踪检索和研究任务进度，只跟踪检索任务，不包括总结任务。

**输入参数**:
- `task` (string): 需要制定计划的复杂任务或问题
- `steps` (array, 必需): 研究计划步骤数组
  - `id` (string): 步骤唯一标识符
  - `description` (string): 步骤描述
  - `status` (string): 状态（pending/in_progress/completed）

**输出**:
- `task`: 任务描述
- `steps`: 步骤数组
- `steps_json`: 步骤的JSON字符串
- `total_steps`: 总步骤数
- `plan_created`: 是否成功创建计划
- `display_type`: "plan"

---

### 3. grep_chunks (关键词搜索工具)

**作用**: Unix风格文本模式匹配工具，在知识库分块中执行精确的关键词搜索（固定字符串匹配，非语义搜索）。

**输入参数**:
- `patterns` (array, 必需): 文本模式数组（1-3个词的关键词，OR逻辑）
- `knowledge_base_ids` (array, 可选): 知识库ID过滤
- `max_results` (integer, 可选): 最大返回结果数（默认50，最大200）

**输出**:
- `patterns`: 搜索的关键词
- `knowledge_results`: 按知识库聚合的结果
- `result_count`: 结果数量
- `total_matches`: 总匹配数
- `knowledge_base_ids`: 搜索的知识库ID
- `max_results`: 最大结果数
- `display_type`: "grep_results"

---

### 4. knowledge_search (语义搜索工具)

**作用**: 使用嵌入向量进行语义搜索，理解用户查询的含义并查找语义相关内容，按语义相似度排序。

**输入参数**:
- `queries` (array, 必需): 1-5个语义问题或概念陈述（简短、表述良好的问题）
- `knowledge_base_ids` (array, 可选): 限制搜索范围的知识库ID（最多10个）

**输出**:
- 按语义相似度排序的搜索结果
- 包含分块ID、内容、分数、匹配类型等信息
- 重新排序后的结果（如果配置了重排序模型）

---

### 5. list_knowledge_chunks (查看文档分块工具)

**作用**: 通过knowledge_id检索文档的完整分块内容，用于查看已知文档的全部分块。

**输入参数**:
- `knowledge_id` (string, 必需): 文档ID
- `limit` (integer, 可选): 每页分块数（默认20，最大100）
- `offset` (integer, 可选): 起始位置（默认0）

**输出**:
- `knowledge_id`: 文档ID
- `knowledge_title`: 文档标题
- `total_chunks`: 总分块数
- `fetched_chunks`: 已获取的分块数
- `page`: 页码
- `page_size`: 每页大小
- `chunks`: 分块数组，每个包含：
  - `seq`: 序号
  - `chunk_id`: 分块ID
  - `chunk_index`: 分块索引
  - `content`: 内容
  - `chunk_type`: 分块类型
  - `images`: 图片信息（如果有）

---

### 6. query_knowledge_graph (查询知识图谱工具)

**作用**: 探索知识图谱中的实体关系和知识网络，仅对配置了图谱提取的知识库有效。

**输入参数**:
- `knowledge_base_ids` (array, 必需): 知识库ID数组（1-10个）
- `query` (string, 必需): 查询内容（实体名称或查询文本）

**输出**:
- `knowledge_base_ids`: 查询的知识库ID
- `query`: 查询内容
- `results`: 查询结果数组
- `count`: 结果数量
- `kb_counts`: 各知识库的结果计数
- `graph_configs`: 图谱配置信息
- `graph_data`: 用于前端可视化的图谱数据
- `has_graph_config`: 是否有图谱配置
- `errors`: 错误信息
- `display_type`: "graph_query_results"

---

### 7. get_document_info (获取文档信息工具)

**作用**: 检索文档的详细元数据信息，包括基本信息、文件信息、处理状态和自定义元数据。

**输入参数**:
- `knowledge_ids` (array, 必需): 文档/知识ID数组

**输出**:
- `documents`: 文档信息数组
  - `knowledge_id`: 文档ID
  - `title`: 标题
  - `description`: 描述
  - `type`: 类型
  - `source`: 来源
  - `file_name`: 文件名
  - `file_type`: 文件类型
  - `file_size`: 文件大小
  - `parse_status`: 解析状态
  - `chunk_count`: 分块数
  - `metadata`: 元数据
- `total_docs`: 成功获取的文档数
- `requested`: 请求的文档数
- `errors`: 错误信息
- `display_type`: "document_info"

---

### 8. database_query (查询数据库工具)

**作用**: 执行SQL查询从数据库检索信息，自动注入tenant_id和软删除过滤，仅允许SELECT语句。

**输入参数**:
- `sql` (string, 必需): SELECT SQL查询语句（不要包含tenant_id条件，会自动添加）

**可查询的表**:
- `knowledge_bases`: 知识库信息
- `knowledges`: 文档信息
- `chunks`: 分块信息

**输出**:
- `columns`: 列名数组
- `rows`: 结果行数组
- `row_count`: 行数
- `display_type`: "database_query"

---

### 9. data_analysis (数据分析工具)

**作用**: 针对CSV或Excel文件，将数据加载到DuckDB内存中并执行SQL进行数据分析。

**输入参数**:
- `knowledge_id` (string, 必需): 要查询的知识ID
- `sql` (string, 必需): 要在知识数据上执行的SQL

**输出**:
- 查询结果，包含列名和数据行
- 支持统计分析和数据聚合

**注意**: 仅支持只读查询（SELECT、SHOW、DESCRIBE、EXPLAIN、PRAGMA）

---

### 10. data_schema (查看数据元信息工具)

**作用**: 获取加载到DuckDB的CSV或Excel文件的schema信息，返回表名、列和行数。

**输入参数**:
- `knowledge_id` (string, 必需): 要查询的知识ID

**输出**:
- `summary`: 表摘要信息
- `columns`: 列信息
- 输出文本包含表结构和列详情

---

### 11. web_search (网络搜索工具)

**作用**: 搜索网络获取最新信息和新闻，在知识库检索（grep_chunks和knowledge_search）之后使用。

**输入参数**:
- `query` (string, 必需): 搜索查询字符串

**输出**:
- 网络搜索结果，包含：
  - 标题
  - URL
  - 摘要
  - 内容片段（可能被截断）
- 结果自动压缩并使用RAG提取相关内容
- 搜索结果存储在会话的临时知识库中

**重要**: 必须先在知识库中完成检索（grep_chunks和knowledge_search），仅在知识库结果不足时使用此工具。

---

### 12. web_fetch (网页抓取工具)

**作用**: 从URL获取网页内容并转换为Markdown，使用LLM分析内容（如果可用）。

**输入参数**:
- `items` (array, 必需): 批量抓取任务数组
  - `url` (string): 要抓取的网页URL（应来自web_search结果）
  - `prompt` (string): 用于分析抓取内容的提示词

**输出**:
- 每个URL的抓取结果
- LLM分析结果（如果配置了聊天模型）
- 原始内容片段

**重要**: 在web_search返回结果后，如果内容被截断或不完整，使用此工具获取完整页面内容。

---

### 13. final_answer (提交最终回答工具)

**作用**: 向用户提交最终答案，必须是Agent调用的最后一个工具。

**输入参数**:
- `answer` (string, 必需): 完整的最终答案，Markdown格式，包含所有引用和格式

**输出**:
- `answer`: 答案内容（直接返回给用户的输出）

**重要规则**:
1. 必须作为最后一个工具调用
2. answer参数必须包含完整、格式良好的响应
3. 包含所有引用、结构和格式

---

### 14. read_skill (读取技能工具)

**作用**: 按需读取技能内容以学习专业能力，加载技能的完整说明（SKILL.md内容）。

**输入参数**:
- `skill_name` (string, 必需): 要读取的技能名称
- `file_path` (string, 可选): 技能目录中特定文件的相对路径

**输出**:
- `skill_name`: 技能名称
- `description`: 技能描述
- `instructions`: 技能说明（如果未指定file_path）
- `instructions_length`: 说明长度
- `files`: 可用文件列表
- 或指定文件的内容（如果指定了file_path）

---

### 15. execute_skill_script (执行技能脚本工具)

**作用**: 在沙箱环境中执行技能中的实用脚本，用于自动化或数据处理。

**输入参数**:
- `skill_name` (string, 必需): 包含脚本的技能名称
- `script_path` (string, 必需): 技能目录中脚本的相对路径（如scripts/analyze.py）
- `args` (array, 可选): 传递给脚本的命令行参数
- `input` (string, 可选): 通过stdin传递给脚本的输入数据

**输出**:
- `skill_name`: 技能名称
- `script_path`: 脚本路径
- `args`: 参数
- `exit_code`: 退出码
- `stdout`: 标准输出
- `stderr`: 标准错误
- `duration_ms`: 执行时长（毫秒）
- `killed`: 是否被终止

**安全特性**:
- 在隔离沙箱中运行
- 默认禁用网络访问
- 文件访问限制在技能目录内

---

### 16. MCP工具 (MCP服务工具)

**作用**: 动态包装MCP（Model Context Protocol）服务工具，实现Tool接口，允许Agent使用外部MCP服务提供的工具。

**输入参数**: 取决于MCP服务提供的工具定义（JSON Schema）

**输出**: 取决于MCP服务提供的工具

**命名规则**: `mcp_{service_name}_{tool_name}`

**特性**:
- 支持多种传输类型（stdio、HTTP等）
- 自动重连机制
- 工具名称 sanitization 以符合OpenAI API要求
- 描述前缀标识外部来源以提高安全性

### 17. code_interpreter (代码解释器)

**作用**: 在沙箱环境中执行 Python 或 JavaScript 代码，用于数据分析、计算和图表生成。

**输入参数**:
- `code` (string, 必需): 要执行的代码
- `language` (string, 可选): 语言，`python`（默认）或 `javascript`
- `file_path` (string, 可选): 可选文件路径，注入为 `FILE_PATH` 变量

**可用环境变量**:
- `PLOT_DIR`: 工作目录，用于保存输出文件
- `FILE_PATH`: 用户提供的文件路径（如果指定）
- Python 预装: `pandas`, `numpy`
- JavaScript 预装: `fs`, `path`

**输出**:
- `stdout`: 标准输出
- `stderr`: 标准错误
- `exit_code`: 退出码
- `images`: 生成的图片列表
- `language`: 执行语言
- `duration_ms`: 执行耗时

**安全机制**:
- 沙箱隔离执行
- 60 秒超时
- 命令白名单限制
- stdout 超过 2000 字符自动截断

**典型用法**:
```python
import pandas as pd
import matplotlib.pyplot as plt

df = pd.read_csv(FILE_PATH)
print(df.describe())

plt.figure(figsize=(10, 6))
df['col'].hist()
plt.savefig(f"{PLOT_DIR}/chart.png")
```

---

### 18. html_interpreter (HTML 渲染器)

**作用**: 将 HTML 内容渲染为可交互的网页报告。支持内联 HTML、文件读取和模板占位符替换三种模式。

**输入参数**:
- `html` (string, 可选): 内联 HTML 内容
- `file_path` (string, 可选): 工作目录中的 HTML 文件路径
- `title` (string, 可选): 报告标题（默认"HTML Report"）
- `data` (map[string]string, 可选): 模板占位符替换数据（`{{KEY}}` → value）

**输出**:
- `output_type`: "html"
- `title`: 报告标题

**使用场景**:
- `code_interpreter` 生成图表后，渲染包含图表的完整报告
- 展示数据仪表盘或交互式可视化
- 生成格式化的数据分析报告

**典型工作流**:
```
1. code_interpreter: 生成数据和图表 → 保存到 PLOT_DIR
2. html_interpreter: 生成包含图表引用的 HTML 报告 → 前端展示
```

**自动包裹**: 如果 HTML 不含 `<!DOCTYPE` 或 `<html` 标签，自动包裹为完整 HTML 文档。

---

## 默认允许的工具

以下工具在默认配置中启用（`DefaultAllowedTools`）：

1. `thinking` - 思考
2. `todo_write` - 制定计划
3. `knowledge_search` - 语义搜索
4. `grep_chunks` - 关键词搜索
5. `list_knowledge_chunks` - 查看文档分块
6. `query_knowledge_graph` - 查询知识图谱
7. `get_document_info` - 获取文档信息
8. `database_query` - 查询数据库
9. `data_analysis` - 数据分析
10. `data_schema` - 查看数据元信息
11. `code_interpreter` - 代码执行
12. `html_interpreter` - HTML 渲染
13. `final_answer` - 提交最终回答

技能相关工具（`read_skill`、`execute_skill_script`）仅在启用技能功能时可用。
解释器工具（`code_interpreter`、`html_interpreter`）默认启用，用于数据分析和报告生成。

---

## 工具使用工作流示例

### 典型知识库查询流程

1. **thinking** - 分析问题，制定检索策略
2. **todo_write** - 创建检索任务列表
3. **grep_chunks** - 关键词搜索定位相关文档
4. **knowledge_search** - 语义搜索查找相关内容
5. **list_knowledge_chunks** - 查看完整分块内容
6. **get_document_info** - 获取文档元数据
7. **thinking** - 综合信息
8. **final_answer** - 提交最终答案

### 数据分析流程

1. **data_schema** - 查看数据表结构
2. **data_analysis** - 执行SQL分析
3. **final_answer** - 提交分析结果

### 网络信息补充流程

1. 完成知识库检索（grep_chunks + knowledge_search）
2. **web_search** - 搜索网络补充信息
3. **web_fetch** - 获取完整网页内容
4. **final_answer** - 提交最终答案

### 数据分析与可视化流程

1. **code_interpreter** - 执行 Python/JavaScript 进行数据分析和图表生成
2. **html_interpreter** - 将分析结果和图表渲染为交互式 HTML 报告
3. **final_answer** - 提交最终分析结论

---

## 工具开发说明

所有工具都基于 `BaseTool` 结构体，实现 `types.Tool` 接口：

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() json.RawMessage
    Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}
```

工具结果结构：

```go
type ToolResult struct {
    Success bool                   // 执行是否成功
    Output  string                 // 文本输出
    Data    map[string]interface{} // 结构化数据
    Error   string                 // 错误信息
}
```

工具注册在 `ToolRegistry` 中，通过 `ExecuteTool` 方法统一执行，包含参数验证、类型转换和输出截断等功能。
