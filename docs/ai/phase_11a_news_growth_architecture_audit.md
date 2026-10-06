# Phase 11A — Autonomous News & Organic Growth Engine Architecture Audit

## Mission & Executive Summary

Tora AI requires a scalable, autonomous organic growth and content discovery engine to acquire technical users, developers, and AI engineers across Thailand and Southeast Asia. 

Phase 11A audits the current system architecture, examines existing open-source and MCP ecosystems, establishes security and copyright invariants, and designs the end-to-end autonomous pipeline:

$$\text{Discover} \to \text{Deduplicate} \to \text{Research} \to \text{Fact-Check} \to \text{Editorial (Thai)} \to \text{SEO} \to \text{Publish} \to \text{Index} \to \text{Distribute} \to \text{Measure} \to \text{Improve}$$

---

## 1. Open Source Ecosystem & Reuse/Integrate Matrix

We evaluated available open-source tools, MCP servers, and distribution platforms:

| Tool / Technology | License | Architectural Role | Strategy / Decision |
| :--- | :--- | :--- | :--- |
| **LukeRenton / Suganthan GSC MCP** | MIT | Search Console search analytics & URL inspection | **Integrate via MCP client**: Query Search Console API for impressions, clicks, CTR, position, and indexing state without hardcoded credentials. |
| **Firecrawl MCP** | MIT (Client) / AGPL (Core) | Web extraction, deep clean markdown scraping | **External API / MCP tool**: Use for targeted technical doc retrieval when RSS feeds only contain summaries. Do not copy AGPL server code into Tora core. |
| **Postiz** | AGPL-3.0 | Multi-platform social media scheduling | **External Service / Webhook only**: Because Postiz is AGPL-3.0, do not vendor or copy source code. Tora's Distribution Engine formats payload derivatives and dispatches via standard webhook or official APIs. |
| **Standard RSS / Atom 1.0** | Open Spec | News discovery from primary sources (OpenAI, Anthropic, Google, DeepSeek) | **Native Go Build**: High-speed, zero-dependency XML parser with strict timeout and byte limit guards. |
| **Google News XML Schema** | Open Spec | Real-time news indexing by Google News crawlers | **Native Go Build**: Serves `/news-sitemap.xml` with `<news:news>` namespace. |
| **OpenGraph & Schema.org JSON-LD** | W3C Standard | Search rich snippets & social previews (Facebook, LinkedIn, X, LINE) | **Native Go Prerender**: Embeds `NewsArticle` and `BreadcrumbList` directly into crawlable HTML. |

---

## 2. Current Tora Source Architecture Audit

### 2.1 Web Router & Public Page Delivery
- Current `router/web-router.go` acts as an SPA fallback for `web/dist` (Dashboard console).
- **SEO Invariant**: Social media preview bots (Facebook, Twitter, LinkedIn, LINE) and search engine crawlers require **server-rendered crawlable HTML** with `<meta property="og:title">`, `<meta property="og:image">`, `<link rel="canonical">`, and `<script type="application/ld+json">`.
- **Solution**: Dedicated public Gin web routes:
  - `GET /news`: News catalog index with semantic HTML, pagination, and category filtering.
  - `GET /news/:slug`: Individual article with full semantic markup, schema, responsive CSS, and internal links.
  - `GET /news/:slug/og.svg`: Dynamic SVG brand creative card for social previews.
  - `GET /sitemap.xml` & `GET /news-sitemap.xml`: XML sitemaps for search engines.
  - `GET /robots.txt`: Search bot directives.

### 2.2 Background Task & Scheduler Infrastructure
- `service/system_task.go` already provides a distributed-lease, multi-node safe task engine (`ScheduledSystemTaskHandler`).
- The News Scout, Deduplication, and Distribution jobs plug directly into this background task framework, guaranteeing that recurring autonomous discovery continues smoothly without requiring manual triggering.

---

## 3. Data Model Architecture (CMS Core)

```
┌─────────────────────────────────────────────────────────────┐
│                       NewsSource                            │
│  - Name, FeedUrl, SiteUrl, TrustTier, PollingPolicy         │
└──────────────────────────────┬──────────────────────────────┘
                               │ fetches
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                      StoryCluster                           │
│  - Title, Summary, PrimarySourceId, RelevanceScores         │
└──────────────────────────────┬──────────────────────────────┘
                               │ generates
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                        NewsPost                             │
│  - Slug, Title, Summary, Markdown, HTML, SeoMetadata        │
│  - ContentRisk (LOW/MED/HIGH), FactCheckStatus              │
└──────────────┬──────────────────────────────┬───────────────┘
               │ creates                      │ tracks
               ▼                              ▼
┌──────────────────────────────┐ ┌────────────────────────────┐
│       NewsDistribution       │ │     NewsAnalyticEvent      │
│  - Facebook, LinkedIn, X     │ │  - PageView, UTM, Signup,  │
│  - Status, ContentPayload    │ │    Quota, Subscription     │
└──────────────────────────────┘ └────────────────────────────┘
```

---

## 4. Editorial Integrity & Anti-Clickbait Policy

1. **No Direct Copying**: Every article is an original technical analysis written in Thai, grounded in facts extracted from primary sources.
2. **Mandatory Fact-Checking**: Primary sources are cited with direct links. Unverified claims or rumors are rejected.
3. **Value-Add for Developers**: Every article includes:
   - What changed technically (API signatures, context window, pricing, latency).
   - How developers in Thailand can access it (model aliases, Tora pricing, SDK examples).
   - Comparison with predecessor models.
4. **Risk Classification**:
   - `LOW`: Routine model, SDK, or API updates -> Automated review and publish.
   - `MEDIUM`: Pricing and benchmark analyses -> Rigorous cross-source comparison required.
   - `HIGH`: Security vulnerabilities, regulatory changes, or unverified claims -> Strictly blocked from auto-publishing without explicit operator review.

---

## 5. Architectural Verdict

**FINAL STATUS: READY TO IMPLEMENT PUBLIC NEWS CMS & SCOUT ENGINE**

All architectural, legal, security, and licensing constraints have been satisfied. We now proceed immediately to implement Phase 11B (Public News CMS) and Phase 11C (Source Registry & Scout Engine).
