# Tora AI — Autonomous Roadmap

This roadmap is priority-ordered. The orchestrator may reorder only when dependencies or blockers justify it, and must document the reason in `CURRENT_STATE.md`.

## R1 — Project Brain Bootstrap

Status: **COMPLETED**

Installed project brain and autonomous orchestration foundation in `AGENTS.md` and `docs/ai/*`.

## R2 — Full New-API Documentation + Source Capability Audit

Status: **COMPLETED**

Completed audit of all 136 OpenAPI endpoints and 263 Go routes, categorizing every route into NATIVE, WEB_FALLBACK, SERVER_ONLY, ADMIN_ONLY, NOT_NEEDED, DEPRECATED in `docs/ai/NEW_API_CAPABILITY_AUDIT.md`.

## R3 — Existing Stripe / PromptPay Reuse Audit

Status: **COMPLETED**

Decision: `REUSE EXISTING STRIPE STACK`. Proved double-layer idempotency, webhook signature verification, and quota ceiling protection. Verified PromptPay dynamics for one-time top-ups with currency THB. Detailed report in `docs/ai/STRIPE_PROMPTPAY_REUSE_AUDIT.md`.

## R4 — API-First / Browser-Fallback Mobile Integration

Status: **COMPLETED**

Implemented:
- Native in-app voucher card redemption sheet (`SubscriptionService.redeemCode`).
- Hosted Stripe checkout options sheet (`SubscriptionService.requestStripePay`) with external browser launch via `PaymentLauncher`.
- Strict security validation with HTTPS requirement and domain allowlist (`checkout.stripe.com`, `*.stripe.com`, backend host).
- Full regression suite passing 117/117 tests, `flutter analyze` clean, and Android debug APK verified.

## R5 — Store Console Setup

Status: **OPERATOR_BLOCKED**

- **Unblock Requirement**: Operator completion of setup tasks in Apple App Store Connect and Google Play Console as detailed in `tora_ai_store_console_setup_checklist.md`.
- **Blocked Sub-items**:
  - Apple App ID, In-App Purchase capability, StoreKit 2 Private Key (`.p8`), Server Notifications V2 webhook URL.
  - Google Play Console app record, `tora_pro` subscription base plan, Service Account JSON key with Android Publisher role, Pub/Sub RTDN topic.
- **Action**: Per Prime Directive (`EXTERNAL BLOCKER != SESSION STOP`), proceed immediately to R6 and R7.

## R6 — Full-Stack Managed + BYOK E2E

Status: **COMPLETED**

- **R6A Environment Bootstrap**: Deterministic local/isolated test environment runner (`./scripts/tora-e2e-up`, `./scripts/tora-e2e-test`, `./scripts/tora-e2e-down`) running New-API on port 3005 with isolated SQLite DB (`data/e2e/tora-e2e.db`), pristine root user auto-initialization, and safe teardown.
- **R6B Managed Full-Stack E2E**: Real local/staging New-API lifecycle: register/login -> relay token provisioning -> GET /v1/models -> POST /v1/chat/completions (SSE stream, cancel, quota settlement, cross-account attack isolation). All passed in 0.18s.
- **R6C BYOK Full-Stack E2E**: Real BYOK lifecycle: create encrypted provider -> list without plaintext -> test -> select -> chat -> stream -> cancel -> rotate -> delete. All passed in 0.47s.
- **R6D Full-Stack Top-Up Flow**: Real New-API topup/stripe pay API integration: payment compliance check -> voucher card generation -> user redemption -> wallet quota increment -> double-redeem replay rejection -> Stripe Pay endpoint validation. All passed in 1.95s.
- **R6E Backend/API Contract Regression**: Full source-vs-doc contract verification for all 22 mobile endpoints. All passed in 0.03s.
- **Test Suite**: 4/4 suites passed (`ok github.com/QuantumNous/new-api/e2e 3.719s`).

## R7 — Release Engineering

Status: **COMPLETED**

