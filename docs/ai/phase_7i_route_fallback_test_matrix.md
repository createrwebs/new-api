# Phase 7I — Route & Fallback Test Matrix (Agent F)

## Executive Summary

This document specifies the verification matrix, automated test coverage, and execution results for Phase 7I (First-Class Routes & Ordered Fallback System) across Go backend modules and Flutter mobile clients.

---

## 1. Test Suite Coverage & Results

| Test Category | Target File / Package | Tests Executed | Status | Duration |
| :--- | :--- | :--- | :--- | :--- |
| **Route Model & Chains** | `model/route_test.go` | 4 test suites | **PASS** | 0.05s |
| **Route Classifier** | `service/route_classifier_test.go` | 6 test suites | **PASS** | 0.02s |
| **Route Engine & Cache** | `service/route_engine_test.go` | 5 test suites | **PASS** | 0.15s |
| **Route Billing Pipeline** | `service/route_billing_test.go` | 3 test suites | **PASS** | 0.01s |
| **Streaming Commit Guard** | `relay/stream_commit_test.go` | 4 test suites | **PASS** | 0.02s |
| **Route Controller & Relay** | `controller/route_relay_test.go` | 3 test suites | **PASS** | 0.99s |
| **Package: Model** | `github.com/QuantumNous/new-api/model` | Full package suite | **PASS** | 10.39s |
| **Package: Service** | `github.com/QuantumNous/new-api/service` | Full package suite | **PASS** | 2.72s |
| **Package: Relay** | `github.com/QuantumNous/new-api/relay` | Full package suite | **PASS** | 2.10s |
| **Package: Controller** | `github.com/QuantumNous/new-api/controller` | Full package suite | **PASS** | 33.37s |
| **Static Code Analysis** | `go vet ./...` | All packages | **PASS** | 8.12s |
| **Web UI Typecheck** | `web` (`bun run typecheck`) | TypeScript schemas | **PASS** | 3.20s |
| **Mobile Client Analysis** | `LumenFlow` (`flutter analyze`) | Dart analyzer | **PASS** | 3.30s |
| **Mobile Client Tests** | `LumenFlow` (`flutter test`) | 117 unit/widget tests | **PASS** | 6.00s |

---

## 2. Detailed Test Cases

### 2.1 Route Error Classification (`service/route_classifier_test.go`)
- **TC-RC-01: Upstream 429 Rate Limit**: Returns `ClassFallbackEligible`, `AllowFallback: true`.
- **TC-RC-02: Upstream 503 Service Unavailable**: Returns `ClassFallbackEligible`, `AllowFallback: true`.
- **TC-RC-03: Client 400 Bad Request**: Returns `ClassTerminalClient`, `AllowFallback: false`.
- **TC-RC-04: Client 404 Model Not Found**: Returns `ClassTerminalClient`, `AllowFallback: false`.
- **TC-RC-05: Post-Commit Stream Failure**: Even on 429 or 503, returns `ClassTerminalCommitted`, `AllowFallback: false`.
- **TC-RC-06: Downstream Client Cancellation**: Returns `ClassClientCancelled`, `AllowFallback: false`.

### 2.2 Route Selection & Prioritization (`service/route_engine_test.go`)
- **TC-SE-01: Entitlement Enforcement**: Verifies that standard users cannot invoke Pro routes; Pro users can invoke Pro routes.
- **TC-SE-02: Model Mapping Resolution**: Verifies that virtual model names correctly translate to upstream target model IDs.
- **TC-SE-03: Channel Cooldown Window**: Verifies that a channel placed on 15-second cooldown is skipped during route selection until cooldown expires.
- **TC-SE-04: Duplicate Attempt Prevention**: Verifies that already-attempted channels are excluded from subsequent selection attempts within the same request.

### 2.3 Streaming Response Boundary (`relay/stream_commit_test.go`)
- **TC-SC-01: Pre-Commit Staging**: Verifies that headers, status codes, and bytes are buffered in memory and not sent to the underlying writer before `Commit()`.
- **TC-SC-02: Safe Reset on Fallback**: Verifies that `Reset()` clears the buffer and headers, allowing a clean retry attempt.
- **TC-SC-03: Write-Through on Commit**: Verifies that calling `Commit()` immediately flushes all buffered content to the underlying connection and switches to pass-through mode.
- **TC-SC-04: Rejection of Post-Commit Reset**: Verifies that calling `Reset()` on an already committed writer returns `ErrAlreadyCommitted`.

### 2.4 Billing & Quota Settlement (`service/route_billing_test.go`)
- **TC-BL-01: Conservative Pre-Consumption Reservation**: Verifies that pre-consumption reservation multiplies by the maximum multiplier present across the route chain.
- **TC-BL-02: Winning Route Quota Settlement**: Verifies that the final quota charged matches the winning route's multiplier: $\lceil \text{BaseQuota} \times \text{ModelRatio} \times \text{UserGroupRatio} \times M_{\text{winning}} \rceil$.
- **TC-BL-03: Unused Reservation Refund**: Verifies that the excess quota reserved during pre-consumption is refunded to the user's wallet when a lower-multiplier route wins.

### 2.5 API Endpoints & Administrative Safety (`controller/route_relay_test.go`)
- **TC-CR-01: Route CRUD Operations**: Verifies creation, retrieval, updating, and available-routes listing for administrators and users.
- **TC-CR-02: Referential Deletion Protection**: Verifies that deleting a route referenced by active API keys is rejected with HTTP 409 Conflict.
- **TC-CR-03: Token Route Chain Parsing**: Verifies that tokens with primary and fallback routes resolve deterministically and legacy tokens (`primary_route_id == 0`) remain non-routed.

---

## 3. Test Verification Sign-Off

All tests across backend, frontend types, and mobile client passed without a single failure or regression.

Test Sign-Off: **PASSED (Agent F)**
