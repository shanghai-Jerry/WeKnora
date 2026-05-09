# Plan: Code Interpreter & HTML Interpreter Tools

## Overview

新增 `code_interpreter` 和 `html_interpreter` 两个 Agent 工具，用于执行任意代码和渲染 HTML 报告。复用项目现有沙箱机制（Docker/Local）提供安全隔离。

## Requirement Doc

`docs/requirements/code-html-interpreter.md`

## Affected Components

| 组件 | 影响 |
|------|------|
| `internal/agent/tools/definitions.go` | 新增工具常量和 UI 定义 |
| `internal/agent/tools/code_interpreter.go` | **新建** — 代码执行工具 |
| `internal/agent/tools/html_interpreter.go` | **新建** — HTML 渲染工具 |
| `internal/application/service/agent_service.go` | 注册工具 + 提取沙箱初始化方法 |
| `internal/types/agent.go` | 无改动（使用现有 Data map 扩展） |

## Implementation Steps

### Step 1: 更新 definitions.go

添加工具常量:

```go
ToolCodeInterpreter = "code_interpreter"
ToolHtmlInterpreter = "html_interpreter"
```

在 `AvailableToolDefinitions()` 中添加 UI 定义，在 `DefaultAllowedTools()` 中添加默认允许。

### Step 2: 新建 code_interpreter.go

**文件**: `internal/agent/tools/code_interpreter.go`

**输入参数**:
```go
type CodeInterpreterInput struct {
    Code     string `json:"code" jsonschema:"Code to execute (Python or JavaScript)"`
    Language string `json:"language,omitempty" jsonschema:"Language: python (default) or javascript"`
    FilePath string `json:"file_path,omitempty" jsonschema:"Optional file path to make available as FILE_PATH variable"`
}
```

**执行流程**:
1. 空值检查 → code 为空返回错误
2. 确定语言（python/javascript），映射到解释器和文件扩展名
3. 构建 work_dir: `data/tmp/{sessionID}/`（跨调用持久化）
4. 注入 preamble 头部代码:
   - Python: `import json, os, pandas as pd, numpy as np; PLOT_DIR = "{work_dir}"; os.makedirs(PLOT_DIR, exist_ok=True)`
   - JavaScript: `const fs = require('fs'); const path = require('path'); const PLOT_DIR = "{work_dir}";`
   - 如有 file_path: `FILE_PATH = "{file_path}"`
5. 扫描 work_dir 中已有图片（用于后续判断新增图片）
6. 拼接 `full_code = preamble + code`
7. 写入临时文件 `_run.py` / `_run.js`
8. 通过 `sandbox.Manager.Execute()` 执行（`SkipValidation: true`, timeout=60s）
9. 解析 stdout，截断超 2000 字符时提示
10. 扫描 work_dir 中新增图片文件
11. 清理临时 `_run.py`（保留 work_dir 供后续复用）
12. 返回 `ToolResult`:
    - Output: 代码原文 + stdout 输出
    - Data: `{"stdout": "...", "images": [...], "exit_code": 0, "language": "python"}`

**构造函数**:
```go
func NewCodeInterpreterTool(sandboxMgr sandbox.Manager, sessionID string) *CodeInterpreterTool
```

**Cleanup**: 无需清理（work_dir 跨调用保留）

### Step 3: 新建 html_interpreter.go

**文件**: `internal/agent/tools/html_interpreter.go`

**输入参数**:
```go
type HtmlInterpreterInput struct {
    HTML     string            `json:"html,omitempty" jsonschema:"Inline HTML content"`
    Title    string            `json:"title,omitempty" jsonschema:"Report title"`
    Data     map[string]string `json:"data,omitempty" jsonschema:"Template placeholder data (key-value pairs)"`
    FilePath string            `json:"file_path,omitempty" jsonschema:"Path to HTML file to read from work directory"`
}
```

**三种模式**:
1. **内联模式** (默认): 直接使用 `html` 参数
2. **文件模式**: 从 `data/tmp/{sessionID}/{file_path}` 读取 HTML 内容
3. **模板替换**: 从 `data/tmp/{sessionID}/{file_path}` 读取模板，用 `data` map 替换 `{{KEY}}` 占位符

**执行流程**:
1. 空值检查: html 和 file_path 均为空 → 返回错误
2. 如有 file_path → 读取文件内容
3. 如有 template_path（复用 file_path） + data → 执行 `{{KEY}}` 替换
4. 返回 `ToolResult`:
    - Output: HTML 内容
    - Data: `{"output_type": "html", "title": "报告标题"}`

**构造函数**:
```go
func NewHtmlInterpreterTool(sessionID string, workDir string) *HtmlInterpreterTool
```

### Step 4: 修改 agent_service.go

1. **提取沙箱初始化方法** `getSandboxManager()`:
   - 从 `initializeSkillsManager()` 中提取沙箱创建逻辑
   - 返回缓存的 `sandbox.Manager` 实例
   - code_interpreter 和 skills 共享同一个沙箱 Manager

2. **在 `registerTools()` 中注册**:
   ```go
   case tools.ToolCodeInterpreter:
       sandboxMgr := s.getSandboxManager(ctx, config)
       toolToRegister = tools.NewCodeInterpreterTool(sandboxMgr, sessionID)

   case tools.ToolHtmlInterpreter:
       workDir := filepath.Join("data", "tmp", sessionID)
       toolToRegister = tools.NewHtmlInterpreterTool(sessionID, workDir)
   ```

### Step 5: 验证

1. `go build ./...` 编译通过
2. 检查工具注册日志输出正确

## Key Design Decisions

| 决策 | 选择 | 原因 |
|------|------|------|
| 沙箱验证 | `SkipValidation: true` | 沙箱本身提供 Docker/Local 隔离，代码解释器需允许 os/subprocess |
| 工作目录 | `data/tmp/{sessionID}/` | 与现有 data/ 结构一致，跨调用持久化 |
| 模板文件源 | 工作目录 | 由 code_interpreter 生成或用户上传到工作目录 |
| 图片服务 | 复用 `/files?file_path=provider://` | 不需要新增路由 |
| 语言支持 | Python + JavaScript | 按用户确认 |
| HTML 输出 | Data["output_type":"html"] | 最小化后端改动，前端按需解析 |

## Expected Deliverables

- `internal/agent/tools/code_interpreter.go` — 代码执行工具
- `internal/agent/tools/html_interpreter.go` — HTML 渲染工具
- `internal/agent/tools/definitions.go` — 更新常量和定义
- `internal/application/service/agent_service.go` — 注册逻辑
- 编译通过，无 lint 错误
