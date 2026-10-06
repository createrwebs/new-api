# Tora AI — Current State

> This file must be updated after every successful autonomous work cycle.

## Overall Status

**AUTONOMOUS WORK EXHAUSTED — WAITING FOR OPERATOR**

All executable engineering, integration, testing, build, security verification, and release gate tasks across mobile and backend are 100% complete. The project is fully operational in mock/isolated environments, release APK and AAB build artifacts are verified, and the codebase is completely prepped for live store deployment. Progress is now paused exclusively on external operator actions in Apple App Store Connect and Google Play Console (Roadmap item R5).

## Completed Major Work

### Backend security / financial integrity

- BYOK encryption / isolation / SSRF defense / rate & concurrency controls
- auth and token security remediation
- Stripe/payment refund and idempotency fixes
- cross-domain release-gate audit

### Mobile foundation

- secure auth foundation
- per-installation relay token
- environment-scoped secure credentials
- account-partitioned local conversations
- managed chat / SSE / cancellation
- server-managed BYOK
- subscription/quota entitlement UX

### Native store billing

- Apple and Google server verification backends
- store product mapping and settlement models
- webhook handling
- Flutter native purchase state machine
- restore / offline verification recovery
- account/environment callback isolation
- duplicate/replay handling
- account-binding hardening using store account tokens

### Full-Stack E2E & Release Engineering (R6 & R7)

- R6A: Isolated, deterministic one-command local E2E environment (`./scripts/tora-e2e-up`, `./scripts/tora-e2e-test`, `./scripts/tora-e2e-down`).
- R6B: Full-stack managed lifecycle verified (registration -> login -> token reveal -> `/v1/models` -> `/v1/chat/completions` SSE stream & cancel -> quota settlement -> cross-account attack isolation).
- R6C: Full-stack BYOK lifecycle verified (OpenRouter masked provider -> key rotation -> deletion -> zero plaintext leak).
- R6D: Full-stack Top-Up lifecycle verified (voucher card redemption -> wallet balance update -> replay rejection -> Stripe Pay endpoint validation).
- R6E: Full mobile endpoint contract sweep (22 endpoints verified).
- R7A: Release builds verified (Android APK 71.8MB, Android AAB 56.3MB, iOS Runner.app 36.2MB with R8 tree shaking).
- R7B: Unified CI release gate runner (`./scripts/verify-release-gates`).
- R7C: Zero-secret repository audit verified (zero hardcoded private keys or production secrets).
- R7D: Comprehensive runtime configuration preflight matrix (`docs/ai/RUNTIME_CONFIGURATION_PREFLIGHT.md`).
- R7E: Observability and structured logging audit (`docs/ai/OBSERVABILITY_AUDIT.md`).
- R7F: Store Console setup checklist and runbook (`tora_ai_store_console_setup_checklist.md`).

### First-Class Routes & Ordered Fallback Engine (R10 / Phase 7I & Phase 7I-S)

- First-class `Route` entity with safe referential deletion guards (`ErrRouteReferencedByKeys`) and entitlement boundaries
- API key primary/fallback route chains completely decoupled from commercial subscription groups
- Authoritative error classifier (`ClassifyRouteError`) distinguishing fallback-eligible vs. terminal errors
- Denial-of-Wallet defense: ambiguous network errors (`connection reset by peer`, read timeouts, `unexpected EOF` after transmission) strictly terminal (`AllowFallback: false`)
- Transparent streaming fallback guard with `CommitDetectingWriter` preventing post-commitment failovers
- Conservative pre-consumption quota reservation filtered exclusively by eligible routes, preventing reservation inflation
- Winning-route exact settlement formula $\lceil \text{BaseQuota} \times \text{ModelRatio} \times \text{UserGroupRatio} \times M_{\text{winning}} \rceil$ with instant difference refund
- In-flight request immutability via `RouteSnapshot` protecting execution against concurrent admin edits
- Distributed Redis channel cooldown (`tora:cooldown:channel:%d`) with TTL auto-expiration
- BYOK terminal failure isolation and per-channel 15-second cooldown backoffs
- 100% backward compatibility for legacy non-routed tokens
- Phase 7I-S Staging Gate passed: 10/10 adversarial/staging tests, 100% backend unit/regression suite, zero data races, `go vet` clean, web typecheck clean, 117/117 mobile tests pass, and debug APK assembled
- Status: **FIRST-CLASS ROUTES STAGING VERIFIED — PRODUCTION PILOT READY** (strictly opt-in; no existing production customer tokens migrated)