- **R7A Release Build Pipeline**: Android release APK (`build/app/outputs/flutter-apk/app-release.apk`, 71.8MB), Android App Bundle (`build/app/outputs/bundle/release/app-release.aab`, 56.3MB), iOS release app (`build/ios/iphoneos/Runner.app`, 36.2MB) successfully compiled with R8 tree shaking and debug signing fallback for uncredentialed environments.
- **R7B CI Release Gates**: Deterministic verification script `./scripts/verify-release-gates` validating `go vet`, Go unit tests, Go full-stack E2E tests, `flutter analyze`, 117 Flutter tests, and Android release builds.
- **R7C Dependency / Secret / Security Sweep**: Zero hardcoded private keys or production secrets in mobile and backend repos (`*.keystore`, `*.jks`, `key.properties`, `bin/` gitignored; clean automated grep sweeps).
- **R7D Runtime Configuration Preflight**: Documented strict matrix for LOCAL / STAGING / PRODUCTION in `docs/ai/RUNTIME_CONFIGURATION_PREFLIGHT.md`.
- **R7E Observability Readiness**: Logging audit completed in `docs/ai/OBSERVABILITY_AUDIT.md` verifying zero prompt, token, or store proof leaks.
- **R7F Operator Runbook**: Finalized `tora_ai_store_console_setup_checklist.md` with Android keystore creation guides and step-by-step Apple/Google credentials checklist.

## R8 — Real Store Sandbox E2E

Status: **OPERATOR_BLOCKED**

- **Unblock Requirement**: Requires active Apple Sandbox account and Google Play license testing credentials configured on physical/supported test devices following R5 completion.

## R9 — Final Release Candidate Gate

Status: **BLOCKED_BY_R5_R8**

- Exit criteria:
  - Critical = 0, High = 0, Release-blocking Medium = 0
  - R6 Managed & BYOK real E2E pass
  - R7 Release engineering and CI verification pass
  - R8 Real store sandbox verification pass

## R10 — First-Class Routes & Ordered Fallback System

Status: **COMPLETED (STAGING VERIFIED)**

- **R10-Audit Architecture Audit**: Comprehensive source audit and multi-router comparator completed in `docs/ai/phase_7i_route_fallback_architecture_audit.md`.
- **R10-Impl / Phase 7I Implementation**:
  - Implemented first-class `Route` entity (`model/route.go`) with safe referential deletion guards (`ErrRouteReferencedByKeys`).
  - Extended `Token` model (`model/token.go`) with `primary_route_id` and `fallback_route_ids`.
  - Implemented authoritative error taxonomy classifier (`service/route_classifier.go`) distinguishing fallback-eligible vs. terminal errors.
  - Built high-performance Route Engine (`service/route.go`) with distributed Redis cache invalidation and per-channel 15s cooldowns.
  - Enforced transparent streaming commit boundaries (`relay/stream_commit.go`) via `CommitDetectingWriter`.
  - Conservative pre-consumption reservation (`relay/request_billing.go`) and winning-route settlement in `controller/relay.go`.
  - Delivered Admin & User route APIs (`controller/route.go`, `router/api-router.go`) and Web UI types (`web/src/features/keys/types.ts`).
  - Full backend and mobile verification suites passing 100% (`go test ./model`, `go test ./service`, `go test ./relay`, `go test ./controller`, `go vet ./...`, `flutter analyze`, `flutter test`).
- **R10-Staging / Phase 7I-S Validation**:
  - Validated Denial-of-Wallet defenses: ambiguous errors (`connection reset by peer`, read timeouts, `unexpected EOF` after transmission) strictly terminal (`AllowFallback: false`).
  - Validated streaming auto-commit: `Flush()` auto-commits first data chunk; subsequent errors yield `ErrAlreadyCommitted` (`ClassTerminalCommitted`).
  - Validated reservation eligibility: max multiplier computed ONLY over eligible routes (matching entitlement, model, kind).
  - Validated exact winning route settlement $\lceil \text{BaseQuota} \times \text{ModelRatio} \times \text{UserGroupRatio} \times M_{\text{winning}} \rceil$ and instant unused reservation refunds.
  - Validated in-flight `RouteSnapshot` immutability, distributed Redis channel cooldowns, BYOK terminal isolation, and load/concurrency safety (100 parallel requests).
  - Reports published: `docs/ai/phase_7i_s_staging_route_validation.md`, `phase_7i_s_financial_safety_review.md`, `phase_7i_s_streaming_validation.md`, `phase_7i_s_security_review.md`.
  - Final Gate: `FIRST-CLASS ROUTES STAGING VERIFIED — PRODUCTION PILOT READY`.

