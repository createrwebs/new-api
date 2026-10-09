#!/usr/bin/env python3
"""
Update model pricing, routing groups, and capabilities for Tora API (www.toraapi.com).
Configures 35 models across 10 providers with their exact official list prices,
group route multipliers, and routing group associations in the database.
"""

import json
import sqlite3
import subprocess
import sys
import time

# -----------------------------------------------------------------------------
# Specification Data
# -----------------------------------------------------------------------------

GROUP_RATIOS = {
    "openai-award": 0.03,
    "openai-stable": 0.08,
    "openai-premium": 0.13,
    "openai-first-topup": 0.015,
    "claude-premium": 0.5,
    "claude-award": 0.25,
    "claude-stable": 0.315,
    "claude-first-topup": 0.03,
    "claude-tmp-cheap": 0.06,
    "jev-premium": 1.15,
    "spark-premium": 1.15,
    "grok-award": 0.01,
    "grok-stable": 0.11,
    "deepseek-stable": 0.6,
    "glm-stable": 0.5,
    "qwen-stable": 0.8,
    "kimi-stable": 0.65,
    "gemini-stable": 0.15,
}

USABLE_GROUPS = {
    "openai-award": "High-value route",
    "openai-stable": "High-stability route",
    "openai-premium": "High-stability route",
    "openai-first-topup": "Limited-time offer route",
    "claude-premium": "High-stability route",
    "claude-award": "High-value route",
    "claude-stable": "High-stability route",
    "claude-first-topup": "Limited-time offer route",
    "claude-tmp-cheap": "Limited-time offer route",
    "jev-premium": "High-stability route",
    "spark-premium": "High-stability route",
    "grok-award": "Limited-time offer route",
    "grok-stable": "High-stability route",
    "deepseek-stable": "High-stability route",
    "glm-stable": "High-stability route",
    "qwen-stable": "High-stability route",
    "kimi-stable": "High-stability route",
    "gemini-stable": "High-stability route",
}

VENDORS = [
    {"id": 1, "name": "Google", "icon": "Gemini"},
    {"id": 2, "name": "Anthropic", "icon": "Anthropic"},
    {"id": 3, "name": "OpenAI", "icon": "OpenAI"},
    {"id": 4, "name": "DeepSeek", "icon": "DeepSeek"},
    {"id": 5, "name": "Zhipu AI", "icon": "Zhipu"},
    {"id": 6, "name": "ElevenLabs", "icon": "ElevenLabs"},
    {"id": 7, "name": "Kling", "icon": "Kling"},
    {"id": 8, "name": "TypeSafe", "icon": "TypeSafe"},
    {"id": 9, "name": "Meta", "icon": "Meta"},
    {"id": 10, "name": "xAI", "icon": "xAI"},
    {"id": 11, "name": "Alibaba Cloud", "icon": "Qwen"},
    {"id": 12, "name": "Moonshot AI", "icon": "Moonshot"},
]

