import subprocess
import json

# 1. Update Channel 15 models & mapping
sql_channel = """
UPDATE channels 
SET 
    models = 'qwen2.5:3b,qwen2.5:1.5b,qwen-2.5-3b,qwen-2.5-1.5b,toraapi-local,qwen-aws,llama3.2:3b,llama3.2,llama-3.2-3b,deepseek-r1:1.5b,deepseek-r1-1.5b',
    model_mapping = '{"qwen-2.5-3b": "qwen2.5:3b", "qwen-2.5-1.5b": "qwen2.5:1.5b", "toraapi-local": "qwen2.5:3b", "qwen-aws": "qwen2.5:3b", "llama3.2": "llama3.2:3b", "llama-3.2-3b": "llama3.2:3b", "deepseek-r1-1.5b": "deepseek-r1:1.5b"}'
WHERE id = 15;

-- Abilities for new models across default, aws, and new-users
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT g, m, 15, true, 25, 10
FROM (VALUES ('default'), ('aws'), ('new-users')) AS groups(g)
CROSS JOIN (VALUES 
    ('llama3.2:3b'),
    ('llama3.2'),
    ('llama-3.2-3b'),
    ('deepseek-r1:1.5b'),
    ('deepseek-r1-1.5b')
) AS models(m)
ON CONFLICT ("group", model, channel_id) DO NOTHING;
"""
subprocess.check_call(["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", sql_channel])
print("Updated channel 15 and abilities!")

# 2. Update ModelRatio and CompletionRatio
cmd = ["docker", "exec", "postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", 
       "SELECT value FROM options WHERE key = 'ModelRatio';"]
out = subprocess.check_output(cmd).decode("utf-8").strip()
model_ratios = json.loads(out)
new_ratios = {
    "llama3.2:3b": 0.02,
    "llama3.2": 0.02,
    "llama-3.2-3b": 0.02,
    "deepseek-r1:1.5b": 0.01,
    "deepseek-r1-1.5b": 0.01
}
for k, v in new_ratios.items():
    model_ratios[k] = v

save_sql = f"UPDATE options SET value = $${json.dumps(model_ratios)}$$ WHERE key = 'ModelRatio';"
subprocess.check_call(["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", save_sql])
print("Updated ModelRatio for Llama and DeepSeek!")

# CompletionRatio
cmd = ["docker", "exec", "postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", 
       "SELECT value FROM options WHERE key = 'CompletionRatio';"]
try:
    out = subprocess.check_output(cmd).decode("utf-8").strip()
    comp_ratios = json.loads(out)
    for k in new_ratios.keys():
        comp_ratios[k] = 1.0
    save_comp_sql = f"UPDATE options SET value = $${json.dumps(comp_ratios)}$$ WHERE key = 'CompletionRatio';"
    subprocess.check_call(["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", save_comp_sql])
    print("Updated CompletionRatio for Llama and DeepSeek!")
except Exception as e:
    print(f"CompletionRatio note: {e}")
