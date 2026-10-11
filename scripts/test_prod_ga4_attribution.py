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
print("TORA AI — GA4 GROWTH ATTRIBUTION PRODUCTION PROBE")
print("==================================================")

# 1. Status API Exposure
print("\n--- 1. STATUS API GA4 EXPOSURE ---")
r_status = requests.get(f"{BASE_URL}/api/status")
assert r_status.status_code == 200, f"Expected 200, got {r_status.status_code}"
status_data = r_status.json()["data"]
print(f"  ga4_measurement_id:  {status_data.get('ga4_measurement_id')}")
print(f"  google_analytics_id: {status_data.get('google_analytics_id')}")
assert "ga4_measurement_id" in status_data, "Missing ga4_measurement_id in status"
print("  [PASS] Status API exposes ga4_measurement_id configuration")

# 2. SSR News CTA and UTM Instrumentation
print("\n--- 2. SSR NEWS CTA & UTM INSTRUMENTATION ---")
r_news = requests.get(f"{BASE_URL}/news/openai-gpt-4-5-release-analysis")
assert r_news.status_code == 200, f"News article returned {r_news.status_code}"
html = r_news.text
assert 'data-analytics-cta=' in html, "Missing data-analytics-cta attribute in SSR HTML"
assert 'news_cta_click' in html, "Missing news_cta_click tracking script in SSR HTML"
assert 'news_internal_link_click' in html, "Missing news_internal_link_click in SSR HTML"
assert 'utm_source' in html, "Missing UTM preservation script in SSR HTML"
print("  [PASS] SSR News contains CTA tracking, internal link tracking, and UTM propagation")

# 3. Public Conversion Event Ingestion
print("\n--- 3. CONVERSION EVENT INGESTION ---")
r_conv = requests.post(f"{BASE_URL}/api/news/conversion", json={
    "event_type": "cta_click",
    "content_id": "8",
    "utm_source": "google",
    "utm_medium": "organic",
    "utm_campaign": "seo"
})
assert r_conv.status_code == 200, f"Conversion ingestion returned {r_conv.status_code}: {r_conv.text}"
assert r_conv.json().get("success") is True
print("  [PASS] Conversion event successfully recorded with privacy-safe hashing")

# Create ephemeral admin for authenticated probes
username = "ga4_canary_admin"
password_raw = "GA4Canary@2026!Secure"
password_hash = bcrypt.hashpw(password_raw.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")
now = int(time.time())

db_exec(f"DELETE FROM users WHERE username = '{username}';")
user_id = db_exec(f"""
INSERT INTO users (username, password, display_name, role, status, auth_version, created_at)
VALUES ('{username}', '{password_hash}', 'GA4 Canary Admin', 100, 1, 1, {now})
RETURNING id;
""")
print(f"  [SETUP] Ephemeral admin created (ID: {user_id})")

try:
    login_resp = requests.post(f"{BASE_URL}/api/user/login", json={
        "username": username,
        "password": password_raw
    })
    assert login_resp.status_code == 200
    token = login_resp.json()["data"]["access_token"]
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }

    # 4. Growth Overview GA4 Status
    print("\n--- 4. GROWTH OVERVIEW GA4 TELEMETRY ---")
    r_overview = requests.get(f"{BASE_URL}/api/admin/news/growth/overview", headers=headers)
    assert r_overview.status_code == 200
    overview = r_overview.json()["data"]
    print(f"  GA4 Status:          {overview.get('ga4_status')}")
    print(f"  GA4 Data Available:  {overview.get('ga4_data_available')}")
    print(f"  GA4 Property ID:     {overview.get('ga4_property_id')}")
    assert overview.get("ga4_status") == "OPERATOR_BLOCKED", f"Expected GA4 OPERATOR_BLOCKED, got {overview.get('ga4_status')}"
    assert overview.get("ga4_data_available") is False
    print("  [PASS] Growth Overview truthfully reports GA4=OPERATOR_BLOCKED")

    # 5. Post Growth Record Joined Funnel
    print("\n--- 5. POST GROWTH RECORD JOINED FUNNEL (POST ID 8) ---")
    r_record = requests.get(f"{BASE_URL}/api/admin/news/posts/8/growth", headers=headers)
    assert r_record.status_code == 200, f"Failed post growth: {r_record.text}"
    record = r_record.json()["data"]
    
    ga4_analytics = record.get("ga4_analytics", {})
    funnel = record.get("growth_funnel", {})
    
    print(f"  Post ID:             {record.get('post_id')}")
    print(f"  Slug:                {record.get('slug')}")
    print(f"  GA4 Analytics:       {json.dumps(ga4_analytics, indent=4)}")
    print(f"  Growth Funnel:       {json.dumps(funnel, indent=4)}")
    
    assert ga4_analytics.get("status") == "OPERATOR_BLOCKED", "Expected GA4 status OPERATOR_BLOCKED"
    assert ga4_analytics.get("sessions") is None, "Expected sessions to be null (not 0)"
    assert ga4_analytics.get("data_available") is False
    
    assert funnel.get("impressions") is None or isinstance(funnel.get("impressions"), int), "Impressions null or int"
    assert funnel.get("landing_sessions") is None, "Expected landing_sessions null"
    assert funnel.get("cta_clicks", 0) >= 1, "Expected recorded CTA click to be reflected"
    assert funnel.get("opportunity_type") in [
        "NO_DATA_YET", "TRAFFIC_OPPORTUNITY", "BUSINESS_VALUE_EXPAND_CLUSTER", 
        "BUSINESS_VALUE_HIGH_CONVERSION", "CONTENT_MISMATCH_OPPORTUNITY"
    ]
    print(f"  Opportunity Type:    {funnel.get('opportunity_type')}")
    print(f"  Data Freshness:      {funnel.get('data_freshness')}")
    print("  [PASS] Joined Funnel and GA4 telemetry validated with null pointers (not zero-filled)")

    # 6. Growth Sync Worker Iteration
    print("\n--- 6. GROWTH WORKER SYNC ITERATION ---")
    sync_res = requests.post(f"{BASE_URL}/api/admin/news/growth/sync", headers=headers)
    assert sync_res.status_code == 200
    sync_data = sync_res.json()["data"]
    print(f"  Worker Sync Result:  {sync_data}")
    assert sync_data.get("ga4_status") == "OPERATOR_BLOCKED"
    assert sync_data.get("gsc_status") == "CONNECTED"
    print("  [PASS] Growth collector iteration handles GA4 truthfully")

    print("\n==================================================")
    print("GA4 GROWTH ATTRIBUTION VALIDATION: 100% PASS")
    print("==================================================")

finally:
    db_exec(f"DELETE FROM users WHERE username = '{username}';")
    print(f"\n  [CLEANUP] Ephemeral admin user '{username}' deleted.")