# 35 models specification
MODELS = [
    # 1. OpenAI (9)
    {
        "name": "gpt-5.6-sol",
        "vendor": "OpenAI",
        "desc": "Context window: 1.05M",
        "tags": "Chat Completions,Responses",
        "ratio": 2.5,
        "completion_ratio": 6.0,
        "groups": ["openai-award", "openai-stable", "openai-premium", "openai-first-topup"],
        "channel_type": 1,
    },
    {
        "name": "gpt-5.6-terra",
        "vendor": "OpenAI",
        "desc": "Context window: 1.05M",
        "tags": "Chat Completions,Responses",
        "ratio": 1.0,
        "completion_ratio": 6.0,
        "groups": ["openai-award", "openai-stable", "openai-premium", "openai-first-topup"],
        "channel_type": 1,
    },
    {
        "name": "gpt-6.1-sol",
        "vendor": "OpenAI",
        "desc": "Context window: 1.05M",
        "tags": "Chat Completions,Responses",
        "ratio": 1.0,
        "completion_ratio": 5.0,
        "groups": ["openai-award", "openai-stable", "openai-premium", "openai-first-topup"],
        "channel_type": 1,
    },
    {
        "name": "gpt-6-luna",
        "vendor": "OpenAI",
        "desc": "Context window: 1.05M",
        "tags": "Responses,Responses Compact",
        "ratio": 0.125,
        "completion_ratio": 5.0,
        "groups": ["openai-award", "openai-stable", "openai-first-topup"],
        "channel_type": 1,
    },
    {
        "name": "gpt-6-sol",
        "vendor": "OpenAI",
        "desc": "Context window: 1.05M",
        "tags": "Chat Completions,Responses",
        "ratio": 1.0,
        "completion_ratio": 5.0,
        "groups": ["openai-award", "openai-stable", "openai-premium", "openai-first-topup"],
        "channel_type": 1,
    },
    {
        "name": "gpt-6-astra",
        "vendor": "OpenAI",
        "desc": "Context window: 1.05M",
        "tags": "Chat Completions,Responses",
        "ratio": 5.0,
        "completion_ratio": 5.0,
        "groups": ["openai-award", "openai-stable", "openai-premium", "openai-first-topup"],
        "channel_type": 1,
    },
    {
        "name": "codex-auto-review",
        "vendor": "OpenAI",
        "desc": "Context window: 272K",
        "tags": "Chat Completions,Responses",
        "ratio": 1.25,
        "completion_ratio": 6.0,
        "groups": ["openai-award", "openai-stable", "openai-premium", "openai-first-topup"],
        "channel_type": 1,
    },
    {
        "name": "gpt-5.4",
        "vendor": "OpenAI",
        "desc": "Context window: 1M",
        "tags": "Chat Completions,Responses",
        "ratio": 1.25,
        "completion_ratio": 6.0,
        "groups": ["openai-award", "openai-stable", "openai-premium", "openai-first-topup"],
        "channel_type": 1,
    },
    {
        "name": "gpt-5.5",
        "vendor": "OpenAI",
        "desc": "Context window: 272K",
        "tags": "Chat Completions,Responses",
        "ratio": 2.5,
        "completion_ratio": 6.0,
        "groups": ["openai-award", "openai-stable", "openai-premium", "openai-first-topup"],
        "channel_type": 1,
    },

    # 2. Anthropic (8)
    {
        "name": "claude-sonnet-5-5",
        "vendor": "Anthropic",
        "desc": "Anthropic Messages, Chat Completions",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 1.0,
        "completion_ratio": 5.0,
        "groups": ["claude-premium", "claude-award", "claude-stable"],
        "channel_type": 14,
    },
    {
        "name": "claude-opus-5-5",
        "vendor": "Anthropic",
        "desc": "Context window: 1M",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 2.0,
        "completion_ratio": 5.0,
        "groups": ["claude-premium", "claude-first-topup", "claude-award", "claude-stable", "claude-tmp-cheap"],
        "channel_type": 14,
    },
    {
        "name": "claude-fable-5-1",
        "vendor": "Anthropic",
        "desc": "Anthropic Messages, Chat Completions",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 5.0,
        "completion_ratio": 5.0,
        "groups": ["claude-premium", "claude-first-topup", "claude-award", "claude-stable", "claude-tmp-cheap"],
        "channel_type": 14,
    },
    {
        "name": "claude-opus-5",
        "vendor": "Anthropic",
        "desc": "Context window: 1M",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 2.5,
        "completion_ratio": 5.0,
        "groups": ["claude-premium", "claude-first-topup", "claude-award", "claude-stable", "claude-tmp-cheap"],
        "channel_type": 14,
    },
    {
        "name": "claude-haiku-4-5",
        "vendor": "Anthropic",
        "desc": "Context window: 200K",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 0.5,
        "completion_ratio": 5.0,
        "groups": ["claude-premium", "claude-first-topup", "claude-award", "claude-stable", "claude-tmp-cheap"],
        "channel_type": 14,
    },
    {
        "name": "claude-sonnet-5",
        "vendor": "Anthropic",
        "desc": "Context window: 1M",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 1.0,
        "completion_ratio": 5.0,
        "groups": ["claude-premium", "claude-first-topup", "claude-award", "claude-stable", "claude-tmp-cheap"],
        "channel_type": 14,
    },
    {
        "name": "claude-opus-4-8",
        "vendor": "Anthropic",
        "desc": "Context window: 1M",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 2.5,
        "completion_ratio": 5.0,
        "groups": ["claude-premium", "claude-first-topup", "claude-award", "claude-stable", "claude-tmp-cheap"],
        "channel_type": 14,
    },
    {
        "name": "claude-fable-5",
        "vendor": "Anthropic",
        "desc": "Context window: 1M",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 5.0,
        "completion_ratio": 5.0,
        "groups": ["claude-premium", "claude-first-topup", "claude-award", "claude-stable", "claude-tmp-cheap"],
        "channel_type": 14,
    },

    # 3. TypeSafe (2)
    {
        "name": "jev-1.13.0",
        "vendor": "TypeSafe",
        "desc": "decisions",
        "tags": "decisions",
        "ratio": 0.021,
        "completion_ratio": 0.0,
        "groups": ["jev-premium"],
        "channel_type": 1,
    },
    {
        "name": "jev-1.13",
        "vendor": "TypeSafe",
        "desc": "decisions",
        "tags": "decisions",
        "ratio": 0.021,
        "completion_ratio": 0.0,
        "groups": ["jev-premium"],
        "channel_type": 1,
    },

    # 4. Meta (1)
    {
        "name": "muse-spark-1.3",
        "vendor": "Meta",
        "desc": "Chat Completions",
        "tags": "Chat Completions",
        "ratio": 0.625,
        "completion_ratio": 3.4,
        "groups": ["spark-premium"],
        "channel_type": 1,
    },

    # 5. xAI (3)
    {
        "name": "grok-4.7",
        "vendor": "xAI",
        "desc": "Context window: 500K",
        "tags": "Chat Completions,Responses",
        "ratio": 1.0,
        "completion_ratio": 3.0,
        "groups": ["grok-award", "grok-stable"],
        "channel_type": 1,
    },
    {
        "name": "grok-4.6",
        "vendor": "xAI",
        "desc": "Context window: 500K",
        "tags": "Chat Completions,Responses",
        "ratio": 1.0,
        "completion_ratio": 3.0,
        "groups": ["grok-award", "grok-stable"],
        "channel_type": 1,
    },
    {
        "name": "grok-4.5",
        "vendor": "xAI",
        "desc": "Context window: 500K",
        "tags": "Chat Completions,Responses",
        "ratio": 1.0,
        "completion_ratio": 3.0,
        "groups": ["grok-award", "grok-stable"],
        "channel_type": 1,
    },

    # 6. DeepSeek (3)
    {
        "name": "deepseek-v4.1-flash",
        "vendor": "DeepSeek",
        "desc": "Chat Completions, Responses",
        "tags": "Chat Completions,Responses",
        "ratio": 0.1612905,
        "completion_ratio": 4.0,
        "groups": ["deepseek-stable"],
        "channel_type": 1,
    },
    {
        "name": "deepseek-v4-flash",
        "vendor": "DeepSeek",
        "desc": "Context window: 1M",
        "tags": "Chat Completions,Responses",
        "ratio": 0.2142855,
        "completion_ratio": 3.0,
        "groups": ["deepseek-stable"],
        "channel_type": 1,
    },
    {
        "name": "deepseek-v4-pro",
        "vendor": "DeepSeek",
        "desc": "Context window: 1M",
        "tags": "Chat Completions",
        "ratio": 0.64285,
        "completion_ratio": 3.0,
        "groups": ["deepseek-stable"],
        "channel_type": 1,
    },

    # 7. Zhipu AI (2)
    {
        "name": "glm-5.3-flash",
        "vendor": "Zhipu AI",
        "desc": "Context window: 1M",
        "tags": "Responses,Anthropic Messages",
        "ratio": 0.057143,
        "completion_ratio": 3.5,
        "groups": ["glm-stable"],
        "channel_type": 1,
    },
    {
        "name": "glm-5.3",
        "vendor": "Zhipu AI",
        "desc": "Chat Completions, Responses",
        "tags": "Chat Completions,Responses",
        "ratio": 0.57145,
        "completion_ratio": 3.5,
        "groups": ["glm-stable"],
        "channel_type": 1,
    },

    # 8. Alibaba Cloud (1)
    {
        "name": "qwen3.8-max",
        "vendor": "Alibaba Cloud",
        "desc": "Context window: 1M",
        "tags": "Anthropic Messages,Chat Completions",
        "ratio": 0.85715,
        "completion_ratio": 3.0,
        "groups": ["qwen-stable"],
        "channel_type": 1,
    },

    # 9. Moonshot AI (1)
    {
        "name": "kimi-k3",
        "vendor": "Moonshot AI",
        "desc": "Context window: 1M",
        "tags": "Chat Completions,Responses",
        "ratio": 1.42855,
        "completion_ratio": 5.0,
        "groups": ["kimi-stable"],
        "channel_type": 1,
    },

    # 10. Google (5)
    {
        "name": "gemini-3.8-flash",
        "vendor": "Google",
        "desc": "Context window: 1M",
        "tags": "Gemini Native,Chat Completions",
        "ratio": 0.75,
        "completion_ratio": 5.0,
        "groups": ["gemini-stable"],
        "channel_type": 1,
    },
    {
        "name": "gemini-3.7-flash",
        "vendor": "Google",
        "desc": "Context window: 1M",
        "tags": "Gemini Native,Chat Completions",
        "ratio": 0.75,
        "completion_ratio": 5.0,
        "groups": ["gemini-stable"],
        "channel_type": 1,
    },
    {
        "name": "gemini-3.6-flash",
        "vendor": "Google",
        "desc": "Context window: 1M",
        "tags": "Gemini Native,Chat Completions",
        "ratio": 0.75,
        "completion_ratio": 5.0,
        "groups": ["gemini-stable"],
        "channel_type": 1,
    },
    {
        "name": "gemini-3.5-flash",
        "vendor": "Google",
        "desc": "Context window: 1M",
        "tags": "Gemini Native,Chat Completions",
        "ratio": 0.75,
        "completion_ratio": 6.0,
        "groups": ["gemini-stable"],
        "channel_type": 1,
    },
    {
        "name": "gemini-3.1-pro-preview",
        "vendor": "Google",
        "desc": "Context window: 1M",
        "tags": "Gemini Native,Chat Completions",
        "ratio": 1.0,
        "completion_ratio": 6.0,
        "groups": ["gemini-stable"],
        "channel_type": 1,
    },
]

