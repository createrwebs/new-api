import subprocess
import json

# Fetch current ModelRatio
cmd = ["docker", "exec", "postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", 
       "SELECT value FROM options WHERE key = 'ModelRatio';"]
out = subprocess.check_output(cmd).decode("utf-8").strip()
model_ratios = json.loads(out)

# Add our new models
new_model_ratios = {
    "qwen2.5:3b": 0.02,
    "qwen2.5:1.5b": 0.01,
    "qwen-2.5-3b": 0.02,
    "qwen-2.5-1.5b": 0.01,
    "qwen-aws": 0.02,
    "toraapi-local": 0.02,
    "nvidia/GLM-5.3-Flash-NVFP4": 0.057143,
    "toraapi": 0.057143,
    "toraapi-fast": 0.057143
}

for k, v in new_model_ratios.items():
    model_ratios[k] = v

updated_json = json.dumps(model_ratios)

# Save back to Postgres
save_sql = f"UPDATE options SET value = $${updated_json}$$ WHERE key = 'ModelRatio';"
subprocess.check_call([
    "docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", save_sql
])
print("Successfully updated ModelRatio in database with new models!")

# Fetch and update CompletionRatio if needed
cmd = ["docker", "exec", "postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", 
       "SELECT value FROM options WHERE key = 'CompletionRatio';"]
try:
    out = subprocess.check_output(cmd).decode("utf-8").strip()
    if out:
        comp_ratios = json.loads(out)
        for k in new_model_ratios.keys():
            if k not in comp_ratios:
                comp_ratios[k] = 1.0
        updated_comp = json.dumps(comp_ratios)
        save_comp_sql = f"UPDATE options SET value = $${updated_comp}$$ WHERE key = 'CompletionRatio';"
        subprocess.check_call([
            "docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", save_comp_sql
        ])
        print("Successfully updated CompletionRatio in database!")
except Exception as e:
    print(f"CompletionRatio update note: {e}")
