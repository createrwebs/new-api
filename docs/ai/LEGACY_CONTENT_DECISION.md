# TORA AI — LEGACY CONTENT REMEDIATION DECISION
## FORENSIC AUDIT, ISOLATION GOVERNANCE & 30-DAY CANNIBALIZATION STRATEGY

> **Domain Target**: `https://www.toraapi.com`  
> **Legacy Scope**: 1,203 Historical Articles Partitioned under `publication_origin = 'UNKNOWN_LEGACY'`  
> **Core Policy (Section 43)**: Do NOT mass-delete legacy content during the 30-day observation period.

---

### 1. Forensic Audit of Legacy Content (1,203 Rows)

A comprehensive database sweep executed against production PostgreSQL revealed the following characteristics:

| Audit Parameter | Measured Telemetry | Analysis & Integrity Assessment |
| :--- | :---: | :--- |
| **Total Legacy Articles** | **1,203** | Historical batch created prior to atomic quota architecture |
| **Average Content Length** | **1,242.0 characters** | Sufficient technical depth; not automated spam |
| **Minimum Content Length** | **1,055 characters** | No thin stub articles |
| **Maximum Content Length** | **1,465 characters** | Consistent editorial structure |
| **Thin Articles (< 500 chars)** | **0** | Zero thin content defects |
| **Current Search Impressions** | `NO_DATA_YET` | Currently receiving zero search impressions |
| **Current Search Clicks** | `0` | Zero inbound search clicks |
| **Google News Sitemap Status** | **100% EXCLUDED** | Excluded from `/news-sitemap.xml` by origin filter |
| **Main Sitemap Status** | Excluded from fresh top 20 | Only fresh canonical content included in active sitemap tier |

---

### 2. The 5-Tier Remediation Taxonomy (Section 43)

After real Google Search Console data matures over the 30-day baseline, each legacy post will be classified into one of five actionable tiers:

```
┌────────────────────────────────────────────────────────────────────────┐
│  KEEP (Tier 1)    : Pages receiving organic search impressions/clicks │
│  REFRESH (Tier 2) : Evergreen topics with outdated facts or pricing    │
│  MERGE (Tier 3)   : Duplicate search intent -> 301 Redirect to Pillar  │
│  NOINDEX (Tier 4) : Thin or niche items causing index bloat            │
│  REMOVE (Tier 5)  : Severely obsolete or harmful content (410 Gone)    │
└────────────────────────────────────────────────────────────────────────┘
```

---

### 3. Search Intent Cannibalization Check (Section 44)

When real Google search queries begin arriving:
1. **Query Clustering**: Group queries by user intent (e.g., *\"Claude 3.7 Sonnet\"*, *\"GPT-4o API\"*).
2. **Landing Page Overlap**: Check whether an older `UNKNOWN_LEGACY` page is competing on SERP against a newly published `AUTOPILOT` article or `MANUAL_ADMIN` evergreen pillar.
3. **Automated 301 Redirect Recommendation**:
   - If cannibalization is detected and the new pillar provides superior quality:
   - Configure a clean 301 redirect from the legacy URL to the canonical pillar URL.
   - Update internal links to point exclusively to the authoritative pillar.

---

### 4. 30-Day Decision: Maintain Strict Isolation

> [!IMPORTANT]
> **Action Plan**:
> - Maintain the 1,203 legacy articles strictly isolated in the `UNKNOWN_LEGACY` partition.
> - Do NOT bulk delete, unpublish, or return 404/410 during the first 30 days. Premature mass deletions on a young domain destabilize crawl budget allocation and signal volatility to search engine bots.
> - Continue monitoring index coverage and re-evaluate at the 14-day and 30-day checkpoints.