def run_psql(sql: str) -> str:
    cmd = ["ssh", "saascover-api", "docker exec -i postgres psql -U root -d new-api -At"]
    res = subprocess.run(cmd, input=sql, text=True, capture_output=True, check=True)
    return res.stdout.strip()

def get_prod_option(key: str) -> str:
    escaped_key = key.replace("'", "''")
    res = run_psql(f"SELECT value FROM options WHERE key = '{escaped_key}';")
    return res.strip()

def set_prod_option(key: str, value: str):
    escaped_key = key.replace("'", "''")
    escaped_val = value.replace("'", "''")
    sql = f"""
    INSERT INTO options (key, value) VALUES ('{escaped_key}', '{escaped_val}')
    ON CONFLICT (key) DO UPDATE SET value = '{escaped_val}';
    """
    run_psql(sql)

def update_production():
    print("=== Updating Production PostgreSQL on saascover-api ===")
    now_ts = int(time.time())

    # 1. Update GroupRatio
    print("1. Updating GroupRatio...")
    raw_gr = get_prod_option("GroupRatio")
    gr = json.loads(raw_gr) if raw_gr else {}
    gr.update(GROUP_RATIOS)
    set_prod_option("GroupRatio", json.dumps(gr))
    
    # 2. Update UserUsableGroups
    print("2. Updating UserUsableGroups...")
    set_prod_option("UserUsableGroups", json.dumps(USABLE_GROUPS))
    
    # 3. Update ModelRatio
    print("3. Updating ModelRatio...")
    raw_mr = get_prod_option("ModelRatio")
    mr = json.loads(raw_mr) if raw_mr else {}
    for m in MODELS:
        mr[m["name"]] = m["ratio"]
    set_prod_option("ModelRatio", json.dumps(mr))
    
    # 4. Update CompletionRatio
    print("4. Updating CompletionRatio...")
    raw_cr = get_prod_option("CompletionRatio")
    cr = json.loads(raw_cr) if raw_cr else {}
    for m in MODELS:
        cr[m["name"]] = m["completion_ratio"]
    set_prod_option("CompletionRatio", json.dumps(cr))
    
    # 5. Clean billing_expr and set billing_mode to ratio
    print("5. Cleaning billing_expr overrides and ensuring ratio billing_mode...")
    raw_expr = get_prod_option("billing_setting.billing_expr")
    expr_map = json.loads(raw_expr) if raw_expr else {}
    for m in MODELS:
        expr_map.pop(m["name"], None)
    set_prod_option("billing_setting.billing_expr", json.dumps(expr_map))
    
    raw_mode = get_prod_option("billing_setting.billing_mode")
    mode_map = json.loads(raw_mode) if raw_mode else {}
    for m in MODELS:
        mode_map[m["name"]] = "ratio"
    set_prod_option("billing_setting.billing_mode", json.dumps(mode_map))
    
    # 6. Update Vendors
    print("6. Updating vendors...")
    for v in VENDORS:
        sql = f"""
        INSERT INTO vendors (id, name, icon, status, created_time, updated_time)
        VALUES ({v['id']}, '{v['name']}', '{v['icon']}', 1, {now_ts}, {now_ts})
        ON CONFLICT (id) DO UPDATE SET name = '{v['name']}', icon = '{v['icon']}', status = 1, updated_time = {now_ts};
        """
        run_psql(sql)
    run_psql("SELECT setval('vendors_id_seq', (SELECT MAX(id) FROM vendors));")

    vendor_name_to_id = {v["name"]: v["id"] for v in VENDORS}
    
    # 7. Update Models table
    print("7. Updating models metadata...")
    for m in MODELS:
        v_id = vendor_name_to_id.get(m["vendor"], 1)
        m_name = m["name"].replace("'", "''")
        m_desc = m["desc"].replace("'", "''")
        m_tags = m["tags"].replace("'", "''")
        
        check = run_psql(f"SELECT id FROM models WHERE model_name = '{m_name}' AND deleted_at IS NULL;")
        if check:
            sql = f"""
            UPDATE models SET
                description = '{m_desc}',
                tags = '{m_tags}',
                vendor_id = {v_id},
                status = 1,
                updated_time = {now_ts}
            WHERE model_name = '{m_name}' AND deleted_at IS NULL;
            """
        else:
            sql = f"""
            INSERT INTO models (model_name, description, tags, vendor_id, status, sync_official, created_time, updated_time)
            VALUES ('{m_name}', '{m_desc}', '{m_tags}', {v_id}, 1, 1, {now_ts}, {now_ts});
            """
        run_psql(sql)

    # 8. Setup Dedicated Channels
    print("8. Setting up dedicated channels with exact route groups...")
    # Alter channels table to allow long group strings
    run_psql('ALTER TABLE channels ALTER COLUMN "group" TYPE character varying(255);')

    # Channels definitions
    openai_standard = [m["name"] for m in MODELS if m["vendor"] == "OpenAI" and m["name"] != "gpt-6-luna"]
    anthropic_standard = [m["name"] for m in MODELS if m["vendor"] == "Anthropic" and m["name"] != "claude-sonnet-5-5"]
    
    channels_def = [
        # id, name, type, models, groups
        (1, "TypeSafe", 1, "jev-1.13.0,jev-1.13", "jev-premium"),
        (2, "Anthropic", 14, ",".join(anthropic_standard), "claude-premium,claude-first-topup,claude-award,claude-stable,claude-tmp-cheap"),
        (3, "OpenAI", 1, ",".join(openai_standard), "openai-award,openai-stable,openai-premium,openai-first-topup"),
        (4, "DeepSeek", 1, "deepseek-v4.1-flash,deepseek-v4-flash,deepseek-v4-pro", "deepseek-stable"),
        (5, "Zhipu AI", 1, "glm-5.3-flash,glm-5.3", "glm-stable"),
        (6, "Alibaba Cloud", 1, "qwen3.8-max", "qwen-stable"),
        (7, "Moonshot AI", 1, "kimi-k3", "kimi-stable"),
        (8, "Google", 1, "gemini-3.8-flash,gemini-3.7-flash,gemini-3.6-flash,gemini-3.5-flash,gemini-3.1-pro-preview", "gemini-stable"),
        (9, "Meta", 1, "muse-spark-1.3", "spark-premium"),
        (10, "xAI", 1, "grok-4.7,grok-4.6,grok-4.5", "grok-award,grok-stable"),
        (11, "OpenAI-Luna", 1, "gpt-6-luna", "openai-award,openai-stable,openai-first-topup"),
        (12, "Anthropic-Sonnet-5-5", 14, "claude-sonnet-5-5", "claude-premium,claude-award,claude-stable"),
    ]

    for cid, cname, ctype, cmodels, cgroups in channels_def:
        sql = f"""
        INSERT INTO channels (id, name, type, status, "key", base_url, models, "group", priority, weight, created_time)
        VALUES ({cid}, '{cname}', {ctype}, 1, 'sk-placeholder-relay', 'https://modelflare.dev', '{cmodels}', '{cgroups}', 0, 0, {now_ts})
        ON CONFLICT (id) DO UPDATE SET
            name = '{cname}',
            type = {ctype},
            status = 1,
            models = '{cmodels}',
            "group" = '{cgroups}';
        """
        run_psql(sql)
    run_psql("SELECT setval('channels_id_seq', (SELECT MAX(id) FROM channels));")

    # 9. Sync abilities table
    print("9. Re-syncing abilities table...")
    # Clear all existing abilities for these models
    m_list_sql = "'" + "','".join([m["name"] for m in MODELS]) + "'"
    run_psql(f"DELETE FROM abilities WHERE model IN ({m_list_sql});")
    
    # Map model to channel ID
    model_to_cid = {}
    for cid, cname, ctype, cmodels, cgroups in channels_def:
        for m in cmodels.split(","):
            model_to_cid[m.strip()] = cid
            
    ability_values = []
    for m in MODELS:
        cid = model_to_cid[m["name"]]
        for g in m["groups"]:
            ability_values.append(f"('{g}', '{m['name']}', {cid}, true, 0, 0)")
            
    # Insert in chunks
    chunk_size = 50
    for i in range(0, len(ability_values), chunk_size):
        chunk = ability_values[i:i+chunk_size]
        run_psql(f"INSERT INTO abilities (\"group\", model, channel_id, enabled, priority, weight) VALUES {','.join(chunk)};")
        
    print(f"Inserted {len(ability_values)} exact abilities.")

    # 10. Restart new-api container to flush cache and reload
    print("10. Restarting new-api container on production...")
    subprocess.run(["ssh", "saascover-api", "docker restart new-api"], check=True)
    print("Waiting 5s for container healthy...")
    time.sleep(5)