### Autonomous News & Closed-Loop Growth Autopilot (R11, R11-M & R11-LIVE)

- Real multi-content type classification: `news`, `guide`, `analysis`, `changelog`
- Fixed historical launchpack timestamps (Sep/Oct 2026; no re-dating on reboot or sitemap pollution)
- Strict Google News sitemap freshness boundary (48h cutoff filter with 47h/48h/49h boundary validation; evergreen guides strictly isolated)
- Permanent `IsSeed` exclusion from Google News sitemap (`/news-sitemap.xml`) preventing bootstrap seed content from ever masquerading as breaking news, while preserving discovery in standard `/sitemap.xml`
- 1200x630 raster PNG dynamic social card generator (`/news/:slug/og.png`) with brand gradient, logo badge, category pill, 3x scaled typography, and OpenGraph metadata
- Google Search Console connector supporting both `GSC_CREDENTIALS_FILE` and `GSC_CREDENTIALS_JSON` with zero credential logging and minimum read-only scope (`webmasters.readonly`)
- 8-class SEO opportunity classifier + autonomous remediation protected by strict 7-day cooldown lock
- LinkedIn API Remediation: Migrated from deprecated `/v2/ugcPosts` to official LinkedIn REST Posts API (`https://api.linkedin.com/rest/posts`) with `LinkedIn-Version: 202401`, `X-Restli-Protocol-Version: 2.0.0`, `w_organization_social` scope, article attachments, and `x-restli-id` response parsing
- Distribution Idempotency Redesign: Completely decoupled content revisions and `UpdatedAt` from distribution keys. Distribution intent uniquely keyed by `(PostId, Platform, DistributionVersion)`. Routine content/SEO edits strictly do NOT trigger duplicate social posts
- Multi-Channel Canary Pipeline: DEV.to -> Facebook Page -> LinkedIn Organization Page with atomic database claims (`status = "processing"`) preventing worker race conditions, and at-least-once reconciliation recovering after worker crash/timeout
- Privacy-safe conversion funnel tracking (`organic_landing`, `social_landing`, `signup`, `api_key_created`, `first_successful_inference`, `topup`, `subscription`) using keyed HMAC-SHA256 (`ATTRIBUTION_HASH_KEY`) with zero raw IPs stored and retention minimization
- Automated Daily Growth Review aggregator (`NewsDailyGrowthReview`) and pipeline health reporting
- 100% test pass across model, service, controller suites, and unified CI release gates (`verify-release-gates`)

### PostgreSQL Staging-Clone Migration & News Runtime Proof (R11-PG)

- Real GORM `AutoMigrate` verified against PostgreSQL 15.19 staging-clone `tora_local_webtest` (56 tables total, all 9 news/story tables successfully created with bigserial PKs, unique constraints, and B-tree indexes)
- Proven Zero Mutation on Production (`new-api`: 40 tables, 0 news tables) and Live Staging (`tora_staging`: 39 tables, 0 news tables)
- Mandatory Safety Guard (`model/database_safety.go` with `CheckDatabaseSafety()`) active in `model.InitDB()` preventing accidental connections to forbidden databases
- Real runtime news smoke tests on PostgreSQL (`http://localhost:3005`) returning 200 for `/news`, all 6 launchpack articles, `/sitemap.xml`, `/news-sitemap.xml` (freshness boundary verified), `/robots.txt`, and 404 for nonexistent slugs
- Direct PostgreSQL write/read/delete data path validated
- Full Admin News API write lifecycle validated (create draft -> DB verified -> draft privacy 404 verified -> update -> publish -> public SSR feed verified -> public SSR article verified -> unpublish -> delete -> DB verified -> clean)
- Autopilot pipeline validated in PostgreSQL (Scout ingestion -> Cluster deduplication & corroboration -> Thai editorial draft -> Multi-channel social distributions queued -> Publish -> Public SSR -> Cleanup)
- TrueType native Thai headline rendering verified on 1200x630 OG PNG cards (`service/news_creative.go`) using Ayuthaya / Waree fonts without English transliteration fallback
- Minimal React Admin News UI implemented in `web/src/features/news/` and routed at `web/src/routes/_authenticated/news-admin/` with `ROLE.ADMIN` guard, sidebar entry, and 100% clean typecheck and build
- 100% CI release gate pass verified via `./scripts/verify-release-gates` (Go vet, unit tests, 5 full-stack E2E suites on port 3006, web typecheck, flutter analyze, flutter test, Android release APK, Android release AAB, zero secrets)

