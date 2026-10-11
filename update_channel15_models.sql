-- Add 7b models to channel 15
UPDATE channels
SET models = models || ',qwen2.5:7b,qwen2.5-7b-instruct,qwen-2.5-7b,qwen-7b,deepseek-r1:7b,deepseek-r1-distill-qwen-7b,deepseek-r1-7b',
    model_mapping = (model_mapping::jsonb || '{"qwen2.5-7b-instruct": "qwen2.5:7b", "qwen-2.5-7b": "qwen2.5:7b", "qwen-7b": "qwen2.5:7b", "deepseek-r1-distill-qwen-7b": "deepseek-r1:1.5b", "deepseek-r1-7b": "deepseek-r1:1.5b"}'::jsonb)::text
WHERE id = 15;

-- Insert abilities for the new models on channel 15
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g, m, 15, true, 20, 10
FROM (VALUES ('default'), ('aws'), ('new-users')) AS grp(g)
CROSS JOIN (VALUES 
    ('qwen2.5:7b'), 
    ('qwen2.5-7b-instruct'), 
    ('qwen-2.5-7b'), 
    ('qwen-7b'),
    ('deepseek-r1:7b'),
    ('deepseek-r1-distill-qwen-7b'),
    ('deepseek-r1-7b')
) AS mdl(m)
ON CONFLICT DO NOTHING;