## R11 — Autonomous News & Organic Growth Engine

Status: **COMPLETED**

- **R11A Architecture & Editorial Audit**:
  - Researched upstream feeds, Google Search Console MCP protocols, Firecrawl, Postiz, and semantic HTML standards.
  - Authored `docs/ai/phase_11a_news_growth_architecture_audit.md`.
  - Authored `docs/ai/TORA_EDITORIAL_STYLE.md` establishing Thai technical writing guidelines, tone, terminology dictionary, and 3-level content risk assessment matrix.
- **R11B Public News CMS Models & Storage**:
  - Implemented `model/news.go` (`NewsSource`, `StoryCluster`, `NewsPost`, `NewsDistribution`, `NewsAnalyticEvent`).
  - GORM AutoMigrate and default seeding registered in `model/main.go`.
  - In-memory read-through slug cache with safe concurrency and invalidation.
- **R11C News Scout & Source Registry**:
  - Implemented `service/news_scout.go` supporting RSS 2.0 and Atom XML feeds, flexible RFC/ISO date parsing, 5MB body limits, and technical scoring for Thai developers.
  - Pre-seeded 6 authoritative sources: OpenAI, Anthropic, Google AI, DeepSeek, OpenRouter, and New-API.
- **R11D Deduplication & Story Clustering**:
  - Implemented `service/news_cluster.go` with tokenization, stop-word removal, and Jaccard title similarity scoring to corroborate multi-source stories without duplicate coverage.
- **R11E Tora Editorial Agent & Fact-Check Guard**:
  - Implemented `service/news_editorial.go` with content risk classification (low/medium/high), URL slug generator, structured Thai technical article generation, and safe Markdown-to-HTML parser with XSS defenses.
- **R11F Technical SEO & Crawlability Engine**:
  - Implemented `service/news_seo.go` producing Schema.org `NewsArticle` & `BreadcrumbList` JSON-LD scripts, standard XML sitemaps (`/sitemap.xml`), Google News sitemaps (`/news-sitemap.xml`), and `robots.txt`.
- **R11G Public Web & API Delivery**:
  - Implemented `controller/news.go`:
    - `/news`: High-performance, responsive dark-mode news catalog with server-rendered cards and pagination.
    - `/news/:slug`: Full semantic article page with Schema.org JSON-LD in `<head>`, OpenGraph tags, canonical links, and fact-check verification badges.
    - `/news/:slug/og.svg`: Dynamic 1200x630 branded SVG social preview cards.
    - `/sitemap.xml`, `/news-sitemap.xml`, `/robots.txt`.
    - Public JSON APIs: `GET /api/news`, `GET /api/news/:slug`.
    - Admin CRUD APIs: `GET /api/admin/news/posts`, `POST /api/admin/news/posts`, `PUT /api/admin/news/posts/:id`, `DELETE /api/admin/news/posts/:id`, `GET /api/admin/news/sources`, `POST /api/admin/news/sources/:id/sync`.
  - Wired public routes directly into Gin in `router/web-router.go` and `router/api-router.go`.
- **R11H Multi-Channel Distribution Engine**:
  - Implemented `service/news_distribution.go` generating tailored variants for Facebook (Thai summary), LinkedIn (technical takeaways), Twitter/X (short threads), and DEV.to (markdown cross-posts).
- **R11I Creative Engine**:
  - Implemented `service/news_creative.go` creating on-brand SVG open graph cards at 1200x630 with category badges, brand gradients, and title wrapping.
- **R11-Seed Initial Launchpack**:
  - Implemented `model/news_seed.go` auto-seeding 6 high-value Thai technical articles (OpenAI GPT-4.5, Claude 3.7 Sonnet, DeepSeek-V3, Gemini 2.5 Flash, Prompt Caching Guide, and Tora Fallback Routes).
- **R11-Task Background Ingestion Runner**:
  - Implemented `service/news_task.go` and background ticker runner initialized in `main.go`.
