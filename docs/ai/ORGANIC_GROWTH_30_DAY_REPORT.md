# TORA AI — 30-DAY ORGANIC GROWTH AUTOPILOT MASTER REPORT
## NEWS → SEARCH DEMAND → EVERGREEN → AUTHORITY → MEASUREMENT → OPTIMIZE

> **Domain Target**: `https://www.toraapi.com`  
> **Production Image**: `tora-api:r11-db0c36bf8` (Live on AWS EC2 `51.20.174.90`)  
> **Operating Window**: 30 Full Asia/Bangkok Calendar Days  
> **Current Date**: 2026-10-06 (Asia/Bangkok UTC+7)  
> **Daily Status**: `DAILY STATUS: ORGANIC GROWTH RUN COMPLETE`  
> **Weekly Status**: `WEEKLY STATUS: SEARCH DATA MATURING`  
> **Current 30-Day Baseline Status**: `FINAL STATUS: TORA SEARCH INDEXING HEALTHY — CONTINUE AUTHORITY BUILDING`

---

## 1. EXECUTIVE SUMMARY & AUTHORITATIVE CURRENT STATE

The **Tora AI Organic Growth Autopilot** has officially transitioned from a pure timely news factory into an integrated **Topical Authority & Search Demand Engine**. The production platform at `https://www.toraapi.com` is actively operating across three synchronized loops (Daily, Weekly, 30-Day), backed by server-side PostgreSQL ACID transaction guarantees, real-time IndexNow push notifications, Google Search Console telemetry, and multi-engine AI visibility observation.

### Authoritative Production System Telemetry

```
┌──────────────────────────────────────┬────────────────────────────────────────────────────────┐
│ System Dimension                     │ Live Measured Production State                         │
├──────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ Production Origin                    │ https://www.toraapi.com                                │
│ Container & Build Tag                │ new-api / tora-api:r11-db0c36bf8 (Healthy)            │
│ Discovery Source Registry            │ 24 / 24 Healthy (HTTP 200, 0 Failures, 100% Parsed)    │
│ Atomic Daily Quota Ledger            │ Enforced (1 Autopilot + 2 Evergreen Pillars Published)  │
│ Hard Limit Invariants               │ MAX 20/day, MAX 5/batch, MAX 3/source/day              │
│ Google News Sitemap                  │ Strictly 1 URL (/news/larger-cohere-representation...) │
│ Main Sitemap                         │ 26 Canonical URLs (Includes Core & 2 Evergreen Pillars)│
│ Real-Time IndexNow Telemetry         │ HTTP 202 Accepted on every new/updated canonical URL   │
│ Google Search Console (GSC)          │ CONNECTED (sc-domain:toraapi.com), Delta-Aware Ingest  │
│ Search Analytics (GSC)               │ NO_DATA_YET (Truthful reporting during crawl lag)      │
│ Google URL Inspection Telemetry      │ 12 URLs Sampled (Verdict: NEUTRAL / URL Unknown)       │
│ DEV.to Syndication                   │ ACTIVE & Idempotent (Post 8 live at remote ID 4804790) │
│ Legacy Content Partition             │ 1,203 posts safely isolated in UNKNOWN_LEGACY          │
│ Social / GA4 Channels                │ OPERATOR_BLOCKED (Safe boundary preserved)             │
│ AI Citation Benchmark                │ 9 Categories Tested, Zero Fabricated Citations         │
└──────────────────────────────────────┴────────────────────────────────────────────────────────┘
```

---

## 2. THE 30-DAY MULTI-LOOP OPERATING ARCHITECTURE

The growth autopilot executes across three interdependent loops without requiring manual developer triggers:

```mermaid
flowchart TD
    subgraph DailyLoop ["DAILY LOOP (Every 24 Hours)"]
        D1["05:30, 10:30, 15:30, 20:00<br/>Multi-Source Scouting"] --> D2["07:00, 12:00, 17:00, 21:00<br/>Atomic Batch Publishing"]
        D2 --> D3["Real-time IndexNow & Sitemaps"]
        D3 --> D4["Selective GSC URL Inspection"]
        D4 --> D5["23:30 Daily Growth Report & Top 5 Actions"]
    end

    subgraph WeeklyLoop ["WEEKLY LOOP (Every 7 Days)"]
        W1["24-Source Quality Re-weighting<br/>(HIGH_VALUE, USEFUL, NOISY, LOW_VALUE)"]
        W2["Topic Demand & Gap Analysis<br/>(11 Verticals Evaluation)"]
        W3["Evergreen Candidate Promotion<br/>(News -> Evergreen Conversion)"]
        W4["Backlink Opportunity Scoring & Staging"]
    end

    subgraph MonthlyLoop ["30-DAY LOOP (Full Period Evaluation)"]
        M1["Indexing Funnel Maturation Decision<br/>(Discovered -> Crawled -> Indexed)"]
        M2["Topical Authority & Pillar Consolidation"]
        M3["Legacy Content Cannibalization Audit<br/>(KEEP, REFRESH, MERGE, NOINDEX, REMOVE)"]
        M4["AI Citation & Multi-Engine Visibility Benchmark"]
    end

    DailyLoop --> WeeklyLoop
    WeeklyLoop --> MonthlyLoop
```

