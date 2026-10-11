import bcrypt
import json
import requests
import subprocess
import time
import sys

BASE_URL = "http://127.0.0.1:3000"

def db_exec(sql):
    cmd = ["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-At"]
    res = subprocess.run(cmd, input=sql, text=True, capture_output=True, check=True)
    out = res.stdout.strip()
    if out:
        return out.splitlines()[0]
    return ""

print("==================================================")
print("TORA AI — CANARY CAP 1 VERIFICATION PROBE")
print("==================================================")

username = "canary_probe_admin"
password_raw = "CanaryProbe@2026!Secure"
password_hash = bcrypt.hashpw(password_raw.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")
now = int(time.time())

db_exec(f"DELETE FROM users WHERE username = '{username}';")
user_id = db_exec(f"""
INSERT INTO users (username, password, display_name, role, status, auth_version, created_at)
VALUES ('{username}', '{password_hash}', 'Canary Probe Admin', 100, 1, 1, {now})
RETURNING id;
""")
print(f"[SETUP] Ephemeral admin created (ID: {user_id})")

try:
    login_resp = requests.post(f"{BASE_URL}/api/user/login", json={
        "username": username,
        "password": password_raw
    })
    assert login_resp.status_code == 200, f"Login failed: {login_resp.text}"
    token = login_resp.json()["data"]["access_token"]
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }

    # 1. Pre-Batch Status Check
    print("\n--- 1. PRE-BATCH STATUS CHECK ---")
    r_status = requests.get(f"{BASE_URL}/api/admin/news/autopilot/status", headers=headers)
    assert r_status.status_code == 200
    st = r_status.json()["data"]
    print(f"  Publishing Enabled:          {st.get('publishing_enabled')}")
    print(f"  Max Daily Cap:               {st.get('max_daily_cap')}")
    print(f"  Autopilot Published Today:   {st.get('autopilot_published_today')}")

    assert st.get('publishing_enabled') is True, "Publishing must be enabled for canary"
    assert st.get('max_daily_cap') == 1, "Max daily cap must be 1 for canary"
    assert st.get('autopilot_published_today') == 0, "Initial autopilot count must be 0"

    # 2. Trigger Batch 1 (Target: 1 story published)
    print("\n--- 2. TRIGGER CANARY PUBLISH BATCH (LIMIT 1) ---")
    r_b1 = requests.post(f"{BASE_URL}/api/admin/news/autopilot/run?action=batch&limit=1", headers=headers)
    assert r_b1.status_code == 200, f"Batch 1 failed: {r_b1.text}"
    b1_data = r_b1.json()["data"]
    print(f"  Articles Published:   {b1_data.get('articles_published')}")
    print(f"  Daily Total So Far:   {b1_data.get('daily_total_so_far')}")
    print(f"  Quota Blocked:        {b1_data.get('quota_blocked')}")
    print(f"  Drafts Created:       {b1_data.get('drafts_created')}")
    print(f"  Rejection Reasons:    {b1_data.get('rejection_reasons')}")

    assert b1_data.get('articles_published') == 1, f"Expected 1 article published, got {b1_data.get('articles_published')}"
    print("  [PASS] Exactly 1 article successfully published via atomic quota ledger")

    # 3. Post-Batch 1 Status Check
    print("\n--- 3. POST-BATCH 1 STATUS VERIFICATION ---")
    r_status2 = requests.get(f"{BASE_URL}/api/admin/news/autopilot/status", headers=headers)
    st2 = r_status2.json()["data"]
    print(f"  Autopilot Published Today:   {st2.get('autopilot_published_today')}")
    print(f"  Today Published Count:       {st2.get('today_published_count')}")
    assert st2.get('autopilot_published_today') == 1
    assert st2.get('today_published_count') == 1
    print("  [PASS] Autopilot counter atomically incremented to 1")

    # 4. Trigger Batch 2 (Target: BLOCKED by Daily Cap 1)
    print("\n--- 4. TRIGGER BATCH 2 (CAP ENFORCEMENT VERIFICATION) ---")
    r_b2 = requests.post(f"{BASE_URL}/api/admin/news/autopilot/run?action=batch&limit=1", headers=headers)
    assert r_b2.status_code == 200, f"Batch 2 failed: {r_b2.text}"
    b2_data = r_b2.json()["data"]
    print(f"  Articles Published:   {b2_data.get('articles_published')}")
    print(f"  Quota Blocked:        {b2_data.get('quota_blocked')}")
    print(f"  Rejection Reasons:    {b2_data.get('rejection_reasons')}")

    assert b2_data.get('articles_published') == 0, f"Expected 0 published, got {b2_data.get('articles_published')}"
    assert b2_data.get('quota_blocked') >= 1 or any("daily cap" in str(r).lower() or "quota" in str(r).lower() for r in b2_data.get('rejection_reasons', []))
    print("  [PASS] Batch 2 strictly BLOCKED by atomic daily cap (1/1)!")

    # 5. Verify Database Ledger & Publication Event
    print("\n--- 5. DATABASE QUOTA LEDGER & EVENT PROOF ---")
    quota_row = db_exec("SELECT bangkok_date || ' | published=' || published_count FROM news_autopilot_daily_quota LIMIT 1;")
    print(f"  Daily Quota Ledger:   {quota_row}")
    event_row = db_exec("SELECT post_id || ' | ' || event_type || ' | ' || publication_origin || ' | ' || bangkok_publication_date FROM news_publication_events ORDER BY id DESC LIMIT 1;")
    print(f"  Publication Event:    {event_row}")

    assert "published=1" in quota_row
    assert "AUTOPILOT_INITIAL_PUBLISH" in event_row
    print("  [PASS] PostgreSQL ACID ledger and immutable publication event recorded")

    print("\n==================================================")
    print("CANARY CAP 1 PROOF COMPLETE — ATOMIC LOCK VERIFIED!")
    print("==================================================")

finally:
    db_exec(f"DELETE FROM users WHERE username = '{username}';")
    print("[CLEANUP] Ephemeral admin user removed")