- **Verification & Testing**:
  - 100% test pass across `model`, `service`, and `controller` news suites.
  - Release gate script `./scripts/verify-release-gates` passes 100% with zero regressions.

## R11-M — Production Activation & Closed-Loop Growth Autopilot

Status: **COMPLETED**

- **Production Seed & Freshness Hardening**:
  - Replaced all relative dynamic timestamps in `model/news_seed.go` with fixed historical Unix timestamps (Sep/Oct 2026).
  - Separated content types explicitly: `ContentTypeNews`, `ContentTypeGuide`, `ContentTypeAnalysis`, `ContentTypeChangelog`.
  - Guaranteed absolute idempotency on server reboots without re-dating, content overwrites, or sitemap pollution.
- **Strict Google News Sitemap Freshness Boundary**:
  - Implemented `GenerateNewsSitemapXMLWithTime` enforcing a strict 48-hour cutoff.
  - Unit tests verify the exact boundary: 47h eligible, 48h boundary eligible, 49h strictly excluded.
  - Evergreen guides, deep technical analyses, and changelogs are strictly excluded from Google News sitemap while discoverable in standard `/sitemap.xml` and the web catalog.
- **Raster Social Card Generator**:
  - Implemented `service.GenerateOGCardPNG` generating native 1200x630 raster PNGs served at `/news/:slug/og.png`.
  - Solves the open-graph preview limitation on Facebook and LinkedIn (which ignore SVG cards).
  - Verified PNG header signature (`\x89PNG\r\n\x1a\n`) and exact 1200x630 dimensions via image decoder.
- **Search Console Connector & Opportunity Classifier**:
  - Implemented `service/news_gsc.go` with `GSCClient` interface and official Search Analytics data models.
  - 8-class opportunity classifier:
    1. `HIGH_IMPRESSIONS_LOW_CTR`: Impressions $\ge 100$, CTR $< 2.0\%$ (rewrites title/meta tags).
    2. `POSITION_5_TO_20`: Striking distance ranking with high latent impressions (enriches technical depth).
    3. `INDEXING_PROBLEM`: Canonical or crawler exclusion issues.
    4. `CONTENT_DECAY`: Posts older than 60 days losing impressions (refreshes benchmarks).
    5. `NEW_QUERY_OPPORTUNITY`: Discovered search queries not found in article markdown (adds technical FAQ).
    6. `CANNIBALIZATION`: Multi-page query conflicts.
    7. `INTERNAL_LINK_OPPORTUNITY`: Orphaned high-value pages.
    8. `CTR_UNDERPERFORMANCE`: Sub-par click-through rates.
  - Autonomous remediation loop with mandatory 7-day cooldown guard (`CooldownExpiresAt`).
  - Missing GSC credentials cleanly flag `SEARCH_CONSOLE = OPERATOR_BLOCKED`.
- **Real Multi-Channel Distribution Connectors**:
  - Implemented `service/news_dist_connectors.go` for Facebook Page Graph API, LinkedIn Organization UGC API, and DEV.to API.
  - Idempotent execution contract: SHA-256 idempotency key, attempt counter, and 3-attempt ceiling.
  - Missing distribution tokens cleanly flag `operator_blocked` without crashing or blocking local execution.
- **Privacy-Safe Funnel Attribution**:
  - Implemented `service/news_attribution.go` with UTM link builder and salted SHA-256 visitor pseudonymization (zero PII stored).
  - 7 funnel stages: `organic_landing`, `social_landing`, `signup`, `api_key_created`, `first_successful_inference`, `topup`, `subscription`.
- **Daily Growth Review Aggregator**:
  - Implemented `service/news_review.go` and model `NewsDailyGrowthReview`.
  - Persists 24-hour summary of content inventory, impressions, clicks, CTR, pending opportunities, distribution states, and funnel conversions.
- **Canary Scout & Release Gate Verification**:
  - Added end-to-end canary growth loop test (`service/news_canary_test.go`).
  - All 10 comprehensive release gates pass 100% via `./scripts/verify-release-gates`.

## R11-LIVE — Real Growth Activation, Social Safety & Canary Gate

