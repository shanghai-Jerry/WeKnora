

select * from embeddings WHERE chunk_id = '6185481d-a1fc-485c-9653-93fa64008b04'


select * from chunks WHERE id = '6185481d-a1fc-485c-9653-93fa64008b04'


SELECT * from knowledge_bases limit 10


SELECT * from knowledges LIMIT 10


SELECT * from tenants  LIMIT 10

SELECT * from users  LIMIT 10



--- 搜索角膜塑形镜的作用和功能
SELECT 
paradedb.score(id) as score,
"id","content","source_id","source_type","chunk_id","knowledge_id","knowledge_base_id","tag_id" 
FROM "embeddings" 
WHERE "knowledge_base_id" = '17a1acf9-eb62-4431-829d-2815372742ab' 
AND id @@@ paradedb.match(field => 'content', value => '角膜塑形镜的作用和功能', distance => 1) 
AND ((is_enabled IS NULL OR is_enabled = true)) 
ORDER BY "score" DESC LIMIT 150


-- 查看 public 模式下某张表（如 documents）的索引
SELECT
    tablename,
    indexname,
    indexdef
FROM
    pg_indexes
WHERE
    schemaname = 'public'
    AND tablename = 'embeddings';