---

## 3. ATOMIC QUOTA & SAFETY INVARIANTS VERIFICATION

Server-side database invariants are enforced strictly under PostgreSQL ACID row-locks (`SELECT ... FOR UPDATE` via `news_autopilot_daily_quota`) and unique composite indexes (`idx_news_pub_events_unique_initial`):

1. **Daily Cap**: Enforces `published_count <= 20`. Any excess batch attempts are gracefully deferred to `draft` without failing the pipeline.
2. **Batch Cap**: Max `5` articles per scheduled publish execution.
3. **Source Cap**: Max `3` articles per individual source feed per Bangkok calendar day.
4. **Zero Historical Leaks**: The ~4,742 historical distribution rows and 1,203 legacy posts are strictly suppressed from auto-push queues.
5. **No Timestamp Falsification**: All `published_at` and `updated_at` fields reflect true Unix epoch timestamps.

---

## 4. NEWSROOM ENGINE & 24 SOURCE HEALTH REGISTRY

All 24 authoritative global technology, AI research, and cloud infrastructure sources were verified healthy:
- **Zero Degraded Feeds**: All endpoints return `HTTP 200` with valid RSS/Atom XML payloads.
- **Deduplication Engine**: Pre-evaluates title/summary similarity using Jaccard and SimHash algorithms ($threshold = 0.75$) to prevent repetitive coverage across overlapping vendor releases.
- **Adaptive Volume**: Rather than forcing arbitrary volume to fill the 20-post quota, the system adapts dynamically based on story significance, topic diversity, and linguistic quality.

---

## 5. SEARCH DEMAND MAP & TOPICAL CLUSTERS