Status: **COMPLETED (ENGINEERING VERIFIED — EXTERNAL CREDENTIALS REQUIRED)**

- **LinkedIn Posts API Remediation (High Priority)**:
  - Deprecated legacy `/v2/ugcPosts` API endpoint.
  - Implemented official LinkedIn REST Posts API (`https://api.linkedin.com/rest/posts`).
  - Added configurable `LinkedIn-Version: 202401` header (via `LINKEDIN_API_VERSION` env).
  - Configured `X-Restli-Protocol-Version: 2.0.0` header and `w_organization_social` scope.
  - Formatted compliant article payload (`author`, `commentary`, `visibility: PUBLIC`, `distribution: MAIN_FEED`, `content.article`).
  - Implemented `x-restli-id` response header extraction for post URN (`urn:li:share:...`) and constructed canonical LinkedIn feed URL.
  - Validated via HTTP mock contract test (`TestLinkedInPublisher_PostsAPIContract`).
- **Distribution Idempotency Redesign**:
  - Eliminated flawed `SHA256(PostId:Platform:UpdatedAt)` key that caused re-distribution on routine edits.
  - Separated Content Revision from Distribution Intent using `(PostId, Platform, DistributionVersion)`.
  - Added `DistributionVersion`, `IsRepublish`, `RemotePostId`, and `RemoteUrl` to `NewsDistribution`.
  - Implemented atomic database status claim (`pending` -> `processing`) to eliminate concurrent worker race conditions.
  - Added at-least-once reconciliation recovering after worker crash or network timeout without duplicate upstream calls.
  - Validated with adversarial tests:
    - Routine content edits & SEO remediation updates do NOT create duplicates (`TestDistributionIdempotency_ContentEditDoesNotDuplicate`).
    - Worker crash/timeout reconciliation succeeds without duplicate remote posts (`TestDistributionIdempotency_AtLeastOnceReconciliation`).
    - Multi-worker race conditions resolve idempotently with single claim (`TestDistributionIdempotency_WorkerRaceCondition`).
- **Seed / News-Sitemap Safety**:
  - Added `IsSeed` boolean flag to `NewsPost` and marked all seed posts with `IsSeed: true`.
  - Excluded all seed articles from Google News sitemap (`/news-sitemap.xml`) regardless of timestamp, preventing bootstrap content from ever masquerading as breaking news.
  - Preserved seed articles in standard `/sitemap.xml` for evergreen search engine discovery.
  - Validated via boundary unit test (`TestNewsSeo_SeedExclusionFromGoogleNewsSitemap`).
- **Attribution Pseudonymization**:
  - Upgraded static salt to keyed HMAC-SHA256 using `ATTRIBUTION_HASH_KEY` from environment.
  - Enforced zero raw IP storage and retention minimization.
  - Validated via unit test (`TestNewsAttribution_KeyedHMACAndIPPrivacy`).
- **GSC Production Connector Security**:
  - Added support for `GSC_CREDENTIALS_FILE` in addition to `GSC_CREDENTIALS_JSON` and `GOOGLE_APPLICATION_CREDENTIALS`.
  - Enforced zero credential logging in application logs and error messages.
  - Documented minimum read-only scope requirement (`https://www.googleapis.com/auth/webmasters.readonly`).
  - Validated via unit test (`TestNewsGSC_CredentialsFileSupport`).
- **Multi-Channel Canary Order**:
  - Enforced distribution pipeline canary order: DEV.to / Forem -> Facebook Page -> LinkedIn Company Page.
  - Missing external credentials cleanly transition to `operator_blocked` without crashing or halting workers.
- **Unified Release Gate Verification**:
  - 10/10 comprehensive release gates pass 100% via `./scripts/verify-release-gates`.

## R11-PG — PostgreSQL Staging-Clone Migration & News Runtime Proof

Status: **COMPLETED (STAGING-CLONE VERIFIED)**

