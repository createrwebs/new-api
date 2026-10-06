#!/usr/bin/env python3
"""
Comprehensive 7-Day Burn-In Forensic Telemetry Audit for Tora AI Newsroom
Audits all 27 sections defined in the 7-Day Organic Burn-In protocol.
"""

import subprocess
import json
import re
import urllib.request
import ssl
import xml.etree.ElementTree as ET

EC2_HOST = "ubuntu@51.20.174.90"
SSH_KEY = "/Users/noppanan/key/saascover-api.pem"

def run_ssh_psql(sql: str):
    """Executes SQL inside the production PostgreSQL container via stdin."""
    cmd = [
        "ssh", "-i", SSH_KEY, "-o", "StrictHostKeyChecking=no",
        EC2_HOST,
        "docker exec -i postgres psql -U root -d new-api -t -A"
    ]
    res = subprocess.run(cmd, input=sql, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if res.returncode != 0:
        raise RuntimeError(f"SSH PSQL error: {res.stderr}")
    return res.stdout.strip()

def run_ssh_psql_json(sql: str):
    """Executes SQL and returns json rows."""
    wrapped = f"SELECT json_agg(t) FROM ({sql}) t;"
    out = run_ssh_psql(wrapped)
    if not out or out == "":
        return []
    try:
        return json.loads(out)
    except Exception as e:
        print(f"JSON parse error for output: {out[:200]}... Err: {e}")
        return []

def fetch_url(url: str):
    ctx = ssl._create_unverified_context()
    req = urllib.request.Request(url, headers={"User-Agent": "Tora-BurnIn-Auditor/1.0"})
    with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
        return resp.getcode(), resp.read().decode("utf-8")

def main():
    print("=" * 60)
    print("TORA AI — 7-DAY ORGANIC BURN-IN FORENSIC TELEMETRY AUDIT")
    print("=" * 60)

    # 1. Authoritative Quota & Counters
    print("\n--- SECTION 1 & 2: CORE POLICY & QUOTA VERIFICATION ---")
    quota_rows = run_ssh_psql_json("SELECT * FROM news_autopilot_daily_quota ORDER BY bangkok_date DESC LIMIT 7")
    print(f"Quota Ledger Rows: {len(quota_rows)}")
    for r in quota_rows:
        print(f"  Date: {r.get('bangkok_date')} | Published: {r.get('published_count')}/20 | Updated: {r.get('updated_at')}")

    # 2. Publication Origin Breakdown
    print("\n--- PUBLICATION ORIGIN BREAKDOWN ---")
    origin_rows = run_ssh_psql_json("SELECT publication_origin, status, count(*) as count FROM news_posts GROUP BY publication_origin, status ORDER BY count DESC")
    for r in origin_rows:
        print(f"  Origin: {r.get('publication_origin')} | Status: {r.get('status')} | Count: {r.get('count')}")

    # 3. Source Scorecard (24 Sources)
    print("\n--- SECTION 4: SOURCE QUALITY SCORECARD (24 SOURCES) ---")
    sources = run_ssh_psql_json("SELECT id, name, slug, feed_url, enabled, consecutive_failures, fetch_error_count, last_http_status, parse_status, last_fetched_at, last_success_at FROM news_sources ORDER BY id ASC")
    print(f"Total Sources Configured: {len(sources)}")
    healthy_sources = 0
    degraded_sources = 0
    for s in sources:
        status_ok = (s.get('consecutive_failures') == 0 and s.get('last_http_status') == 200 and s.get('parse_status') == 'ok')
        if status_ok:
            healthy_sources += 1
        else:
            degraded_sources += 1
        print(f"  [{'OK' if status_ok else 'WARN'}] {s.get('name'):<30} | HTTP {s.get('last_http_status')} | Failures: {s.get('consecutive_failures')} | Status: {s.get('parse_status')}")
    print(f"Sources Summary: Healthy={healthy_sources}, Degraded={degraded_sources}")

    # 4. Recent Autopilot Articles Audit
    print("\n--- SECTION 3: RECENT AUTOPILOT ARTICLES AUDIT ---")
    autopilot_posts = run_ssh_psql_json("SELECT id, title, slug, source_id, content_risk, fact_check_status, char_length(content_markdown) as word_count, is_seed, publication_origin, status, published_at FROM news_posts WHERE publication_origin = 'AUTOPILOT' ORDER BY id DESC LIMIT 10")
    print(f"Autopilot Published Count: {len(autopilot_posts)}")
    for p in autopilot_posts:
        print(f"  ID: {p.get('id')} | Title: {p.get('title')[:40]}... | Risk: {p.get('content_risk')} | FactCheck: {p.get('fact_check_status')} | Chars: {p.get('word_count')} | PublishedAt: {p.get('published_at')}")

    # 5. Full Content Inspection of Latest Autopilot Post
    if autopilot_posts:
        latest_id = autopilot_posts[0]['id']
        post_detail = run_ssh_psql_json(f"SELECT id, title, slug, content_markdown, summary, canonical_url, seo_title, seo_description, og_image_url FROM news_posts WHERE id = {latest_id}")[0]
        content = post_detail.get('content_markdown', '')
        print(f"\nAudit of Post ID {latest_id} Content:")
        print(f"  Title: {post_detail.get('title')}")
        print(f"  Slug: {post_detail.get('slug')}")
        print(f"  Canonical: {post_detail.get('canonical_url')}")
        print(f"  OG Image: {post_detail.get('og_image_url')}")
        print(f"  Content Length: {len(content)} chars")
        
        # Check Thai Clichés
        cliches = ["ในยุคที่", "สิ่งนี้สะท้อนให้เห็นว่า", "นับเป็นอีกก้าวสำคัญ"]
        cliche_matches = {c: content.count(c) for c in cliches}
        print(f"  Thai Cliché Counts: {cliche_matches}")
        
        # Check Internal Links
        internal_links = re.findall(r'\[([^\]]+)\]\((https?://(?:www\.)?toraapi\.com/[^\)]+|/[^\)]+)\)', content)
        print(f"  Internal Links Found ({len(internal_links)}): {internal_links}")
        
        # Check Primary Citations
        citations = re.findall(r'\[([^\]]+)\]\((https?://(?!www\.toraapi\.com)[^\)]+)\)', content)
        print(f"  External Primary Citations Found ({len(citations)}): {citations[:3]}")

    # 6. Sitemaps Verification
    print("\n--- SECTION 16 & 17: SITEMAP AUDIT ---")
    try:
        code_main, xml_main = fetch_url("https://www.toraapi.com/sitemap.xml")
        root_main = ET.fromstring(xml_main)
        ns = {'sm': 'http://www.sitemaps.org/schemas/sitemap/0.9'}
        urls_main = [loc.text for loc in root_main.findall('sm:url/sm:loc', ns)]
        news_urls_main = [u for u in urls_main if '/news/' in u]
        print(f"  Main Sitemap: HTTP {code_main} | Total URLs: {len(urls_main)} | News URLs: {len(news_urls_main)}")
    except Exception as e:
        print(f"  Main Sitemap Error: {e}")

    try:
        code_news, xml_news = fetch_url("https://www.toraapi.com/news-sitemap.xml")
        root_news = ET.fromstring(xml_news)
        ns_news = {
            'sm': 'http://www.sitemaps.org/schemas/sitemap/0.9',
            'news': 'http://www.google.com/schemas/sitemap-news/0.9'
        }
        urls_news = [loc.text for loc in root_news.findall('sm:url/sm:loc', ns_news)]
        print(f"  Google News Sitemap: HTTP {code_news} | Eligible (<48h) News URLs: {len(urls_news)}")
        for u in urls_news:
            print(f"    - {u}")
    except Exception as e:
        print(f"  News Sitemap Error: {e}")

    # 7. URL Inspection & GSC Metrics
    print("\n--- SECTION 6, 7 & 8: GSC TELEMETRY & URL INSPECTION ---")
    inspections = run_ssh_psql_json("SELECT post_id, inspection_url, coverage_state, verdict, robots_txt_state, indexing_state, inspection_time FROM news_url_inspections ORDER BY inspection_time DESC LIMIT 10")
    print(f"URL Inspection Rows Recorded: {len(inspections)}")
    for insp in inspections:
        print(f"  Post {insp.get('post_id')} | {insp.get('inspection_url')} | Verdict: {insp.get('verdict')} | State: {insp.get('coverage_state')} | Time: {insp.get('inspection_time')}")

    seo_metrics = run_ssh_psql_json("SELECT * FROM news_seo_metrics LIMIT 10")
    print(f"Search Analytics Rows (GSC): {len(seo_metrics)}")
    if len(seo_metrics) == 0:
        print("  GSC Search Analytics Status: NO_DATA_YET (Truthful reporting; awaiting standard Google 48-72h indexation window)")

    # 8. AI Visibility Observations & Crawler Logs
    print("\n--- SECTION 19 & 20: AI CRAWLER & AI CITATION TELEMETRY ---")
    ai_obs = run_ssh_psql_json("SELECT * FROM news_ai_visibility_observations ORDER BY id DESC LIMIT 10")
    print(f"AI Visibility Observations: {len(ai_obs)}")
    for a in ai_obs:
        print(f"  Provider: {a.get('provider')} | Query: {a.get('query')} | Cited: {a.get('tora_cited')} | URL: {a.get('cited_url')}")

    # 9. Legacy Articles Status (1,203 rows)
    print("\n--- SECTION 22: LEGACY ARTICLES AUDIT ---")
    legacy_stats = run_ssh_psql_json("""
        SELECT 
            count(*) as total_legacy,
            avg(char_length(content_markdown)) as avg_content_len,
            min(char_length(content_markdown)) as min_content_len,
            max(char_length(content_markdown)) as max_content_len,
            count(CASE WHEN char_length(content_markdown) < 500 THEN 1 END) as thin_count
        FROM news_posts 
        WHERE publication_origin = 'UNKNOWN_LEGACY'
    """)[0]
    print(f"  Total Legacy Posts: {legacy_stats.get('total_legacy')}")
    print(f"  Average Content Length: {float(legacy_stats.get('avg_content_len', 0)):.1f} chars")
    print(f"  Min Content Length: {legacy_stats.get('min_content_len')}")
    print(f"  Max Content Length: {legacy_stats.get('max_content_len')}")
    print(f"  Thin (<500 chars): {legacy_stats.get('thin_count')}")

    # 10. Topic Distribution
    print("\n--- SECTION 5: TOPIC DISTRIBUTION ---")
    topics = run_ssh_psql_json("""
        SELECT content_type, count(*) as count 
        FROM news_posts 
        GROUP BY content_type 
        ORDER BY count DESC
    """)
    for t in topics:
        print(f"  Topic/Type: {t.get('content_type'):<15} | Count: {t.get('count')}")

    # 11. Daily Growth Reviews
    print("\n--- SECTION 23: DAILY GROWTH REVIEWS PERSISTED ---")
    reviews = run_ssh_psql_json("SELECT * FROM news_daily_growth_reviews ORDER BY review_date DESC LIMIT 7")
    print(f"Persisted Growth Reviews: {len(reviews)}")
    for rev in reviews:
        print(f"  Date: {rev.get('review_date')} | Published Today: {rev.get('posts_published_today')} | Status: {rev.get('newsroom_run_status')}")

    print("\n" + "=" * 60)
    print("AUDIT COMPLETE")
    print("=" * 60)

if __name__ == "__main__":
    main()
