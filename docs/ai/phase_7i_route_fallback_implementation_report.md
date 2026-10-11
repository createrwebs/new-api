# Phase 7I — First-Class Routes & Ordered Fallback Implementation Report

## Executive Summary

Phase 7I has implemented the first-class Route Engine and ordered fallback routing system for Tora AI / New-API (`/Users/noppanan/new-api`, branch `feat/formobile`) according to the approved architecture specification in [`phase_7i_route_fallback_architecture_audit.md`](file:///Users/noppanan/new-api/docs/ai/phase_7i_route_fallback_architecture_audit.md).

The implementation establishes first-class `Route` entities, decouples routing policies from commercial entitlement groups, enforces transparent streaming fallback boundaries via `CommitDetectingWriter`, guarantees BYOK isolation, enforces conservative pre-consumption quota reservations, and preserves 100% backward compatibility for legacy non-routed API keys.

---

## 1. Prime Invariants Verification

| Invariant | Description | Verification Status | Implementation Enforcement |
| :--- | :--- | :--- | :--- |
| **0.1 Route != Entitlement Group** | Commercial entitlement (`default`, `pro`) governs quota and model visibility; Route governs upstream channel selection. | **VERIFIED** | `Route` entity has no entitlement-granting fields. `ValidateRouteEntitlement` ensures callers cannot invoke routes exceeding their subscription tier. |
| **0.2 BYOK Terminal Boundary** | BYOK channels cannot fallback to Managed channels; BYOK errors are terminal. | **VERIFIED** | Enforced in `controller/relay.go:379` (`is_byok` check) and `getChannelForRoute`. |
| **0.3 Streaming Commitment Boundary** | Fallback is strictly forbidden once any byte or header is emitted downstream. | **VERIFIED** | `CommitDetectingWriter` tracks commitment state. Once `committed == true`, `ClassifyRouteError` forces `ClassTerminalCommitted` with `AllowFallback: false`. |
| **0.4 Billing Authority** | Conservative pre-charge reserves max multiplier across chain; settled against winning route. | **VERIFIED** | `PrepareRequestBilling` reserves `maxMultiplier`; `Relay` sets winning route multiplier upon success. Tested in `service/route_billing_test.go`. |
| **0.5 Backward Compatibility** | Legacy tokens (`primary_route_id == 0`) retain identical behavior. | **VERIFIED** | Tokens without routes bypass `CommitDetectingWriter` and execute standard New-API channel selection. |

---

## 2. Implemented Architecture & Source Modules

### 2.1 Data Models & Migrations
- **[`model/route.go`](file:///Users/noppanan/new-api/model/route.go)**:
  - Definition of `Route` entity (`Id`, `Name`, `Slug`, `Description`, `Enabled`, `Kind`, `RoutingPolicy`, `ChannelIds`, `ChannelTags`, `ModelMapping`, `CostMultiplier`, `MinUserGroup`, `CreatedAt`, `UpdatedAt`, `DeletedAt`).
  - Safe deletion guard `DeleteRouteSafe`: prevents deletion of routes currently referenced by active tokens with `ErrRouteReferencedByKeys`.
  - CRUD operations and helper methods (`GetChannelIds`, `SetChannelIds`, `GetChannelTags`, `SetChannelTags`, `GetModelMapping`, `SetModelMapping`, `CleanRouteCache`).
- **[`model/token.go`](file:///Users/noppanan/new-api/model/token.go)**:
  - Added `PrimaryRouteId int` (`gorm:"default:0;index"`).
  - Added `FallbackRouteIds string` (`gorm:"type:text"`).
  - Implemented `GetFallbackRouteIds() []int`, `SetFallbackRouteIds([]int)`, `GetRouteChain() []int`, and `IsRouteEnabled() bool`.
  - Updated `Update()` select column list to persist route configuration.
- **[`model/main.go`](file:///Users/noppanan/new-api/model/main.go)**:
  - Registered `&Route{}` in database auto-migration list.

### 2.2 Routing Engine & Selection Logic
- **[`service/route_classifier.go`](file:///Users/noppanan/new-api/service/route_classifier.go)**:
  - Authoritative classification function `ClassifyRouteError(ctx, err, isCommitted)`.
  - Distinguishes `ClassFallbackEligible` (429, 502/503/504, pre-response drops), `ClassProviderAuthFailure` (401/403 upstream), `ClassTerminalClient` (400, 404, 422, 413, 415), `ClassTerminalLocalAuth`, `ClassTerminalQuota`, `ClassTerminalCommitted`, and `ClassClientCancelled`.
- **[`service/route.go`](file:///Users/noppanan/new-api/service/route.go)**:
  - High-performance caching layer (`CacheGetRoute`, `CleanRouteCache`) with Redis invalidation broadcast (`tora:cache:route:invalidate`).
  - Channel cooldown management (`SetChannelCooldown`, `IsChannelInCooldown`) supporting automatic 15-second rate-limit backoffs.
  - Priority and weighted channel selection (`SelectNextChannelInRoute`) resolving upstream priority inversion bugs by selecting deterministically from highest-priority available tier and excluding previously attempted channels within the request lifecycle.
  - Upstream model mapping resolution (`ResolveModelForRoute`) and commercial entitlement validation (`ValidateRouteEntitlement`).

### 2.3 Streaming Response Protection & Commit Detection
- **[`relay/stream_commit.go`](file:///Users/noppanan/new-api/relay/stream_commit.go)**:
  - `CommitDetectingWriter` wrapping Gin's `ResponseWriter`.
  - Buffers headers, status codes, and early chunks until route viability is confirmed.
  - Exposes `Reset()` for non-committed attempts to wipe staged data prior to fallback.
  - Exposes `Commit()` to flush headers and buffers directly to client connection upon success or terminal exit.
  - Returns `ErrAlreadyCommitted` if reset is attempted after commitment.

### 2.4 Conservative Billing Pipeline
- **[`relay/request_billing.go`](file:///Users/noppanan/new-api/relay/request_billing.go)**:
  - Extended `PrepareRequestBilling` to inspect token route chain.
  - Computes `maxMultiplier = max_{r in Chain}(CostMultiplier)` and scales conservative pre-consumption quota reservation.
- **[`controller/relay.go`](file:///Users/noppanan/new-api/controller/relay.go)**:
  - Replaces `route_multiplier` in `relayInfo.PriceData` with the winning route's actual multiplier upon successful attempt.
  - Quota settlement refunds the difference between the conservative pre-consumption ceiling and the winning route's actual cost.
  - Formula: $\text{SettledQuota} = \lceil \text{BaseQuota} \times \text{ModelRatio} \times \text{UserGroupRatio} \times \text{WinningRouteCostMultiplier} \rceil$.

### 2.5 API Endpoints & Observability
- **[`controller/route.go`](file:///Users/noppanan/new-api/controller/route.go)**:
  - Admin CRUD: `GET /api/routes`, `GET /api/routes/:id`, `POST /api/routes`, `PUT /api/routes`, `DELETE /api/routes/:id`.
  - User Available Routes: `GET /api/routes/available` (filtered by caller's entitlement group).
  - Safe deletion returning HTTP 409 Conflict if route is bound to active tokens.
- **[`controller/token.go`](file:///Users/noppanan/new-api/controller/token.go)**:
  - Updated `AddToken` and `UpdateToken` to accept and validate `primary_route_id` and `fallback_route_ids`.
  - Validates route existence, enablement, and user subscription eligibility.
- **[`router/api-router.go`](file:///Users/noppanan/new-api/router/api-router.go)**:
  - Registered route endpoints in user and admin router groups.
- **[`service/log_info_generate.go`](file:///Users/noppanan/new-api/service/log_info_generate.go)**:
  - Added structured route audit fields to relay logs: `winning_route_id`, `route_name`, `route_multiplier`, and admin-only `route_chain_trace`.

### 2.6 Web UI Schemas
- **[`web/src/features/keys/types.ts`](file:///Users/noppanan/new-api/web/src/features/keys/types.ts)**:
  - Extended `apiKeySchema` with `primary_route_id` (number, default 0) and `fallback_route_ids` (array of numbers, default `[]`).
  - Added `RouteItem` interface for UI dropdown bindings.

---

## 3. Test & Verification Summary

1. **Model Tests** (`go test ./model -count=1`): **PASS** (10.39s)
   - Tested route serialization, deduplication, safe deletion, token route chains.
2. **Service Tests** (`go test ./service -count=1`): **PASS** (2.72s)
   - Tested error classifier (all classes), route engine, cooldowns, entitlement, model mapping, conservative billing.
3. **Relay Tests** (`go test ./relay -count=1`): **PASS** (2.10s)
   - Tested `CommitDetectingWriter` pre-commit buffer, reset, commit, already-committed errors, billing multipliers.
4. **Controller Tests** (`go test ./controller -count=1`): **PASS** (33.37s)
   - Tested route relay execution, admin CRUD, token validation, responses websocket/streaming compatibility.
5. **Static Analysis** (`go vet ./...`): **PASS** (0 warnings/errors)
6. **Web Frontend Typecheck** (`bun run typecheck`): **PASS** (0 errors)
7. **Mobile Client Regression** (`flutter analyze`, `flutter test`): **PASS** (117/117 tests passed)

---

## 4. Conclusion & Readiness

Phase 7I has successfully implemented first-class routes and ordered fallback routing with complete architectural decoupling, mathematical billing accuracy, and deterministic streaming safety.

Final Status:
`FINAL STATUS: FIRST-CLASS ROUTES IMPLEMENTED — READY FOR STAGING`