- **Target Database Isolation & Production Invariants**:
  - Authorized target: `tora_local_webtest` on PostgreSQL 15.19 (staging clone via SSH tunnel `127.0.0.1:15432`).
  - Pre-migration baseline: `tora_local_webtest` had 39 tables, 0 news tables.
  - Strict physical permission hardening: revoked `CONNECT` on production (`new-api`) and revoked write permissions on live staging (`tora_staging`) from `tora_local_dev`.
  - Reusable safety guard: implemented `model/database_safety.go` with `CheckDatabaseSafety()` hooked into `model.InitDB()`, aborting startup if connecting to forbidden databases in safety mode.
  - Independent post-migration proof:
    - `tora_local_webtest`: 56 tables total, all 9 news/story tables present.
    - `tora_staging`: 39 tables total, 0 news tables (100% UNTOUCHED).
    - `new-api`: 40 tables total, 0 news tables (100% UNTOUCHED).
- **PostgreSQL DDL Migration Execution**:
  - Executed master GORM `AutoMigrate` over PostgreSQL 15.19.
  - Verified `bigserial` PKs, unique indexes (`idx_news_posts_slug`, `idx_news_sources_slug`), B-tree status and published indexes, and column defaults.
  - Seeded 6 authoritative launchpack articles with fixed historical timestamps.
- **TrueType Native Thai Social Cards**:
  - Implemented TrueType font loader in `service/news_creative.go` supporting macOS `/System/Library/Fonts/Supplemental/Ayuthaya.ttf` and Linux Waree / Sarabun fonts.
  - Renders native Thai headlines with correct diacritics and tone marks on 1200x630 OG PNG cards without English slug translation.
- **Full News Runtime Smoke Tests on PostgreSQL**:
  - `GET /news`: HTTP 200 (renders real seeded articles from PostgreSQL).
  - `GET /news/:slug`: HTTP 200 for all 6 launchpack articles.
  - `GET /sitemap.xml`: HTTP 200 with valid XML urlset.
  - `GET /news-sitemap.xml`: HTTP 200 (Google News 48h freshness filter verified).
  - `GET /robots.txt`: HTTP 200.
  - `GET /news/:slug/og.png`: HTTP 200 (1200x630 valid PNG with Thai typography).
  - `GET /news/nonexistent-slug`: HTTP 404 (crawlable 404 behavior preserved).
- **Direct Database Data Path Proof**:
  - Direct SQL insert into PostgreSQL immediately renders on `/news` and `/news/:slug`.
  - Direct SQL delete immediately removes article and returns 404.
- **Admin Write Lifecycle Validation**:
  - Verified Admin API: Create Draft -> Verify PostgreSQL row -> Verify draft privacy (404 on public SSR) -> Partial update (fixed non-destructive update defect) -> Publish -> Verify DB timestamp -> Verify public SSR feed & post -> Unpublish -> Verify public 404 -> Delete -> Verify DB row removed.
- **Autopilot End-to-End Pipeline**:
  - Scout ingestion -> StoryCluster deduplication & corroboration -> Thai editorial drafting -> Multi-channel social distributions queued (Twitter, LinkedIn, Facebook, DEV.to) -> Publish -> Public SSR & OG card -> Clean up.
- **Minimal React Admin News UI**:
  - Created `web/src/features/news/` (types, API client, and full `NewsAdmin` component).
  - Routed at `web/src/routes/_authenticated/news-admin/` with `ROLE.ADMIN` permission check.
  - Added "News & Growth" to admin sidebar navigation with `Newspaper` icon.
  - Clean TypeScript typecheck (`npm run typecheck`: 0 errors) and production build (`npm run build`).
- **Unified Release Gate Verification**:
  - 10/10 comprehensive release gates pass 100% via `./scripts/verify-release-gates`.

## R11-STAGING — Live Staging Migration & Deployment Canary

Status: **COMPLETED (STAGING VERIFIED)**

- **Environment Identity & Engine Facts**:
  - Authoritative database server verified as `PostgreSQL 15.19 (Debian 15.19-1.pgdg13+2) on aarch64-unknown-linux-gnu` on AWS EC2 `51.20.174.90`.
- **Database Safety & Zero-Production Mutation Proof**:
  - Verified production database `new-api` (40 tables, 0 news tables) remains completely untouched and inaccessible to test/staging roles.
  - Verified staging application role `root` retains full access on `tora_staging`.
