INSERT INTO channels (
    type, name, key, base_url, models, model_mapping, status, priority, weight, test_model, created_time, test_time, response_time, "group"
) VALUES (
    1,
    'Modal - Qwen 3.6 35B B200',
    'wk-IaPxNpRcSE187z4fMeG9gk.ws-3JXbidDmc0HpWBUGUb5y0o',
    'https://websbringer--ep-toraapi-server.ap-south.modal.direct',
    'toraapi,qwen-3.6-35b,qwen3.6-35b,toraapi-b200',
    '{"qwen-3.6-35b": "toraapi", "qwen3.6-35b": "toraapi", "toraapi-b200": "toraapi"}',
    1,
    20,
    10,
    'toraapi',
    EXTRACT(epoch FROM now())::bigint,
    EXTRACT(epoch FROM now())::bigint,
    1500,
    'default,qwen-stable'
) RETURNING id, name, models;