### Live Staging Migration & Deployment Canary (R11-STAGING)

- Database Engine Fact Reconciliation: Confirmed authoritative engine across all databases is `PostgreSQL 15.19 (Debian 15.19-1.pgdg13+2) on aarch64-unknown-linux-gnu`.
- Permissions & Isolation Invariant: Production role `root` retains full access on `new-api`. Test role `tora_local_dev` connection revoked on `new-api` (`FATAL: permission denied`) and public schema mutation revoked on `tora_staging`. Production `new-api` (40 tables, 0 news tables) remains 100% untouched.
- Pre-Migration Backup & Restore Drill: Timestamped logical backup created at `/home/ubuntu/backups/tora_staging_pre_r11_20261006_024507.dump` (160 KB). Verified via `pg_restore --list` (482 TOC entries) and full restoration into temporary database `tora_restore_test` matching exactly 39 tables, 1 user, 1 token.
- Schema Diff & Additive Safety: Verified schema changes are 100% ADDITIVE (17 new tables, 3 new nullable columns on `tokens`, 0 destructive alterations). Old staging binary coexisted without downtime or errors.
- Live Staging Migration: Executed GORM AutoMigrate against `tora_staging` via SSH tunnel. Total tables increased from 39 to 56 (all 9 news/growth tables created with indexes). Default sources and launchpack articles seeded.
- Application Deployment: Built and deployed `src-tora-api-staging:latest` container from `feat/formobile` with `fonts-thai-tlwg-ttf` installed. Booted in 3,913 ms behind Caddy on `https://staging-api.toraapi.com`. Tagged previous image as `rollback-b4a92ebf4`.
- Live Public HTTP Verification: Tested on `https://staging-api.toraapi.com` with HTTP 200 on `/news`, `/news/:slug`, `/sitemap.xml`, `/news-sitemap.xml` (freshness boundary verified), `/robots.txt`, `/news/:slug/og.png` (23,368 bytes), and 404 on nonexistent slugs.
- React Admin News UI: Verified `/news-admin` serves React SPA dashboard with IBM Plex Sans Thai typography. Admin API endpoints return 401 when unauthorized and 200 when authenticated with Admin credentials.
- Staging Write Lifecycle: Verified complete end-to-end write lifecycle on `tora_staging` (draft -> draft privacy 404 -> update -> publish -> public SSR index/post/OG -> unpublish -> delete -> clean).
- Autopilot Canary: Verified controlled scout ingestion -> Jaccard clustering (`StoryCluster` ID 1) -> Thai editorial draft generation (`news_posts`) -> 4 distribution records queued (`news_distributions`) -> clean teardown. Real social distribution remained strictly off.
- Native Thai OG PNG: 1200x630 RGBA PNG card generation verified using native Thai TrueType font `/usr/share/fonts/truetype/tlwg/Waree.ttf`.
- Observation Window: Monitored staging container: 0 panics, 0 database errors, 0 background loops, 0.00% CPU, 33.28 MiB RAM usage (1.81% of host limit).
- 100% Release Gates: Master runner `./scripts/verify-release-gates` passed all 10 gates.

### Same-Origin Production News Release Gate (R11-PROD-PREFLIGHT)