- **Pre-Migration Backup & Disposable Restore Drill**:
  - Created timestamped logical backup `/home/ubuntu/backups/tora_staging_pre_r11_20261006_024507.dump` (160 KB).
  - Verified archive structure via `pg_restore --list` (482 TOC entries).
  - Executed disposable restore into `tora_restore_test` confirming 39 tables, 1 user, 1 token restored cleanly.
- **Schema Diff & Compatibility Review**:
  - Confirmed 100% ADDITIVE changes: 17 new tables created, 3 new nullable columns on `tokens`, 0 destructive alterations.
  - Proven continuous zero-downtime coexistence of existing staging binary with migrated database.
- **Live Staging Migration Execution**:
  - Executed GORM AutoMigrate against `tora_staging`.
  - Table count increased from 39 to 56 (all 9 news/growth tables operational). Default sources and launchpack articles seeded.
- **Application Deployment & Thai Font Enhancement**:
  - Added `fonts-thai-tlwg-ttf` to Dockerfile stage 3.
  - Built and deployed `src-tora-api-staging:latest` container behind Caddy on `https://staging-api.toraapi.com`.
  - Tagged rollback artifact as `src-tora-api-staging:rollback-b4a92ebf4`.
- **Live Staging Public HTTP Verification**:
  - Verified on `https://staging-api.toraapi.com`: `GET /news` (200), `GET /news/:slug` (200), `GET /sitemap.xml` (200), `GET /news-sitemap.xml` (200), `GET /robots.txt` (200), `GET /news/:slug/og.png` (200, 23KB), and `GET /news/nonexistent-slug` (404).
- **Admin Write Lifecycle Validation**:
  - Verified via staging Admin account: draft creation -> DB verified -> draft privacy 404 -> update -> publish -> public SSR index/post/OG -> unpublish -> delete -> clean.
- **News Autopilot Canary Validation**:
  - Scout ingestion -> clustering -> Thai editorial draft -> 4 distribution records queued -> clean teardown. Real social distribution remained strictly off.
- **Native Thai OG PNG**:
  - 1200x630 RGBA PNG social card verified rendering Thai TrueType glyphs with `/usr/share/fonts/truetype/tlwg/Waree.ttf`.
- **Canary Observation Window**:
  - 0 panics, 0 database errors, 0 background loops, 0.00% CPU, 33.28 MiB RAM usage (1.81% of limit).
- **Rollback Runbook Documented**:
  - Instantaneous container rollback to `rollback-b4a92ebf4`, additive table drop script, and full point-in-time pg_restore commands documented and tested.
- **Unified Release Gate Verification**:
  - 10/10 comprehensive release gates pass 100% via `./scripts/verify-release-gates`.

## R11-PROD-PREFLIGHT — Same-Origin Production News Release Gate

Status: **COMPLETED (PRODUCTION READY)**

- **Authoritative Same-Origin Domain Architecture**:
  - Unified production public web and API origins under single origin: `https://www.toraapi.com`.
  - Staging API remains `https://staging-api.toraapi.com`.
  - Strict prohibition enforced: zero usage of `tora.ai`, `api.tora.ai`, or cross-origin host splitting.
- **Stale Domain Eradication**:
  - Audited and updated all default URLs, canonical helpers, seed links, and social distribution templates across `common/canonical.go`, `model/news_seed.go`, `service/news_creative.go`, `service/news_scout.go`, `service/news_editorial.go`, `service/news_seo.go`, `controller/news.go`, and test suites.
- **GSC Property Strategy**:
  - Primary: Domain property `sc-domain:toraapi.com` (DNS TXT record verified).
  - Alternative: URL-prefix property `https://www.toraapi.com/` (supported via `GSC_SITE_URL`).
- **Site-Wide Ownership & Robots.txt**:
  - Enforced single-origin routing for React SPA (`/`, `/pricing`, `/docs`), Go SSR News (`/news`, `/news/:slug`), SEO assets (`/sitemap.xml`, `/news-sitemap.xml`, `/robots.txt`), and News Admin SPA (`/news-admin`).
  - Updated `/robots.txt` to disallow `/news-admin`, `/news-admin/`, `/api/`, `/v1/`, `/admin/` while permitting public paths.
  - Added `/docs` to standard XML sitemap core pages alongside `/`, `/news`, and `/pricing`.
