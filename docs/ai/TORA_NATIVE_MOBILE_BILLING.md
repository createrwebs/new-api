# Tora Native Mobile Billing & Anti-Exploit Enforcement (Billing V2)

**Identifier:** `TORA-BILLING-N3-MOBILE`  
**Date:** 2026-10-08  
**Scope:** Financial security, prepaid execution semantics, anti-refund abuse defense, and zero-credit fair retry for mobile devices.  

---

## 1. The Core Attack Surface in Client-Side Inference

In client-side execution (whether web browser WebGPU or mobile CoreML/NNAPI), computation occurs exclusively on hardware controlled by the user.  
If a system adopts classic optimistic reservation:
```
1. Client requests ticket -> Server holds quota in escrow
2. Client executes open-source model locally on-device
3. Client obtains final high-resolution asset
4. Client terminates network or refuses to call /complete
5. Escrow expires after TTL -> Server automatically refunds quota
```
**Consequence**: The user obtains the computed output for free 100% of the time.

---

## 2. Billing V2: Mandatory `PREPAID_EXECUTION` for Mobile

Under Tora Billing V2, both `NATIVE_BROWSER` and `NATIVE_MOBILE` enforce `PREPAID_EXECUTION`:

1. **Atomic Pre-Activation Deduction**:
   - At `POST /api/studio/native/ticket`, the user's wallet is atomically deducted:
     ```go
     if billingPolicy == model.BillingPolicyPrepaidExecution {
         if err := model.SettleUserWalletPreConsume(requestId); err != nil {
             _ = model.RefundUserWalletPreConsume(requestId)
             return nil, err
         }
         status = model.TicketStatusCharged
         chargedQuota = spec.Quota
         chargedCredits = spec.Credits
         chargedAt = now
     }
     ```
   - The ticket status is created directly as `CHARGED`.
2. **Permanent Quota Commitment**:
   - The charge is finalized **before** the signed cryptographic execution token (`auth_token`) is returned to the mobile app.
   - Blocking or canceling the `/complete` endpoint yields **zero financial gain** for an attacker because quota was already deducted.
3. **No Automatic Refund on Expiry**:
   - The reconciliation sweeper (`ReconcileExpiredNativeTickets`) cleans abandoned `CHARGED` or `FAILED_CLIENT` tickets past TTL by setting status to `EXPIRED` with **zero wallet refund**:
     ```go
     // Clean charged tickets whose fair retry window has fully elapsed (DO NOT REFUND)
     _ = model.UpdateNativeTicketStatus(t.TicketId, model.TicketStatusExpired, map[string]interface{}{
         "error_reason": "prepaid retry window expired without completion",
     })
     ```

---

## 3. Fair Zero-Credit Retry Window (Customer Protection)

To protect legitimate users against real operating system interruptions (e.g. incoming phone call, low memory OS kill, device crash):

- Every `NATIVE_MOBILE` ticket receives an authoritative **30-minute fair retry window** (`retry_until = now + 1800`).
- If an execution fails, the mobile client calls `POST /api/studio/native/retry` with the same `ticket_id`.
- The server verifies `now <= ticket.RetryUntil`, increments `retry_count`, generates a fresh nonce, and issues a new HMAC `auth_token` for **zero additional Tora Credits**.
- User quota is **not** charged a second time.

---

## 4. Cross-Platform Pricing Parity

Mobile native pricing is identical to web native pricing:

| Tool ID | Execution Class | Tora Credits | Internal Quota | USD Value | Billing Policy |
|---|---|---|---|---|---|
| `background-remove` | `NATIVE_MOBILE` | **2 Credits** | 2,000 | $0.0040 | `PREPAID_EXECUTION` |
| `image-upscale-2x` | `NATIVE_MOBILE` | **3 Credits** | 3,000 | $0.0060 | `PREPAID_EXECUTION` |
| `product-pack` | `DETERMINISTIC_SERVER` | **5 Credits** | 5,000 | $0.0100 | `SUCCESS_SETTLEMENT` |

Zero mobile markup. Zero store currency conversion. 100% unified with web.
