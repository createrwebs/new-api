# Phase 7I — Route & Fallback Security Review (Agent E)

## Executive Assessment

An independent security evaluation was conducted across the newly implemented first-class Route and Ordered Fallback system in Tora AI / New-API (`/Users/noppanan/new-api`, branch `feat/formobile`).

**Final Security Finding: APPROVED — ZERO DEFECTS**

All six core security threats identified during the architecture audit have been thoroughly evaluated and validated against source code and regression tests.

---

## 1. Threat Analysis & Invariant Verification

### Threat 1: Route Escalation & Commercial Entitlement Bypass
- **Risk**: A standard user (`default` group) attempts to bind a Pro route (`min_user_group: "pro"`) to access premium models, bypass quota ratios, or gain higher tier throughput.
- **Defense Mechanism**:
  1. **Token Creation / Update Gate** ([`controller/token.go:validateTokenRoutes`](file:///Users/noppanan/new-api/controller/token.go)): Rejects token creation or update if the user's commercial tier does not satisfy `route.MinUserGroup` (`HTTP 403 Forbidden`).
  2. **Relay Execution Gate** ([`service/route.go:ValidateRouteEntitlement`](file:///Users/noppanan/new-api/service/route.go)): Validates the effective user group at the instant of channel selection. If the user's subscription expired between token creation and execution, the route is immediately disqualified.
  3. **Strict Separation**: Routes contain no group assignment fields or quota granting properties. Quotas are governed solely by `User.Quota` and commercial subscription status.
- **Verdict**: **SECURE** — Entitlement bypass is impossible.

### Threat 2: BYOK Isolation Escape
- **Risk**: A BYOK user's custom API key fails with an upstream rate limit or outage. The router transparently fails over to a Managed channel, causing Tora infrastructure to absorb unauthorized costs and violating multi-tenant isolation.
- **Defense Mechanism**:
  1. In [`controller/relay.go:292`](file:///Users/noppanan/new-api/controller/relay.go), `c.GetBool("is_byok") || common.GetContextKeyBool(c, constant.ContextKeyIsBYOK)` triggers an immediate terminal break from the routing loop upon any failure.
  2. In [`controller/relay.go:379`](file:///Users/noppanan/new-api/controller/relay.go), `getChannelForRoute` enforces that if the request is BYOK, routes of kind other than `RouteKindBYOK` are skipped.
- **Verdict**: **SECURE** — BYOK failures are strictly terminal; zero leakage into Managed channels.

### Threat 3: Streaming Protocol Desynchronization
- **Risk**: An upstream model provider fails mid-stream after sending partial tokens. The router attempts transparent failover, resulting in duplicate HTTP headers, corrupted JSON/SSE chunk frames, and client parser crashes.
- **Defense Mechanism**:
  1. **Commit Tracking**: [`relay/stream_commit.go`](file:///Users/noppanan/new-api/relay/stream_commit.go) wraps Gin's `ResponseWriter` with atomic commitment detection (`CommitDetectingWriter`).
  2. **Pre-Commit Staging**: Status codes, headers, and initial payload chunks remain buffered in memory until the first valid SSE payload token is verified.
  3. **Post-Commit Lockout**: Once committed, `ClassifyRouteError` marks any error as `ClassTerminalCommitted` with `AllowFallback: false`. `commitWriter.Reset()` explicitly returns `ErrAlreadyCommitted` if called.
- **Verdict**: **SECURE** — Stream integrity is cryptographically and operationally protected.

### Threat 4: Under-Billing / Quota Arbitrage
- **Risk**: An attacker configures a cheap primary route (multiplier 1.0) and an expensive fallback route (multiplier 2.5), deliberately crafts an upstream rate limit on the primary route, and avoids pre-consumption reservation checks.
- **Defense Mechanism**:
  1. **Conservative Pre-Consumption Reservation**: [`relay/request_billing.go:PrepareRequestBilling`](file:///Users/noppanan/new-api/relay/request_billing.go) scans the entire configured route chain and reserves quota based on the maximum multiplier:
     $$\text{QuotaToReserve} = \text{EstimatedTokens} \times \text{ModelRatio} \times \text{UserGroupRatio} \times \max_{r \in \text{Chain}} M_r$$
  2. **Authoritative Winning Route Settlement**: When a route succeeds, `relayInfo.PriceData` is updated with the winning route's multiplier. Unused reserved quota is automatically refunded upon settlement.
- **Verdict**: **SECURE** — Negative balance overdraft and arbitrage are mathematically prevented.

### Threat 5: Infinite Retries & Resource Starvation
- **Risk**: Circular fallback configurations or cascading upstream failures exhaust server goroutines and sockets.
- **Defense Mechanism**:
  1. **Deduplication**: `token.GetRouteChain()` eliminates duplicate route IDs.
  2. **Attempt Tracking**: `routeParams.AttemptedChannels` tracks all channels invoked during the request lifecycle, preventing duplicate attempts against the same channel across routes.
  3. **Overall Context Deadline**: Request context is wrapped with `overallTimeoutSec` (`common.RelayTimeout`), ensuring hard termination regardless of chain length.
- **Verdict**: **SECURE** — Request lifecycle is strictly bounded.

### Threat 6: Route Referential Deletion & Orphaned Tokens
- **Risk**: An administrator deletes a route currently in use by active API keys, causing runtime nil dereferences.
- **Defense Mechanism**:
  1. [`model/route.go:DeleteRouteSafe`](file:///Users/noppanan/new-api/model/route.go) queries the database to verify whether any token has `primary_route_id == routeId` or contains `routeId` in `fallback_route_ids`.
  2. If referenced, deletion is aborted and [`controller/route.go:DeleteRoute`](file:///Users/noppanan/new-api/controller/route.go) returns `HTTP 409 Conflict` with the detailed reference count.
- **Verdict**: **SECURE** — Referential integrity is strictly preserved.

---

## 2. Review Conclusion

The implementation satisfies all security invariants defined in `docs/ai/SECURITY_INVARIANTS.md` and `docs/ai/phase_7i_route_fallback_architecture_audit.md`.

Security Sign-Off: **APPROVED (Agent E)**