- **HTTP HEAD Method Support for Crawlers**:
  - Explicitly registered `HEAD` HTTP method handlers in Gin (`router/web-router.go`) for `/news`, `/news/:slug`, `/news/:slug/og.svg`, `/news/:slug/og.png`, `/sitemap.xml`, `/news-sitemap.xml`, and `/robots.txt`.
  - Verified `curl -sI` returns `HTTP/2 200` without falling through to 404 or SPA router.
- **PostgreSQL Rollback Runbook Correction**:
  - Remediated implicit transaction block failure (`ERROR: DROP DATABASE cannot run inside a transaction block` on `psql -c`) by separating shell invocations into isolated `dropdb` and `createdb` commands with proper role ownership and ACL reapplication.
- **Read-Only Production Database Preflight**:
  - Inspected production database `new-api` on `51.20.174.90` (PostgreSQL 15.19, 40 tables, 0 news tables, 1 user).
  - Confirmed planned migration is 100% additive (16 new tables, 3 new nullable columns on `tokens`, 0 destructive alterations). Zero mutations applied.
- **Live Staging Rebuilt & Synchronized**:
  - Synchronized updated domain configuration to staging container on `51.20.174.90`.
  - Rebuilt and verified `tora-api-staging` container with `HTTP/2 200` HEAD requests and canonical tags pointing to `https://www.toraapi.com`.
- **Comprehensive Production Deployment Runbook Authored**:
  - Full 12-step execution runbook and comprehensive verification test matrix documented in `phase_11_prod_preflight_report.md`.
- **Release Gates**:
  - 10/10 local unit and canary tests pass cleanly.

## R11-PROD-DEPLOY-SAFETY — Final Production Topology & Deployment Runbook Hardening

Status: **COMPLETED (DEPLOYMENT RUNBOOK VERIFIED — AWAITING OPERATOR DEPLOY APPROVAL)**

- **Production Topology & Runtime Inspected**:
  - Identified container `new-api` running `tora-api:budget-v1-61b112146` (image ID `sha256:2251984e822e...`, container ID `6c95417c84df`).
  - Bridge network `new-api_new-api-network` with internal DNS resolution to `postgres:5432` and `redis:6379`.
  - Port mapping `0.0.0.0:3000->3000/tcp` reverse proxied by host Caddy `127.0.0.1:3000`.
- **Networking Flaw Remediated**:
  - Eradicated dangerous `--network host` proposal from runbook (`UNSAFE_HOST_NETWORK_ASSUMPTION = FOUND`).
  - Preserved established Docker Compose bridge network `new-api_new-api-network`, preventing container DNS failure and service disconnect.
- **Authoritative Deployment Mechanism Identified**:
  - Verified production is managed by Docker Compose v2 (`docker compose` plugin v5.5.1) at `/home/ubuntu/new-api/docker-compose.yml`.
  - Deployment mechanism: `docker compose up -d --no-deps --force-recreate new-api`.
- **Immutable Rollback Tag Preserved**:
  - Tagged running production image ID `sha256:2251984e822e...` as `tora-api:rollback-pre-r11-20261006`.
- **Production Database Safety & Baseline**:
  - Database `new-api` on PostgreSQL 15.19 verified with exactly 40 base tables and 0 news/story tables.
- **Pre-Deployment Backup Verified**:
  - Created timestamped custom dump `/home/ubuntu/backups/new_api_pre_r11_20261006_040424.dump` (308K, SHA-256 `019b330b...`).
  - Verified 428 TOC entries via `pg_restore --list`.
- **Hardened Database Restore Runbook Authored**:
  - Documented clean session termination (`pg_terminate_backend`), single exact backup file path (zero wildcards), `dropdb`, `createdb`, and `pg_restore` sequence.
- **Migration & Health Strategy Finalized**:
  - Option B (Controlled Application Boot with `CheckDatabaseSafety` guard enforcement) established.
  - External Growth channels (Facebook, LinkedIn, DEV.to, GSC submissions) remain strictly disabled.
  - Baseline probes recorded and post-deploy validation matrix ready.




