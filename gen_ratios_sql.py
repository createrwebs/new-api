new_ratios = {
    # AWS EC2 #1 Models (3B & 1.5B)
    "qwen2.5:3b": 0.02,
    "qwen-2.5-3b": 0.02,
    "qwen2.5:1.5b": 0.01,
    "qwen-2.5-1.5b": 0.01,
    "toraapi-local": 0.02,
    "qwen-aws": 0.02,
    "llama3.2:3b": 0.02,
    "llama3.2": 0.02,
    "llama-3.2-3b": 0.02,
    "deepseek-r1:1.5b": 0.01,
    "deepseek-r1-1.5b": 0.01,
    "toraapi-fast": 0.01,

    # Kaggle GPU Models (7B)
    "qwen2.5:7b": 0.05,
    "qwen2.5-7b-instruct": 0.05,
    "qwen-2.5-7b": 0.05,
    "qwen-7b": 0.05,
    "deepseek-r1:7b": 0.05,
    "deepseek-r1-distill-qwen-7b": 0.05,
    "deepseek-r1-7b": 0.05
}

import json

sql = f"""UPDATE options
SET value = (value::jsonb || '{json.dumps(new_ratios)}'::jsonb)::text
WHERE key = 'ModelRatio';
"""

with open("d:\\new-api\\update_model_ratios.sql", "w", encoding="utf-8") as f:
    f.write(sql)
print("Generated update_model_ratios.sql successfully!")
