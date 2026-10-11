import subprocess
import json

# 1. Update UserUsableGroups
cmd = ["docker", "exec", "postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", 
       "SELECT value FROM options WHERE key = 'UserUsableGroups';"]
out = subprocess.check_output(cmd).decode("utf-8").strip()
usable_groups = json.loads(out)
usable_groups["new-users"] = "New Users Trial / สิทธิ์ทดลองใช้งานสำหรับผู้ใช้ใหม่"
updated_usable = json.dumps(usable_groups)
save_sql = f"UPDATE options SET value = $${updated_usable}$$ WHERE key = 'UserUsableGroups';"
subprocess.check_call(["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", save_sql])
print("Updated UserUsableGroups with new-users!")

# 2. Update GroupRatio
cmd = ["docker", "exec", "postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", 
       "SELECT value FROM options WHERE key = 'GroupRatio';"]
out = subprocess.check_output(cmd).decode("utf-8").strip()
group_ratios = json.loads(out)
group_ratios["new-users"] = 1.0
updated_ratios = json.dumps(group_ratios)
save_sql = f"UPDATE options SET value = $${updated_ratios}$$ WHERE key = 'GroupRatio';"
subprocess.check_call(["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", save_sql])
print("Updated GroupRatio with new-users!")

# 3. Update Channel 15 (AWS EC2) and Channel 14 (Modal GLM)
sql_update_channels = """
UPDATE channels SET "group" = 'default,aws,qwen-stable,new-users' WHERE id = 15;
UPDATE channels SET "group" = 'default,glm-stable,new-users' WHERE id = 14;

-- Abilities for channel 15 with new-users
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT 'new-users', m, 15, true, 25, 10
FROM (VALUES 
    ('qwen2.5:3b'), 
    ('qwen2.5:1.5b'), 
    ('qwen-2.5-3b'), 
    ('qwen-2.5-1.5b'), 
    ('toraapi-local'), 
    ('qwen-aws')
) AS models(m)
ON CONFLICT ("group", model, channel_id) DO NOTHING;

-- Abilities for channel 14 with new-users
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight)
SELECT 'new-users', m, 14, true, 30, 10
FROM (VALUES 
    ('nvidia/GLM-5.3-Flash-NVFP4'), 
    ('glm-5.3-flash'), 
    ('glm-5.3'), 
    ('toraapi'), 
    ('toraapi-fast')
) AS models(m)
ON CONFLICT ("group", model, channel_id) DO NOTHING;
"""
subprocess.check_call(["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", sql_update_channels])
print("Updated channels and abilities with new-users group!")
