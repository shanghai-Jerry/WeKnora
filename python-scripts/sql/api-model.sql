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
    '409dda1f-6ab9-4206-b52e-9c0c4d75b32b',
    10000,
    'bge-rarank-v2-m3',
    'Rerank',
    'remote',
    'local rerank model',
    '{"base_url":"http://rerank:8000","api_key":"***","provider":"generic"}',
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


-- extract_config 字段太小，需要修改为text， 长JSON就可以导入
--
update knowledges
set extract_config = '[]'
where id = '409dda1f-6ab9-4206-b52e-9c0c4d75b32b';


-- 插入AntAngelMed模型
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
    '409dda1f-6ab9-4206-b52e-9c0c4d75b32b',
    10000,
    'bge-rarank-v2-m3',
    'Rerank',
    'remote',
    'local rerank model',
    '{"base_url":"http://rerank:8000","api_key":"***","provider":"generic"}',
    false,
    'active',
    now(),
    now(),
    null,
    false
  );
