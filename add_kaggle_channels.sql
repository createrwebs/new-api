-- Channels 17, 18, 19, 20 for Kaggle GPU Workers
DELETE FROM abilities WHERE channel_id IN (17, 18, 19, 20);
DELETE FROM channels WHERE id IN (17, 18, 19, 20);

INSERT INTO channels (
    id, type, key, status, name, weight, priority, base_url, models, model_mapping, "group", created_time, auto_ban
) VALUES 
(
    17, 1, 'kaggle-token', 1,
    'Kaggle GPU Worker 1 (noppananthe - Tesla T4)',
    10, 30, 'http://54.206.183.194:11434',
    'qwen2.5:7b,qwen2.5-7b-instruct,qwen-2.5-7b,qwen-7b,deepseek-r1:7b,deepseek-r1-distill-qwen-7b,deepseek-r1-7b',
    '{"qwen2.5-7b-instruct": "qwen2.5:7b", "qwen-2.5-7b": "qwen2.5:7b", "qwen-7b": "qwen2.5:7b", "deepseek-r1-distill-qwen-7b": "deepseek-r1:7b", "deepseek-r1-7b": "deepseek-r1:7b"}',
    'default,aws,new-users', EXTRACT(EPOCH FROM NOW())::bigint, 1
),
(
    18, 1, 'kaggle-token', 1,
    'Kaggle GPU Worker 2 (codeatheerachart - Tesla T4)',
    10, 30, 'http://54.206.183.194:11434',
    'qwen2.5:7b,qwen2.5-7b-instruct,qwen-2.5-7b,qwen-7b,deepseek-r1:7b,deepseek-r1-distill-qwen-7b,deepseek-r1-7b',
    '{"qwen2.5-7b-instruct": "qwen2.5:7b", "qwen-2.5-7b": "qwen2.5:7b", "qwen-7b": "qwen2.5:7b", "deepseek-r1-distill-qwen-7b": "deepseek-r1:7b", "deepseek-r1-7b": "deepseek-r1:7b"}',
    'default,aws,new-users', EXTRACT(EPOCH FROM NOW())::bigint, 1
),
(
    19, 1, 'kaggle-token', 1,
    'Kaggle GPU Worker 3 (weeblysdwebmagic - Tesla T4)',
    10, 30, 'http://54.206.183.194:11434',
    'qwen2.5:7b,qwen2.5-7b-instruct,qwen-2.5-7b,qwen-7b,deepseek-r1:7b,deepseek-r1-distill-qwen-7b,deepseek-r1-7b',
    '{"qwen2.5-7b-instruct": "qwen2.5:7b", "qwen-2.5-7b": "qwen2.5:7b", "qwen-7b": "qwen2.5:7b", "deepseek-r1-distill-qwen-7b": "deepseek-r1:7b", "deepseek-r1-7b": "deepseek-r1:7b"}',
    'default,aws,new-users', EXTRACT(EPOCH FROM NOW())::bigint, 1
),
(
    20, 1, 'kaggle-token', 1,
    'Kaggle GPU Worker 4 (creatorwebs - Tesla T4)',
    10, 30, 'http://54.206.183.194:11434',
    'qwen2.5:7b,qwen2.5-7b-instruct,qwen-2.5-7b,qwen-7b,deepseek-r1:7b,deepseek-r1-distill-qwen-7b,deepseek-r1-7b',
    '{"qwen2.5-7b-instruct": "qwen2.5:7b", "qwen-2.5-7b": "qwen2.5:7b", "qwen-7b": "qwen2.5:7b", "deepseek-r1-distill-qwen-7b": "deepseek-r1:7b", "deepseek-r1-7b": "deepseek-r1:7b"}',
    'default,aws,new-users', EXTRACT(EPOCH FROM NOW())::bigint, 1
);

-- Insert abilities for channels 17, 18, 19, 20
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT grp.g, mdl.m, ch.id, true, 30, 10
FROM (VALUES (17), (18), (19), (20)) AS ch(id)
CROSS JOIN (VALUES ('default'), ('aws'), ('new-users')) AS grp(g)
CROSS JOIN (VALUES 
    ('qwen2.5:7b'), 
    ('qwen2.5-7b-instruct'), 
    ('qwen-2.5-7b'), 
    ('qwen-7b'),
    ('deepseek-r1:7b'),
    ('deepseek-r1-distill-qwen-7b'),
    ('deepseek-r1-7b')
) AS mdl(m);
