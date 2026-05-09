# Code Interpreter & HTML Interpreter

## Original Requirement
新增 code_interpreter 和 html_interpreter 两个 Agent 工具，用于执行任意代码和渲染 HTML 报告。code_interpreter 支持 Python 和 JavaScript，html_interpreter 支持内联 HTML、模板替换和文件读取三种模式。

## Scope
### In Scope
- code_interpreter: 执行 Python/JavaScript 代码，支持数据分析、计算、图表生成
- html_interpreter: 渲染 HTML 为可交互网页报告，支持三种模式
- 复用项目沙箱机制（Docker/Local）提供安全隔离
- 工作目录跨调用持久化（`data/tmp/{sessionID}/`）
- 自动生成图片检测和 URL 返回
- 模板占位符 `{{KEY}}` 自动替换

### Out of Scope
- 前端 HTML 渲染面板（需单独实现）
- 图片静态文件服务路由（复用现有 `/files` 端点）
- 代码自动修复（_try_repair_truncated_code）
- 图片 URL 自动修正（html_interpreter 中的 UUID 前缀修复）

## Key Decisions & Tradeoffs
- **跳过沙箱验证**: 使用 `SkipValidation=true`，依赖沙箱隔离层提供安全防护，允许更广泛的代码执行
- **Python + JavaScript 双语言**: 按用户确认，同时支持两种语言
- **模板文件从工作目录读取**: 模板文件在 `data/tmp/{sessionID}/` 中，由 code_interpreter 生成或用户上传
- **ToolResult.Data 扩展**: 使用 `Data["output_type"]` 标记 HTML 输出，前端按需解析

## Affected Components
- `internal/agent/tools/code_interpreter.go`: 新建代码执行工具
- `internal/agent/tools/html_interpreter.go`: 新建 HTML 渲染工具
- `internal/agent/tools/definitions.go`: 添加工具常量和 UI 定义
- `internal/application/service/agent_service.go`: 注册工具 + 提取沙箱初始化方法

## Change History
| Date | Change | Reason |
|------|--------|--------|
| 2026-05-09 | Initial creation | 原始需求 |

## Related Links
- Plan: `plans/code-html-interpreter-20260509.md`
- Reference: `/Users/moineye/.dbgpt/DB-GPT/docs/question/code-interpreter-and-html-interpreter.md`
- Sandbox: `docs/modules/sandbox-isolation.md`
