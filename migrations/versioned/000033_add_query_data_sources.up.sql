-- Migration: 000033_add_query_data_sources
-- Description: Create query_data_sources table for AI query (sql_query tool)

CREATE TABLE IF NOT EXISTS query_data_sources (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,  -- mysql, postgresql, sqlite
    config JSONB NOT NULL,      -- 存储连接配置（加密后）
    description TEXT,
    status VARCHAR(32) DEFAULT 'active',  -- active, inactive, error
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_query_data_sources_tenant_id ON query_data_sources (tenant_id);
CREATE INDEX IF NOT EXISTS idx_query_data_sources_type ON query_data_sources (type);
CREATE INDEX IF NOT EXISTS idx_query_data_sources_status ON query_data_sources (status);
CREATE INDEX IF NOT EXISTS idx_query_data_sources_deleted_at ON query_data_sources (deleted_at);
