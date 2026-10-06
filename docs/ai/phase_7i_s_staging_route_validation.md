# Phase 7I-S — Staging Route / Fallback Validation & Multi-Agent Engineering Report

## Mission & Executive Summary

Phase 7I-S validates the First-Class Route & Ordered Fallback system implemented in Phase 7I under rigorous staging and adversarial conditions before any production rollout.

The staging gate explicitly verifies that:
1. Upstream network drops and ambiguous disconnections cannot trigger duplicate upstream billing or denial-of-wallet amplification.
2. Real HTTP/SSE stream commitments permanently terminate fallback once response bytes are transmitted downstream.
3. Model mapping, channel group matching, and commercial subscription tiers (Free vs. Pro) are strictly enforced at execution time.
4. Billing calculations, conservative pre-charge reservations, and settlement refunds are mathematically exact.
5. Route definitions are frozen into immutable snapshots (`RouteSnapshot`) at request start to prevent race conditions during concurrent admin edits.
6. Channel cooldowns operate reliably across multi-node environments via distributed Redis tracking.
7. Existing non-routed tokens and legacy workflows suffer zero regression.

**Final Gate Assessment**: `FINAL STATUS: ROUTES STAGING VERIFIED — PRODUCTION PILOT READY`

---

## 1. Scope & Release Boundary

To ensure absolute operational safety:
- **Production Tokens Unaffected**: Routes are strictly opt-in (`primary_route_id == 0` maintains 100% legacy routing behavior).
- **Zero Customer Migration**: Existing production tokens have not been altered or migrated.
- **No Public Commercial Exposure**: Route management remains restricted to authenticated operators (`role >= 100`) and private internal tokens.

---

## 2. Test Execution Matrix & Results

All test suites were executed cleanly in local and staging configurations:

| Component | Target / Suite | Verification Command | Outcome |
| :--- | :--- | :--- | :--- |
| **Route Error Classifier** | `service/route_classifier_test.go` | `go test ./service -run TestRouteClassifier` | **PASS** (16/16 taxonomy cases) |
| **Stream Commitment Guard** | `relay/stream_commit_test.go` | `go test ./relay -run TestCommitDetectingWriter` | **PASS** (6/6 streaming scenarios) |
| **Route Engine & Settlement** | `service/route_engine_test.go`, `route_billing_test.go` | `go test ./service -run "TestRoute\|TestChannel"` | **PASS** (Entitlement, Model map, Cooldown, Quota) |
| **Staging Full-Stack Scenarios** | `controller/route_staging_test.go` | `go test ./controller -v -run TestRouteStaging` | **PASS** (10/10 end-to-end staging tests) |
| **Full Go Backend Regression** | `model`, `service`, `relay`, `controller` | `go test ./model ./service ./relay ./controller -count=1` | **PASS** (100% pass across all packages) |
| **Race Detector** | `service`, `relay`, `controller` | `go test -race ./service ./relay ./controller -run "TestRoute\|TestCommit"` | **PASS** (Zero data races detected) |
| **Go Vet Static Analysis** | Entire repository | `go vet ./...` | **PASS** (Zero warnings) |
| **Web Admin UI** | `web` | `bun run typecheck`, `vitest run auto-group-form.test.ts` | **PASS** (Zero TS errors, 9/9 unit tests pass) |
| **Mobile Application** | `LumenFlow` | `flutter analyze`, `flutter test`, `flutter build apk --debug` | **PASS** (No issues, 117/117 tests pass, APK built) |

---

## 3. Staging Validation Scenarios Verified

### 3.1 Adversarial Network Failure Validation (`TestRouteStaging_DenialOfWallet_AdversarialScenarios`)
Simulated upstream TCP resets, read timeouts, and mid-stream EOFs:
- **TCP Reset after request transmission**: Categorized as `AMBIGUOUS_EXECUTION_FAILURE` (`AllowFallback: false`). No fallback triggered; request aborted to protect wallet.
- **Read Timeout awaiting headers**: Categorized as `AMBIGUOUS_EXECUTION_FAILURE` (`AllowFallback: false`). Protects against slow-responding models already executing.
- **Unexpected EOF during chunked transfer**: Categorized as `AMBIGUOUS_EXECUTION_FAILURE` (`AllowFallback: false`).
- **Clean 429 Rate Limit**: Categorized as `EXPLICIT_PROVIDER_FAILURE` (`AllowFallback: true`). Moves safely to next channel/route.
- **Connection Refused (pre-dial)**: Categorized as `SAFE_PRE_EXECUTION_FAILURE` (`AllowFallback: true`).

### 3.2 Real HTTP/SSE Stream Commitment (`TestRouteStaging_RealStreamCommitment_FullStack`)
Evaluated `CommitDetectingWriter` integrated into Gin test context:
- Headers sent and initial event flushed (`Flush()`).
- In-flight error raised by upstream mock.
- Writer returns `relay.ErrAlreadyCommitted`.
- Classifier marks request `ClassTerminalCommitted` (`AllowFallback: false`).

### 3.3 Model Mapping & Entitlement Security (`TestRouteStaging_ModelMappingSecurityAndEntitlement`)
- Free user on `"default"` group blocked from Pro routes (`MinUserGroup: "pro"`).
- Pro user allowed.
- Admin / Root accounts granted operational override.
- Model remapping (e.g., `gpt-4` -> `gpt-4o`) validated at runtime.

### 3.4 Billing Exactness & Reservation Eligibility (`TestRouteStaging_BillingExactnessAndReservationEligibility`)
- Token chain containing high-multiplier routes that the user or model is ineligible for (e.g. Pro-only or image-only routes) does **not** inflate `maxMultiplier`.
- Authoritative settlement formula applied to winning route:
  $$\text{SettledQuota} = \lceil \text{BaseQuota} \times \text{ModelRatio} \times \text{UserGroupRatio} \times M_{\text{winning}} \rceil$$
- Difference between conservative reservation and winning settlement refunded immediately.

### 3.5 In-Flight Route Snapshot Immutability (`TestRouteStaging_RouteSnapshotConsistency`)
- Request generates an immutable `RouteSnapshot` at entry.
- Underlying DB and memory cache modified concurrently by admin.
- In-flight request completes strictly using its initial snapshot without state corruption.

### 3.6 Distributed Cooldown & Concurrency (`TestRouteStaging_DistributedCooldownAndPriorityTiers` & `TestRouteStaging_LoadAndConcurrency`)
- Failed channel placed in Redis cooldown (`tora:cooldown:channel:%d`).
- Subsequent requests prioritize healthy channels in top tier.
- Cooldown expires automatically after TTL.
- 100 concurrent requests processed without races or deadlocks.

---

## 4. Multi-Agent Verification Checklist

- [x] **Architecture Invariants**: Route != Entitlement Group strictly respected.
- [x] **Denial-of-Wallet Prevention**: Ambiguous socket drops never fallback.
- [x] **Stream Commitment**: Once bytes reach client, fallback is blocked.
- [x] **Exact Financial Settlement**: Winning route multiplier rules all billing.
- [x] **Legacy Compatibility**: Tokens with `primary_route_id == 0` bypass route engine.
- [x] **Cross-Platform Verification**: Go backend, TypeScript Web UI, and Flutter Mobile pass all tests.
