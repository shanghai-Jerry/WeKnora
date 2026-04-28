--  插入rerank模型
INSERT INTO models (
    id,
    tenant_id,
    name,
    type,
    source,
    description,
    parameters,
    is_default,
    status,
    created_at,
    updated_at,
    deleted_at,
    is_builtin
  )
VALUES (
    '409dda1f-6ab9-4206-b52e-9c0c4d75b33c',
    10000,
    'mac-bge-rarank-v2-m3',
    'Rerank',
    'remote',
    'local rerank model',
    '{"base_url":"http://host.docker.internal:8000","api_key":"***","provider":"generic"}',
    false,
    'active',
    now(),
    now(),
    null,
    false
  );

-- 插入KnowledgeQA模型
INSERT INTO models (
    id,
    tenant_id,
    name,
    type,
    source,
    description,
    parameters,
    is_default,
    status,
    created_at,
    updated_at,
    deleted_at,
    is_builtin
  )
VALUES (
    '409dda1f-6ab9-4206-b52e-9c0c4d75b45d',
    10000,
    'qwen/qwen3.6-27b',
    'KnowledgeQA',
    'local',
    'local knowledge qa model',
    '{"base_url":"http://host.docker.internal:1234/v1","api_key":"***","provider":"generic"}',
    false,
    'active',
    now(),
    now(),
    null,
    false
  );


-- source 字段太小，需要修改为text， 长URL就可以导入

ALTER TABLE knowledges
ALTER COLUMN source TYPE text;
