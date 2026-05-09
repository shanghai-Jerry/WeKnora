# Changelog

本文档记录所有需求变更和实现历史，按时间倒序排列。

## 格式说明
- 每个条目包含：功能名称、日期、关联的计划文档和需求文档

## 变更记录

| Date | Feature | Plan | Requirement | Summary | Affected Components |
|------|---------|------|-------------|---------|---------------------|
| 2026-05-08 | Data Source Selection in Chat | [plans/data-source-selection-20260508.md](plans/data-source-selection-20260508.md) | [docs/requirements/data-source-selection-in-chat.md](docs/requirements/data-source-selection-in-chat.md) | 在创建聊天页面添加数据源选择功能，支持从 Query Data Sources 中选择并传递给会话 | stores/settings.ts, components/Input-field.vue, components/DataSourceSelector.vue, views/creatChat/creatChat.vue, views/chat/index.vue |
| 2026-04-30 | Planning Workflow Setup | [plans/planning-workflow-setup-20260430.md](plans/planning-workflow-setup-20260430.md) | - | 添加需求规划工作流到 AGENTS.md，创建 docs/requirements/ 目录和 CHANGELOG.md | AGENTS.md, docs/ |