- Same-Origin Domain Invariant: Unified production public web and API origins under `https://www.toraapi.com`. Staging API remains `https://staging-api.toraapi.com`.
- Stale Domain Eradication: All obsolete references to `tora.ai`, `api.tora.ai`, and `staging-api.tora.ai` completely audited and updated across `common/canonical.go`, `model/news_seed.go`, `service/news_creative.go`, `service/news_scout.go`, `service/news_editorial.go`, `service/news_seo.go`, `controller/news.go`, and test suites.
- GSC Property Architecture: Primary property defined as domain property `sc-domain:toraapi.com` (DNS TXT verified), with URL-prefix property `https://www.toraapi.com/` fully supported via `GSC_SITE_URL`.
- Site-Wide Ownership & Robots.txt: Enforced clean single-origin routing for React SPA (`/`, `/pricing`, `/docs`), Go SSR News (`/news`, `/news/:slug`), SEO assets (`/sitemap.xml`, `/news-sitemap.xml`, `/robots.txt`), and News Admin SPA (`/news-admin`). `/robots.txt` explicitly disallows `/news-admin`, `/api/`, `/v1/`, `/admin/` while allowing public paths.
- Sitemap Enhancements: Added `/docs` to standard XML sitemap core pages alongside `/`, `/news`, and `/pricing`.
- HTTP Crawler Preflight Compatibility: Registered `HEAD` method support for all public news and SEO endpoints, verified returning `HTTP/2 200` without 404 fallthrough.
- Corrected PostgreSQL Rollback Runbook: Fixed implicit transaction block failure (`ERROR: DROP DATABASE cannot run inside a transaction block`) by specifying isolated `dropdb` and `createdb` commands and ACL preservation.
- Read-Only Production Database Preflight: Verified production database `new-api` on `51.20.174.90` (PostgreSQL 15.19, 40 tables, 0 news tables, 1 user). Confirmed migration is 100% additive (16 new tables, 3 new nullable columns, 0 destructive alterations). Zero mutations applied to production.
- Live Staging Synchronization: Rebuilt and deployed `tora-api-staging` container with updated domain configuration. Verified live `HEAD` HTTP/2 200, robots.txt, and canonical links to `https://www.toraapi.com`.
- Controlled Production Deployment Runbook: Authored complete 12-step execution runbook and comprehensive URL test matrix.
- Release Gates: 10/10 local unit and canary tests pass cleanly.

### Prime Directive: External Blocker != Session Stop

An external/operator blocker is NOT a reason to end the autonomous engineering session.
When an item is blocked by operator-dependent external access (such as Apple StoreKit, Google Play Console, or social platform OAuth tokens):
1. Mark the item `OPERATOR_BLOCKED`.
2. Record the exact unblock requirements in `CURRENT_STATE.md` and `ROADMAP.md`.
3. Immediately select and advance the next `EXECUTABLE_NOW` release-related task.
4. Continue until all safe executable engineering work is exhausted.

## Growth Engine Independent Status Dimensions

