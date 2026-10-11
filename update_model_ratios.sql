UPDATE options
SET value = (value::jsonb || '{"qwen2.5:3b": 0.02, "qwen-2.5-3b": 0.02, "qwen2.5:1.5b": 0.01, "qwen-2.5-1.5b": 0.01, "toraapi-local": 0.02, "qwen-aws": 0.02, "llama3.2:3b": 0.02, "llama3.2": 0.02, "llama-3.2-3b": 0.02, "deepseek-r1:1.5b": 0.01, "deepseek-r1-1.5b": 0.01, "toraapi-fast": 0.01, "qwen2.5:7b": 0.05, "qwen2.5-7b-instruct": 0.05, "qwen-2.5-7b": 0.05, "qwen-7b": 0.05, "deepseek-r1:7b": 0.05, "deepseek-r1-distill-qwen-7b": 0.05, "deepseek-r1-7b": 0.05}'::jsonb)::text
WHERE key = 'ModelRatio';