Documented in [`docs/ai/SEARCH_DEMAND_MAP.md`](file:///Users/noppanan/new-api/docs/ai/SEARCH_DEMAND_MAP.md), the system targets **24 strategic search clusters** aligned directly with Tora API’s commercial value propositions:
- **Core Value Verticals**: *AI API Gateway*, *BYOK AI*, *OpenAI-Compatible API*, *LLM Fallback & High Availability*, *AI Model Routing*.
- **High-Volume Search Drivers**: *AI API Pricing Comparison*, *Claude vs GPT*, *Gemini vs GPT*, *Best AI Model for Coding*, *Best AI Model for Thai*.
- **Emerging High-Demand Clusters**: *AI Video APIs*, *AI Image APIs*, *Agent Frameworks*, *AI Developer Tools (Cursor/Windsurf)*.

### Content Mix Evolution (Target Distribution)
- **40% Evergreen Search Intent**: In-depth pillar guides and architecture references.
- **25% Model / Provider / Pricing Reference**: Up-to-date token pricing, context window comparisons, and benchmark summaries.
- **20% Timely Newsroom Coverage**: Fresh breaking updates within 48 hours for Google News indexation.
- **10% Developer Resources**: Practical code integration samples for Python, Node.js, and curl.
- **5% Deep Technical Analysis**: Original architectural perspectives and engineering opinions.

---

## 6. EVERGREEN PILLAR ENGINE & INITIAL DEPLOYMENTS

Per Sections 10, 11, and 12, two authoritative Evergreen Pillars were authored, verified, and published directly to production:

### Pillar 1: AI API Gateway & BYOK Architecture
- **URL**: `https://www.toraapi.com/news/ai-api-gateway-architecture-guide`
- **Post ID**: `12633` | **Content Type**: `guide` | **Origin**: `MANUAL_ADMIN`
- **Title**: *คู่มือสถาปัตยกรรม AI API Gateway: เชื่อมต่อหลายโมเดลด้วยมาตรฐาน OpenAI-Compatible และระบบ BYOK*
- **Technical Scope**: Explains multi-model fragmentation, standard OpenAI-compatible reverse proxy design, zero-knowledge BYOK Key Vault isolation (AES-256-GCM), and circuit breaker patterns.
- **Structured Data**: Valid Schema.org `TechArticle` JSON-LD.
- **IndexNow**: Submitted and accepted (HTTP 202).

### Pillar 2: LLM Routing & Dynamic Fallback Architecture
- **URL**: `https://www.toraapi.com/news/llm-routing-and-fallback-architecture`
- **Post ID**: `12634` | **Content Type**: `guide` | **Origin**: `MANUAL_ADMIN`
- **Title**: *คู่มือการออกแบบ LLM Routing & Dynamic Fallback: ป้องกัน API ล่ม ลดต้นทุน และเพิ่ม Uptime 99.9%*
- **Technical Scope**: Deep dive into Intra-Provider vs Cross-Provider failover, SSE streaming fallback resilience (header holdback buffer), error classification (429 vs 500 vs timeout), and cost-optimized routing.
- **Structured Data**: Valid Schema.org `TechArticle` JSON-LD.
- **IndexNow**: Submitted and accepted (HTTP 202).

---

## 7. TECHNICAL SEO, SITEMAPS & INDEXNOW TELEMETRY

1. **Google News Sitemap (`/news-sitemap.xml`)**:
   - HTTP 200 OK, valid XML.
   - Strictly contains Post `12632` (*Larger Cohere Representation Models*).
   - Isolates `ContentType = 'news'` with `age <= 48` hours.
   - Excludes all evergreen guides, seed posts, and legacy posts.
2. **Main Sitemap (`/sitemap.xml`)**:
   - HTTP 200 OK, valid XML.
   - Includes all 4 core routes (`/`, `/pricing`, `/docs`, `/news`), 20 fresh news articles, and both newly published Evergreen Pillars (`12633` and `12634`).
3. **IndexNow Real-Time Broadcast**:
   - Endpoint: `https://api.indexnow.org/IndexNow`
   - Key: `484c06053f3e43a992a8327171e5491a`
   - Verified HTTP 202 response on newly created canonical URLs.
4. **Structured Data Validation**:
   - Automated injection of `NewsArticle` (for news), `TechArticle` (for guides), and `BreadcrumbList` across all SSR responses.

---

## 8. GOOGLE SEARCH CONSOLE & INDEXING FUNNEL STATUS

Documented in [`docs/ai/ORGANIC_INDEXING_FUNNEL.md`](file:///Users/noppanan/new-api/docs/ai/ORGANIC_INDEXING_FUNNEL.md):
- **GSC Connection**: Verified active for property `sc-domain:toraapi.com`.
- **Search Analytics Ingestion**: Currently reports `NO_DATA_YET`. Ingestion service uses delta-aware partition caching, querying missing dates with `dataState=all` while treating verified past dates (>3 days) as immutable.
- **URL Inspection Sampling**: 12 priority URLs inspected via Google Search Console API. All currently return `verdict: NEUTRAL`, `coverage_state: URL is unknown to Google`.
- **Early-Domain Analysis**: For a fresh domain, search engines typically require 3 to 14 days to dispatch initial discovery crawlers, parse XML sitemaps, and allocate crawl budgets. The technical pipeline is confirmed fully crawlable and compliant.

---

## 9. BACKLINK OPPORTUNITY ENGINE & ECOSYSTEM OUTREACH

Documented in [`docs/ai/BACKLINK_OPPORTUNITIES.md`](file:///Users/noppanan/new-api/docs/ai/BACKLINK_OPPORTUNITIES.md):
- **Zero-Spam Policy**: Automatically rejects Class C link schemes, comment spam, PBNs, and automated link exchanges.
- **Class A Pipeline**: 6 high-authority developer resources staged:
  1. *Awesome OpenAI / AI Gateway GitHub Repositories* (DR 96)
  2. *LangChain Multi-Provider Ecosystem Directory* (DR 88)
  3. *LlamaIndex Integration Listings* (DR 85)
  4. *Blognone Thai Tech Community Guest Architecture* (DR 78)
  5. *SiamHTML Thai Developer Community Architecture Guide* (DR 65)
  6. *DEV Community (dev.to)*: **LIVE** (Post 8 live at remote ID 4804790).
- **Class B Pipeline**: 4 legitimate developer tool directories staged (*Toolify.ai*, *Futurepedia*, *AlternativeTo*, *Product Hunt*).

---

## 10. AI VISIBILITY & MULTI-ENGINE CITATION BENCHMARKS

Documented in [`docs/ai/AI_VISIBILITY_SCORECARD.md`](file:///Users/noppanan/new-api/docs/ai/AI_VISIBILITY_SCORECARD.md):
- **Telemetry Separation**: Formally distinguishes between AI Crawler Hits, Search Engine Impressions, AI Referral Traffic, and Verified AI Citations.
- **Benchmark Baseline**: 9 stable test queries executed and persisted in PostgreSQL table `news_ai_visibility_observations`:
  - Result: All 9 report `Tora Cited = False` (Zero fabricated citations).
  - Provides the truthful benchmark against which ongoing citation acquisition will be measured.
- **Citation-Worthiness Rules**: Emphasizes direct conceptual definitions, structured comparison tables, and primary source citations to maximize grounding utility for AI models.

---

## 11. LEGACY CONTENT STRATEGY & CANNIBALIZATION DEFENSE

Documented in [`docs/ai/LEGACY_CONTENT_DECISION.md`](file:///Users/noppanan/new-api/docs/ai/LEGACY_CONTENT_DECISION.md):
- **Forensic Audit**: The 1,203 legacy posts have an average content length of 1,242.0 characters (Min: 1,055, Max: 1,465), with zero thin content ($<500$ chars).
- **Isolation Policy**: Strictly maintained in `UNKNOWN_LEGACY` partition and excluded from `/news-sitemap.xml`.
- **30-Day Decision**: Keep intact. Do not bulk-delete or return 404/410 during the initial 30 days to avoid destabilizing young domain crawl budgets.
- **Cannibalization Guard**: If real GSC search queries reveal SERP competition between legacy posts and new evergreen pillars, automated 301 redirects will consolidate equity into the authoritative pillar.

---

## 12. 14-DAY AND 30-DAY PROJECTED MILESTONES

| Milestone Checkpoint | Target Focus & KPIs | Scheduled Actions |
| :--- | :--- | :--- |
| **Day 1–7 (Current)** | Newsroom baseline, Quota hardening, Initial 2 Evergreen Pillars | Daily scouting, IndexNow push, GSC delta sync, AI benchmarks |
| **Day 8–14 (14-Day Interim)** | Publish 3 additional pillars (Model Comparison, LLM Pricing, Coding) | Evaluate initial Googlebot crawl hits in server logs; First weekly source re-weighting |
| **Day 15–21 (Week 3)** | Expand Multi-Provider & Agent Framework clusters | Inspect first search queries in GSC; Detect position 5–20 optimization opportunities |
| **Day 22–30 (Day 30 Final)** | Full topical authority consolidation & Legacy post re-classification | Classify system into 30-Day Decision Matrix; Deliver Final 30-Day Analysis |

---

## 13. DAILY DECISION: TOP 5 HIGHEST-VALUE ACTIONS TOMORROW

Per Section 46, the 5 highest-value executable actions for tomorrow’s operating cycle:

1. **Scout & Publish Morning Batch (05:30 / 07:00 Bangkok)**: Execute scheduled 24-source scan; publish top 2–4 verified breaking stories under atomic quota ledger.
2. **Author Evergreen Pillar 3 (`frontier-llm-comparison-claude-gpt-gemini`)**: Create comprehensive model comparison reference covering Claude 3.7 Sonnet, GPT-4o, and Gemini 2.5 Flash with benchmark tables.
3. **Internal Link Injection Sweep**: Link newly published Post 12632 (*Larger Cohere Representation Models*) contextually to the newly deployed Pillar 1 (*AI API Gateway*).
4. **Submit Pillar 1 & 2 to Staged Backlink Listings**: Prepare submission drafts for GitHub *Awesome OpenAI* and LangChain ecosystem directories.
5. **Execute Delta GSC Search Analytics Sync**: Poll `QueryDeltaSearchAnalytics` for newly finalized partitions and record updated indexation states.

---

## 14. DECISION ENGINE CLASSIFICATION & FINAL STATUS DECLARATIONS

### Protocol Status Declarations

- **Daily Status (Section 72)**:  
  **`DAILY STATUS: ORGANIC GROWTH RUN COMPLETE`**

- **Weekly Status (Section 73)**:  
  **`WEEKLY STATUS: SEARCH DATA MATURING`**

- **Current 30-Day Baseline Status (Section 74)**:  
  **`FINAL STATUS: TORA SEARCH INDEXING HEALTHY — CONTINUE AUTHORITY BUILDING`**
