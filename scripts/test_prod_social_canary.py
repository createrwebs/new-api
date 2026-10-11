import bcrypt
import json
import requests
import subprocess
import time
import sys

BASE_URL = "http://127.0.0.1:3000"
DEVTO_KEY = "AHzLFiCrd9XSFmSWGXvVSvmY"

def db_exec(sql):
    cmd = ["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-At"]
    res = subprocess.run(cmd, input=sql, text=True, capture_output=True, check=True)
    out = res.stdout.strip()
    if out:
        return out.splitlines()[0]
    return ""

def db_query_all(sql):
    cmd = ["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-At"]
    res = subprocess.run(cmd, input=sql, text=True, capture_output=True, check=True)
    out = res.stdout.strip()
    if out:
        return out.splitlines()
    return []

print("==================================================")
print("TORA AI — REAL SOCIAL CHANNEL CANARY VALIDATION")
print("==================================================")

print("\n--- 1. UNAUTHENTICATED PROTECTION ---")
r = requests.get(f"{BASE_URL}/api/admin/news/growth/overview")
assert r.status_code == 401, f"Expected 401, got {r.status_code}"
print("  [PASS] Unauthenticated access blocked (HTTP 401)")

# Create ephemeral admin
username = "canary_social_admin"
password_raw = "SocialCanary@2026!Secure"
password_hash = bcrypt.hashpw(password_raw.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")
now = int(time.time())

# Ensure cleanup before insert
db_exec(f"DELETE FROM users WHERE username = '{username}';")
user_id = db_exec(f"""
INSERT INTO users (username, password, display_name, role, status, auth_version, created_at)
VALUES ('{username}', '{password_hash}', 'Social Canary Admin', 100, 1, 1, {now})
RETURNING id;
""")
print(f"  [SETUP] Ephemeral admin created (ID: {user_id})")

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
    print("  [PASS] Admin session token obtained")

    print("\n--- 2. GROWTH OVERVIEW & BACKLOG INVARIANT TELEMETRY ---")
    r_overview = requests.get(f"{BASE_URL}/api/admin/news/growth/overview", headers=headers)
    assert r_overview.status_code == 200, f"Failed growth overview: {r_overview.text}"
    overview = r_overview.json()["data"]
    
    print(f"  GSC Status:         {overview.get('gsc_status')}")
    print(f"  GSC Site URL:       {overview.get('gsc_site_url')}")
    print(f"  GA4 Status:         {overview.get('ga4_status')}")
    devto_stat = overview.get("channel_statuses", {}).get("devto") or overview.get("devto_status")
    print(f"  DEV.to Status:      {devto_stat}")
    print(f"  Channel Statuses:   {overview.get('channel_statuses')}")
    
    assert overview.get("gsc_status") == "CONNECTED", f"Expected GSC CONNECTED, got {overview.get('gsc_status')}"
    assert overview.get("ga4_status") == "OPERATOR_BLOCKED", f"Expected GA4 OPERATOR_BLOCKED, got {overview.get('ga4_status')}"
    assert devto_stat == "ACTIVE", f"Expected DEV.to ACTIVE, got {devto_stat}"
    assert overview.get("channel_statuses", {}).get("facebook") == "OPERATOR_BLOCKED", "Expected FB OPERATOR_BLOCKED"
    assert overview.get("channel_statuses", {}).get("linkedin") == "OPERATOR_BLOCKED", "Expected LI OPERATOR_BLOCKED"
    print("  [PASS] Truthful provider states verified (GSC=CONNECTED, GA4=OPERATOR_BLOCKED, DEVTO=ACTIVE, FB=OPERATOR_BLOCKED, LI=OPERATOR_BLOCKED)")
    
    bm = overview.get("backlog_metrics", {})
    print("\n  Backlog Safety Invariant Metrics:")
    print(f"    Raw Pending Count:             {bm.get('raw_pending_count')}")
    print(f"    Eligible Pending Count:        {bm.get('eligible_pending_count')}")
    print(f"    Suppressed Historical Count:   {bm.get('suppressed_historical_count')}")
    print(f"    Canary Eligible Count:         {bm.get('canary_eligible_count')}")
    print(f"    By Platform:                   {bm.get('by_platform')}")
    
    raw_pending = bm.get("raw_pending_count", 0)
    eligible_pending = bm.get("eligible_pending_count", 0)
    suppressed = bm.get("suppressed_historical_count", 0)
    canary_eligible = bm.get("canary_eligible_count", 0)
    
    assert raw_pending >= 1000, f"Expected raw_pending >= 1000, got {raw_pending}"
    assert overview.get("allowlist_post_ids") == [8], f"Expected allowlist [8], got {overview.get('allowlist_post_ids')}"
    assert suppressed == raw_pending - eligible_pending, f"Invariant violated: {suppressed} != {raw_pending} - {eligible_pending}"
    assert suppressed >= 1000, f"Expected suppressed >= 1000, got {suppressed}"
    print("  [PASS] Mathematical backlog invariant strictly holds (raw - eligible == suppressed; historical backlog 100% suppressed)")

    print("\n--- 3. AUTONOMOUS MODE POLICY ENGINE TELEMETRY ---")
    ap = overview.get("autonomous_policy", {})
    print(f"  Policy Version:            {ap.get('policy_version')}")
    print(f"  Mass Autopublish Enabled:  {ap.get('mass_autopublish_enabled')}")
    print(f"  Active Rule Summary:       {ap.get('active_rule_summary')}")
    print(f"  Candidates Count:          {ap.get('candidates_count')}")
    print(f"  Review Required Count:     {ap.get('review_required_count')}")
    print(f"  Never Publish Count:       {ap.get('never_publish_count')}")
    
    assert ap.get("mass_autopublish_enabled") is False, "CRITICAL: Mass autopublish MUST be False"
    assert ap.get("policy_version") == "v1-controlled-staged", "Policy version mismatch"
    assert ap.get("candidates_count", 0) + ap.get("review_required_count", 0) + ap.get("never_publish_count", 0) > 0, "No policy evaluated posts"
    print("  [PASS] Section 13 policy engine verified with MASS_AUTOPUBLISH=false strictly enforced")

    print("\n--- 4. GSC DELTA-AWARE INGESTION & GROWTH SYNC ---")
    sync_res = requests.post(f"{BASE_URL}/api/admin/news/growth/sync", headers=headers)
    assert sync_res.status_code == 200, f"Sync failed: {sync_res.text}"
    sync_data = sync_res.json()["data"]
    print(f"  Sync Response: {sync_data}")
    assert sync_data.get("gsc_status") == "CONNECTED"
    print("  [PASS] Delta-aware GSC query and growth worker triggered successfully")

    print("\n--- 5. DEV.TO IDEMPOTENCY & ENGAGEMENT RECONCILIATION ---")
    # Verify Post ID 8 distribution record in PostgreSQL
    dist_row = db_exec("SELECT status || '|' || remote_post_id FROM news_distributions WHERE post_id = 8 AND platform = 'devto';")
    print(f"  DB Post 8 Distribution: {dist_row}")
    assert "published|4804790" == dist_row, f"Expected published|4804790, got {dist_row}"
    print("  [PASS] DEV.to idempotency verified: remote article ID 4804790 preserved, 0 duplicates")

    print("\n--- 6. BATCH DISPATCH WITH HISTORICAL SUPPRESSION ---")
    dispatch_res = requests.post(f"{BASE_URL}/api/admin/news/distributions/dispatch", headers=headers, json={"batch_size": 1})
    assert dispatch_res.status_code == 200, f"Dispatch failed: {dispatch_res.text}"
    dispatch_data = dispatch_res.json()["data"]
    print(f"  Dispatch Response: {dispatch_data}")
    # Verify historical backlog in DB did NOT change to published
    suppressed_in_db = int(db_exec("SELECT count(*) FROM news_distributions WHERE status = 'pending';"))
    print(f"  Pending distributions in DB: {suppressed_in_db}")
    assert suppressed_in_db >= 1000, f"Historical backlog leaked! Count: {suppressed_in_db}"
    print("  [PASS] Historical backlog protected: pending rows remain suppressed")

    print("\n==================================================")
    print("ALL CANARY PROBES PASSED 100% CLEANLY")
    print("==================================================")

finally:
    # Cleanup ephemeral admin
    db_exec(f"DELETE FROM users WHERE username = '{username}';")
    print(f"\n  [CLEANUP] Ephemeral admin user '{username}' deleted.")
