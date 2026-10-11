-- 1. Insert channel 15 (AWS EC2 - Qwen 2.5 3B)
INSERT INTO channels (
    id, type, name, key, base_url, models, model_mapping, status, priority, weight, test_model, created_time, test_time, response_time, "group"
) VALUES (
    15,
    1,
    'AWS EC2 - Qwen 2.5 3B (Sydney Graviton2)',
    'ollama',
    'http://54.206.183.194:11434',
    'qwen2.5:3b,qwen2.5:1.5b,qwen-2.5-3b,qwen-2.5-1.5b,toraapi-local,qwen-aws',
    '{"qwen-2.5-3b": "qwen2.5:3b", "qwen-2.5-1.5b": "qwen2.5:1.5b", "toraapi-local": "qwen2.5:3b", "qwen-aws": "qwen2.5:3b"}',
    1,
    25,
    10,
    'qwen2.5:3b',
    EXTRACT(epoch FROM now())::bigint,
    EXTRACT(epoch FROM now())::bigint,
    1200,
    'default,aws,qwen-stable'
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    key = EXCLUDED.key,
    base_url = EXCLUDED.base_url,
    models = EXCLUDED.models,
    model_mapping = EXCLUDED.model_mapping,
    status = EXCLUDED.status,
    priority = EXCLUDED.priority,
    weight = EXCLUDED.weight,
    test_model = EXCLUDED.test_model,
    "group" = EXCLUDED."group";

-- Advance sequence if needed
SELECT setval('channels_id_seq', (SELECT MAX(id) FROM channels));

-- 2. Populate abilities for channel 14 (Modal GLM)
DELETE FROM abilities WHERE channel_id = 14;
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g, m, 14, true, 30, 10
FROM (VALUES ('default'), ('glm-stable')) AS groups(g)
CROSS JOIN (VALUES 
    ('nvidia/GLM-5.3-Flash-NVFP4'), 
    ('glm-5.3-flash'), 
    ('glm-5.3'), 
    ('toraapi'), 
    ('toraapi-fast')
) AS models(m)
ON CONFLICT ("group", model, channel_id) DO NOTHING;

-- 3. Populate abilities for channel 15 (AWS EC2 Qwen)
DELETE FROM abilities WHERE channel_id = 15;
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g, m, 15, true, 25, 10
FROM (VALUES ('default'), ('aws'), ('qwen-stable')) AS groups(g)
CROSS JOIN (VALUES 
    ('qwen2.5:3b'), 
    ('qwen2.5:1.5b'), 
    ('qwen-2.5-3b'), 
    ('qwen-2.5-1.5b'), 
    ('toraapi-local'), 
    ('qwen-aws')
) AS models(m)
ON CONFLICT ("group", model, channel_id) DO NOTHING;
