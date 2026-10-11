UPDATE channels 
SET 
    name = 'Modal - GLM 5.3 Flash (NVFP4)',
    base_url = 'https://websbringer--ep-glm-5-3-flash-nvfp4-server.us-west.modal.direct',
    key = 'wk-IaPxNpRcSE187z4fMeG9gk.ws-3JXbidDmc0HpWBUGUb5y0o',
    models = 'nvidia/GLM-5.3-Flash-NVFP4,glm-5.3-flash,glm-5.3,toraapi,toraapi-fast',
    model_mapping = '{"glm-5.3-flash": "nvidia/GLM-5.3-Flash-NVFP4", "glm-5.3": "nvidia/GLM-5.3-Flash-NVFP4", "toraapi": "nvidia/GLM-5.3-Flash-NVFP4", "toraapi-fast": "nvidia/GLM-5.3-Flash-NVFP4"}',
    test_model = 'nvidia/GLM-5.3-Flash-NVFP4',
    status = 1,
    priority = 30,
    weight = 10,
    "group" = 'default,glm-stable'
WHERE id = 14;