def update_local_sqlite():
    db_path = "/Users/noppanan/new-api/data/one-api.db"
    print(f"\n=== Updating local SQLite at {db_path} ===")
    conn = sqlite3.connect(db_path)
    cur = conn.cursor()
    
    def get_opt(k):
        cur.execute("SELECT value FROM options WHERE key = ?", (k,))
        row = cur.fetchone()
        return row[0] if row else ""
        
    def set_opt(k, v):
        cur.execute("INSERT INTO options (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?", (k, v, v))
        
    # GroupRatio
    raw = get_opt("GroupRatio")
    gr = json.loads(raw) if raw else {}
    gr.update(GROUP_RATIOS)
    set_opt("GroupRatio", json.dumps(gr))
    
    # UserUsableGroups
    set_opt("UserUsableGroups", json.dumps(USABLE_GROUPS))
    
    # ModelRatio
    raw = get_opt("ModelRatio")
    mr = json.loads(raw) if raw else {}
    for m in MODELS:
        mr[m["name"]] = m["ratio"]
    set_opt("ModelRatio", json.dumps(mr))
    
    # CompletionRatio
    raw = get_opt("CompletionRatio")
    cr = json.loads(raw) if raw else {}
    for m in MODELS:
        cr[m["name"]] = m["completion_ratio"]
    set_opt("CompletionRatio", json.dumps(cr))
    
    # billing_expr
    raw = get_opt("billing_setting.billing_expr")
    expr = json.loads(raw) if raw else {}
    for m in MODELS:
        expr.pop(m["name"], None)
    set_opt("billing_setting.billing_expr", json.dumps(expr))
    
    conn.commit()
    conn.close()
    print("Local SQLite updated successfully.")

if __name__ == "__main__":
    update_production()
    update_local_sqlite()
    print("\nALL TASKS COMPLETED!")