| Dimension | Truthful Status | Verified Capabilities / Unblock Requirements |
| :--- | :--- | :--- |
| `NEWS_ENGINEERING` | `PASS` | Semantic HTML, Schema.org mapping, 1200x630 PNG card, deduplication clustering, clean `go vet`, 100% tests pass. |
| `NEWS_AUTOPUBLISH` | `READY` | Background scout runner, multi-feed polling, automatic editorial drafting and publishing. |
| `GOOGLE_INDEXING` | `GOOGLE_INDEXING_TECHNICALLY_READY` | `/sitemap.xml` + `/news-sitemap.xml` strictly filtering 48h freshness and excluding `IsSeed`; live indexing requests require GSC Operator authorization. |
| `SEARCH_CONSOLE` | `ACTIVE (LIVE PROVEN ON sc-domain:toraapi.com)` | Full OAuth2 JWT exchange verified, `siteFullUser` permission verified on `sc-domain:toraapi.com`, Search Analytics query 200 OK, URL Inspection 200 OK. Backend Go client deployed on EC2 (`tora-api:r11-290404938`). |
| `SEO_CLOSED_LOOP` | `SEO_ENGINEERING_VERIFIED` | 8-class opportunity classifier, striking-distance heuristics, autonomous remediation, 7-day cooldown safety lock verified. |
| `FACEBOOK_DISTRIBUTION` | `DISTRIBUTION_READY (OPERATOR_BLOCKED)` | Adapter complete with Graph API v26.0; awaiting `FACEBOOK_PAGE_ACCESS_TOKEN` & `FACEBOOK_PAGE_ID`. |
| `LINKEDIN_DISTRIBUTION` | `DISTRIBUTION_READY (OPERATOR_BLOCKED)` | Official REST Posts API (`/rest/posts`) connector verified via contract test with `LinkedIn-Version: 202609`; awaiting `LINKEDIN_ACCESS_TOKEN` & `LINKEDIN_ORG_ID`. |
| `DEV_DISTRIBUTION` | `ACTIVE (LIVE PROVEN IN PRODUCTION)` | Real DEV.to article published (ID 4804790, https://dev.to/createrwebs/building-advertising-for-the-way-people-use-ai-5c8) via live Tora AI backend with canonical URL back to https://www.toraapi.com. |
| `CONVERSION_ATTRIBUTION`| `VERIFIED` | UTM builder, keyed HMAC-SHA256 (`ATTRIBUTION_HASH_KEY`) visitor pseudonyms with zero raw IPs stored, 7-stage funnel logging, and `NewsDailyGrowthReview`. |

## Roadmap Status Classification

| Item | Status | Notes / Blockers |
| :--- | :--- | :--- |
| **R1 Project Brain Bootstrap** | `COMPLETED` | AGENTS.md and docs/ai/* installed and active. |
| **R2 New-API Capability Audit** | `COMPLETED` | 136 OpenAPI paths / 263 Go routes classified. |
| **R3 Stripe / PromptPay Reuse Audit** | `COMPLETED` | Existing Stripe stack verified for reuse; PromptPay verified for THB one-time topups. |
| **R4 Mobile Top-Up & Hosted Checkout** | `COMPLETED` | Voucher card redemption sheet and Stripe checkout options sheet verified. |
| **R5 Store Console Setup** | `OPERATOR_BLOCKED` | Requires operator action in Apple App Store Connect & Google Play Console. |
| **R6 Full-Stack Managed + BYOK E2E** | `COMPLETED` | 4/4 suites passed: Managed lifecycle, BYOK isolation, Top-up voucher/Stripe pay, 22-endpoint contract sweep. |
| **R7 Release Engineering** | `COMPLETED` | Release APK/AAB verified, `./scripts/verify-release-gates` passing 100%, zero-secret sweep clean, preflight & observability docs ready. |
| **R8 Real Store Sandbox E2E** | `OPERATOR_BLOCKED` | Requires live sandbox accounts on physical/test devices following R5. |
| **R9 Final Release Candidate Gate** | `BLOCKED_BY_R5_R8` | Blocked until R5 and R8 exit criteria pass. |
| **R10 First-Class Routes & Fallback** | `COMPLETED (STAGING VERIFIED)` | Phase 7I architecture & implementation complete; Phase 7I-S financial safety, streaming, and staging verification complete. |
| **R11 Autonomous News & Growth Engine** | `COMPLETED` | Crawlable public CMS (/news, /news/:slug), Schema.org JSON-LD, XML sitemaps, robots.txt, 6 launchpack articles seeded, multi-channel distribution, background scout runner, 100% test pass. |
| **R11-M Closed-Loop Growth Autopilot** | `COMPLETED` | Production seeds fixed, 48h Google News boundary verified, 1200x630 PNG card generator, GSC opportunity classifier + 7-day cooldown, multi-channel connectors with idempotency, privacy-safe conversion funnel, daily growth review, 100% release gates pass. |
| **R11-LIVE Real Growth Activation & Canary Gate**| `COMPLETED` | LinkedIn Posts API migrated, distribution idempotency decoupled from content edits, seed sitemap exclusion enforced, keyed HMAC attribution active, GSC file credentials supported, 100% release gates passed. |
| **R11-PG PostgreSQL Staging-Clone Migration & Proof** | `COMPLETED (STAGING-CLONE VERIFIED)` | PostgreSQL 15.19 AutoMigrate verified (56 tables, 9 news tables), zero production/staging mutation proven, safety guard active, full runtime smoke tests, write lifecycle, and autopilot validated, TrueType Thai OG cards rendered, React Admin News UI integrated, 100% release gates pass. |
| **R11-STAGING Live Staging Migration & Canary** | `COMPLETED (STAGING VERIFIED)` | PostgreSQL 15.19 Live Staging (tora_staging) migrated (56 tables, 9 news tables), backup verified with test restore, additive-only changes confirmed, tora-api-staging deployed with Thai fonts, public SSR, admin write lifecycle, autopilot canary, and 10/10 release gates passed. |
| **R11-PROD-PREFLIGHT Same-Origin Prod Release Gate** | `COMPLETED (PRODUCTION READY)` | Unified https://www.toraapi.com same-origin architecture verified, stale domain references eradicated, GSC property architecture defined, robots.txt Disallow /news-admin, HEAD support, rollback runbook corrected, read-only new-api DB preflight passed (16 additive tables, 0 destructive changes). |
| **R11-PROD-DEPLOY-SAFETY Production Deployment Hardening** | `COMPLETED` | Docker Compose bridge network preserved, --network host rejected, rollback tag tora-api:rollback-pre-r11-20261006 preserved, fresh backup created. |
| **R11-PROD-DEPLOY Controlled Production Deployment & Canary** | `COMPLETED (R11 PRODUCTION = PASS)` | PostgreSQL 15.19 production database new-api migrated additively from 40 to 56 tables. Immutable image tora-api:r11-b43005f89 deployed behind Caddy on https://www.toraapi.com. All 14 news/core HTTP gates, desktop/mobile real browser canary, admin auth & write lifecycle canary passed. |
| **R11-GROWTH-ACTIVATION Controlled Real-World Growth Channel Canary** | `COMPLETED (DEV.TO & GSC LIVE PROVEN)` | Global kill switch active (NEWS_DISTRIBUTION_ENABLED=true), provider kill switch (DEVTO_ENABLED=true), allowlist surge protection (NEWS_DISTRIBUTION_ALLOWLIST_POST_IDS=8) active. Production container running tora-api:r11-290404938. Live article published on DEV.to (ID 4804790). Google Search Console fully connected and proven on sc-domain:toraapi.com with siteFullUser permissions. FB & LinkedIn remain OPERATOR_BLOCKED awaiting credentials. |

## External / Operator Blockers

Real release readiness still requires:

### Apple (`OPERATOR_BLOCKED`)

- App Store Connect app/product setup
- `com.saascover.tora.pro.monthly`
- sandbox tester
- App Store server credentials (`.p8`, Key ID, Issuer ID)
- Server Notifications V2 configuration
- physical/supported device sandbox purchase
- restore/reconciliation validation

### Google (`OPERATOR_BLOCKED`)

- Play Console app/package setup
- `tora_pro` / `monthly`
- test track / license tester
- service account JSON with Android Publisher access
- Pub/Sub RTDN configuration
- real test purchase
- restore/reconciliation validation

### Growth Channels (`CONTROLLED ACTIVATION & SOCIAL CANARY`)

- Google Search Console (`GSC = CONNECTED / WARMING UP`): Delta-aware ingestion implemented (`service/news_gsc.go`), caching `FINAL` partitions (age > 3 days) immutably and querying missing/partial dates with `dataState=all`. Hourly worker queries delta partitions; selective URL inspection limited to max 2/run.
- Google Analytics 4 (`GA4 = OPERATOR_BLOCKED`): Production web audit revealed no active `gtag.js` or Measurement ID installed (comments only); `GA4_PROPERTY_ID` not configured. Read-only GA4 Data API client built in `service/news_ga4.go` ready for activation upon operator credential provision.
- DEV Community (`DEVTO = ACTIVE`): `DEVTO_API_KEY` active, Post 8 published live (Article ID `4804790`). Enforces idempotent update policy (`DEVTO_UPDATE_POLICY=UPDATE_EXISTING`) preventing duplicate articles. Real stats (views, reactions, comments) collected live.
- Meta Facebook (`FACEBOOK = OPERATOR_BLOCKED`): Graph API v26.0 connector with Page ID and permissions verification, Thai text formatting, and pre-flight idempotency checks. Awaiting `FACEBOOK_PAGE_ACCESS_TOKEN` & `FACEBOOK_PAGE_ID`.
- LinkedIn (`LINKEDIN = OPERATOR_BLOCKED`): REST API 202609 connector with Organization ID verification and pre-flight idempotency checks. Awaiting `LINKEDIN_ACCESS_TOKEN` & `LINKEDIN_ORG_ID`.
- Backlog Safety Invariant: Historical backlog (~4,742 distributions across platforms) strictly suppressed. Only allowlisted canary posts (`NEWS_DISTRIBUTION_ALLOWLIST_POST_IDS=8`) are eligible.
- Autonomous Mode Policy Gate: Section 13 policy engine (`service/news_policy.go`) active with risk tiers (LOW, MEDIUM, HIGH) requiring successful canonical publication first. `MASS_AUTOPUBLISH=false` strictly enforced.

## Active Engineering Direction

Core product & growth engineering complete:

```text
[R1-R4 COMPLETED] -> [R6-R7 COMPLETED] -> [R10 COMPLETED] -> [R11 COMPLETED] -> [R11-M COMPLETED] -> [R11-LIVE COMPLETED] -> [R11-PG COMPLETED] -> [R11-STAGING COMPLETED] -> [R11-PROD-PREFLIGHT COMPLETED] -> [R11-PROD-DEPLOY-SAFETY COMPLETED] -> [R11-PROD-DEPLOY COMPLETED (R11 PRODUCTION = PASS)] -> [R11-GROWTH-ACTIVATION COMPLETED] -> [R11-GROWTH-CLOSED-LOOP COMPLETED] -> [R11-SOCIAL-CANARY COMPLETED (DEV.TO LIVE, GSC CONNECTED, FB/LI OPERATOR_BLOCKED)] -> [R5 OPERATOR_BLOCKED] -> [R8 OPERATOR_BLOCKED] -> [R9 BLOCKED]
```

## Truthful Verification State

- `R11 LOCAL POSTGRES = PASS`
- `R11 STAGING = PASS`
- `R11 PROD PREFLIGHT = PASS`
- `R11 PROD DEPLOY SAFETY = PASS`
- `R11 PRODUCTION = PASS`
- `GROWTH CLOSED LOOP = PASS`
- `DEVTO = ACTIVE (LIVE IDEMPOTENT IN PRODUCTION — ARTICLE 4804790)`
- `GSC = CONNECTED (LIVE PROVEN ON sc-domain:toraapi.com — DELTA-AWARE INGESTION)`
- `GA4 = OPERATOR_BLOCKED (NO ACTIVE GTAG / MEASUREMENT ID ON WEB ORIGIN)`
- `FACEBOOK = OPERATOR_BLOCKED (AWAITING OPERATOR CREDENTIALS)`
- `LINKEDIN = OPERATOR_BLOCKED (AWAITING OPERATOR CREDENTIALS)`
- `BACKLOG SAFETY INVARIANT = PASS (SUPPRESSED HISTORICAL BACKLOG)`
- `AUTONOMOUS POLICY GATE = PASS (MASS_AUTOPUBLISH=FALSE STRICTLY ENFORCED)`

## Currently Executing Task

**FINAL STATUS: SOCIAL CANARIES PARTIALLY VERIFIED — OPERATOR CREDENTIALS REQUIRED**:
1. **Delta-Aware GSC Scheduling**: Made Search Analytics ingestion delta-aware; cached finalized historical partitions immutably; fetching only missing/recent partitions with `dataState=all`; tagged observations as `FINAL` (age > 3 days) or `PARTIAL`; selective URL inspection limited to 2/run.
2. **GA4 Readiness Audit**: Audited `https://www.toraapi.com` web frontend — no active GA4 tag found; marked `GA4 = OPERATOR_BLOCKED`. Implemented complete read-only GA4 Data API client in `service/news_ga4.go`.
3. **Facebook & LinkedIn Connectors & Operator Boundary**: Graph API v26.0 and LinkedIn REST API 202609 connectors updated with token/ID verification, pre-flight idempotency checks, and explicit `OPERATOR_BLOCKED` failure classification when credentials are not configured.
4. **Backlog Safety Invariant**: Enforced strict suppression of historical backlog (~4,742 rows) while allowing canary allowlist post 8. Computed and exposed `raw_pending_count`, `eligible_pending_count`, `suppressed_historical_count`, and `canary_eligible_count` in overview telemetry.
5. **Autonomous Mode Policy Engine**: Built `service/news_policy.go` evaluating publish eligibility based on editorial risk (`LOW` -> candidate for future auto-publish, `MEDIUM` -> review required, `HIGH` -> never auto-publish). Canonical Tora publication verified first. `MASS_AUTOPUBLISH=false` strictly enforced.

