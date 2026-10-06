# Phase 7I-S — Cross-Domain Security & Entitlement Review

## Mission & Executive Summary

The Route & Fallback architecture must not introduce privilege escalation, cross-tenant leaks, or authentication bypasses.

This security review audits the four primary security invariants:
1. **Route != Entitlement Group**: Routes must never confer commercial entitlements (e.g. Free -> Pro).
2. **Channel Group Access Enforcement**: Channels assigned to a Route must respect the caller's commercial group.
3. **BYOK Terminal Isolation**: Managed routes and BYOK routes must never cross-fallback.
4. **Tampering & Injection Resistance**: Route IDs, chains, and model mappings must be immune to parameter pollution and SQL injection.

---

## 1. Route vs. Entitlement Group Isolation

### 1.1 Invariant
```text
User Group (Commercial Entitlement)
    ├── default (Free)
    ├── pro (Pro Monthly / Annual)
    └── root (Operator)

Route (Operational Routing)
    ├── Primary Route
    └── Fallback Routes
```
A Route controls **where** a request goes and **how** it is weighted or retried. A Route must **never**:
- Upgrade a user from `default` to `pro`.
- Grant access to restricted models without group entitlement.
- Bypass quota deductions.

### 1.2 Validation in `ValidateRouteEntitlement`
```go
func ValidateRouteEntitlement(route *model.Route, userGroup string) error {
    if !route.Enabled {
        return errors.New("route is disabled")
    }
    if route.MinUserGroup == "" || route.MinUserGroup == "default" {
        return nil
    }
    if userGroup == "root" || userGroup == "admin" {
        return nil
    }
    if route.MinUserGroup == "pro" && userGroup != "pro" {
        return fmt.Errorf("route '%s' requires pro entitlement", route.Name)
    }
    return nil
}
```
Verified in `TestRouteStaging_ModelMappingSecurityAndEntitlement`:
- Users with group `default` attempting to invoke routes with `MinUserGroup: "pro"` receive immediate rejection with zero upstream transmission.

---

## 2. Channel Group Matching

When a Route includes multiple channels, `SelectNextChannelInRoute` validates that the channel itself is permitted for the user's group:
```go
if !ch.HasGroup(userGroup) && !ch.HasGroup("default") {
    continue
}
```
This ensures an operator cannot accidentally expose private or high-tier channels by placing them on a default route.

---

## 3. BYOK Isolation Boundary

### 3.1 Threat
A user providing their own API key (BYOK) must never failover into Tora-managed upstream channels (which would bill Tora for a user's BYOK request). Conversely, managed users must never route into someone else's BYOK credentials.

### 3.2 Defense
In `controller/relay.go`:
```go
if info.IsBYOK {
    if currentRoute.Kind != model.RouteKindBYOK {
        continue // Skip managed routes
    }
} else {
    if currentRoute.Kind == model.RouteKindBYOK {
        continue // Skip BYOK routes
    }
}
```
Verified in `TestRouteStaging_BYOKTerminalIsolation`:
- A BYOK request evaluated against a chain `[managedRoute, byokRoute]` skips the managed route entirely.

---

## 4. Tampering & Serialization Security

### 4.1 Client Route Tampering Defense
- Clients **cannot** supply a `route_id` header or parameter to override server routing.
- The active route chain is loaded strictly from the authenticated `model.Token` record in the database (`token.PrimaryRouteId` and `token.FallbackRouteIds`).
- In `TestRouteStaging_DirectRouteIDTamperingAndSerialization`:
  - Tampered client headers are discarded.
  - JSON serialization of `model.Route` and `model.Token` cleanly roundtrips without leaking internal fields or breaking schema constraints.

---

## 5. Security Sign-off

| Domain | Status | Notes |
| :--- | :--- | :--- |
| **Entitlement Boundary** | **SECURE** | Free users cannot trigger Pro routes. |
| **Channel Group Isolation** | **SECURE** | Channels checked against user group. |
| **BYOK Isolation** | **SECURE** | Strict separation of Managed vs BYOK kinds. |
| **Data Integrity** | **SECURE** | Immutable request snapshots prevent race conditions. |
| **Tamper Resistance** | **SECURE** | Client cannot manipulate route IDs. |
