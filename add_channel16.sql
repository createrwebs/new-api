DELETE FROM abilities WHERE channel_id = 16;
DELETE FROM channels WHERE id = 16;

INSERT INTO channels (
    id, type, key, status, name, weight, priority, base_url, models, model_mapping, "group", created_time, auto_ban
) VALUES (
    16,
    1,
    'sd-turbo',
    1,
    'AWS EC2 - SD-Turbo Image (Sydney Sapphire Rapids)',
    10,
    25,
    'http://13.239.38.104:8000',
    'sd-turbo,dall-e-2,dall-e-3,sd-image',
    '{"dall-e-2": "sd-turbo", "dall-e-3": "sd-turbo", "sd-image": "sd-turbo"}',
    'default,aws,new-users',
    EXTRACT(EPOCH FROM NOW())::bigint,
    1
);

INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g, m, 16, true, 25, 10
FROM (VALUES ('default'), ('aws'), ('new-users')) AS grp(g)
CROSS JOIN (VALUES ('sd-turbo'), ('dall-e-2'), ('dall-e-3'), ('sd-image')) AS mdl(m);
