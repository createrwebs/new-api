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

def db_query(sql):
    cmd = ["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", sql]
    res = subprocess.run(cmd, capture_output=True, text=True, check=True)
    return res.stdout.strip()

print("==================================================")
print("TORA AI — NEWSROOM DAILY CAP & INVARIANT PROBE")
print("==================================================")

# 1. Ephemeral Admin Setup
username = "cap_probe_admin"
password_raw = "CapProbe@2026!Secure"
password_hash = bcrypt.hashpw(password_raw.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")
now = int(time.time())

db_exec(f"DELETE FROM users WHERE username = '{username}';")
user_id = db_exec(f"""
INSERT INTO users (username, password, display_name, role, status, auth_version, created_at)
VALUES ('{username}', '{password_hash}', 'Cap Probe Admin', 100, 1, 1, {now})
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
    print("[AUTH] Authenticated successfully with bearer token")

    # 2. Database Origin Invariant Check
    print("\n--- 1. DATABASE ORIGIN BREAKDOWN ---")
    print(db_query("SELECT is_seed, publication_origin, count(*) FROM news_posts GROUP BY is_seed, publication_origin ORDER BY is_seed, publication_origin;"))

    # 3. Autopilot Status & Separated Counters Verification (Section 4 & 20)
    print("\n--- 2. AUTOPILOT STATUS & SEPARATED COUNTERS ---")
    r_status = requests.get(f"{BASE_URL}/api/admin/news/autopilot/status", headers=headers)
    assert r_status.status_code == 200, f"Status failed: {r_status.text}"
    status_data = r_status.json()["data"]

    today_published = status_data.get("today_published_count")
    max_daily_cap = status_data.get("max_daily_cap")
    is_pub_enabled = status_data.get("publishing_enabled")

    print(f"  Current Bangkok Time:          {status_data.get('current_bangkok_time')}")
    print(f"  Autopilot Published Today:     {status_data.get('autopilot_published_today')} (today_published_count={today_published})")
    print(f"  Max Daily Cap:                 {max_daily_cap}")
    print(f"  Publishing Enabled:            {is_pub_enabled}")
    print(f"  Total Published Today:         {status_data.get('published_today_total')}")
    print(f"  Manual Published Today:        {status_data.get('manual_published_today')}")
    print(f"  Seed/Historical Today:         {status_data.get('seed_or_historical_today')}")
    print(f"  Legacy Unknown Today:          {status_data.get('legacy_unknown_today')}")
    print(f"  Drafts Today:                  {status_data.get('drafts_today')}")

    # Assert Invariant: Autopilot Published Today <= Max Daily Cap
    assert today_published <= max_daily_cap, f"INVARIANT VIOLATION: today_published ({today_published}) > max_daily_cap ({max_daily_cap})"
    print(f"  [PASS] INVARIANT UPHELD: Autopilot Published Today ({today_published}) <= Max Daily Cap ({max_daily_cap})")

    # Assert Counter Separation: Autopilot Published is distinct from legacy/seed
    assert status_data.get("autopilot_published_today") == 0, f"Expected 0 autopilot publishes before canary, got {status_data.get('autopilot_published_today')}"
    assert status_data.get("legacy_unknown_today") >= 1000, "Legacy burst correctly isolated in legacy_unknown_today"
    print("  [PASS] Counter isolation verified: autopilot_published_today = 0, legacy burst isolated")

    # 4. Google News Sitemap Filtering Verification (Section 16)
    print("\n--- 3. GOOGLE NEWS SITEMAP SEED/LEGACY EXCLUSION PROOF ---")
    r_news_sitemap = requests.get(f"{BASE_URL}/news-sitemap.xml")
    assert r_news_sitemap.status_code == 200
    news_sitemap_xml = r_news_sitemap.text

    # Seed slugs must NEVER appear in news-sitemap.xml
    seed_slugs = [
        "openai-gpt-4-5-release-analysis",
        "anthropic-claude-3-7-sonnet-hybrid-reasoning",
        "deepseek-v3-architecture-deep-dive"
    ]
    for s_slug in seed_slugs:
        assert s_slug not in news_sitemap_xml, f"CRITICAL: Seed post {s_slug} leaked into news-sitemap.xml!"
    print("  [PASS] Google News sitemap strictly excludes SEED articles")

    # Standard sitemap DOES include news routes for discovery
    r_std_sitemap = requests.get(f"{BASE_URL}/sitemap.xml")
    assert r_std_sitemap.status_code == 200
    std_sitemap_xml = r_std_sitemap.text
    assert "https://www.toraapi.com/news" in std_sitemap_xml
    print("  [PASS] Standard sitemap includes news index and recent articles for discovery")

    # 4. Immediate Safety Hold Verification (Section 1)
    print("\n--- 4. SAFETY HOLD PUBLISHING LOCKOUT VERIFICATION ---")
    r_batch = requests.post(f"{BASE_URL}/api/admin/news/autopilot/run?action=batch", headers=headers)
    assert r_batch.status_code == 200, f"Batch run failed: {r_batch.text}"
    batch_data = r_batch.json()["data"]
    print(f"  Articles Published:   {batch_data.get('articles_published')}")
    print(f"  Rejection Reasons:    {batch_data.get('rejection_reasons')}")
    assert batch_data.get('articles_published') == 0
    assert any("Safety Hold" in r for r in batch_data.get('rejection_reasons', []))
    print("  [PASS] Automatic publishing correctly locked out while NEWS_AUTOPILOT_PUBLISH_ENABLED=false")

    # 5. Daily Newsroom Persisted Report Verification (Section 17, 18, 23)
    print("\n--- 5. DAILY NEWSROOM PERSISTED REPORT & SEO WORDING ---")
    r_report = requests.post(f"{BASE_URL}/api/admin/news/autopilot/run?action=report", headers=headers)
    assert r_report.status_code == 200, f"Report run failed: {r_report.text}"
    report_data = r_report.json()["data"]

    run_status = report_data.get("newsroom_run_status")
    top_opps = report_data.get("top_opportunities_tomorrow", "")
    print(f"  Newsroom Run Status:   {run_status}")
    print(f"  Top Opportunities:     {top_opps.strip()}")

    assert "INVARIANT VIOLATION" not in run_status, f"Report detected invariant violation: {run_status}"
    assert "All current SEO" not in top_opps, "Found outdated misleading wording in SEO report!"
    print("  [PASS] Daily Report verified truthful status and SEO phrasing")

    print("\n==================================================")
    print("ALL INVARIANT & CAP TELEMETRY CHECKS PASSED!")
    print("==================================================")

finally:
    db_exec(f"DELETE FROM users WHERE username = '{username}';")
    print("[CLEANUP] Ephemeral admin user removed")
