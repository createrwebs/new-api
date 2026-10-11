# Tora AI — Existing Stripe / TopUp / PromptPay Reuse Audit

## Executive Summary

**DECISION: REUSE EXISTING STRIPE STACK**

A rigorous, line-by-line audit of the New-API payment codebase (`/Users/noppanan/new-api`, branch `feat/formobile`) confirms that the existing **Stripe / TopUp stack is production-ready, idempotent, cryptographically verified, and fully reusable**.

**NO second Stripe engine or new payment backend is required.**

Key findings:
1. **Webhook Authenticity**: Strictly verified using the official Stripe Go SDK (`webhook.ConstructEventWithOptions`) against `setting.StripeWebhookSecret`. Unsigned or forged payloads are rejected immediately with HTTP 400.
2. **Double-Layer Idempotency & Locking**: Protected by both an in-memory reference-counted order lock (`LockOrder(tradeNo)`) and a transactional SQL row lock (`lockForUpdate(tx)` / `SELECT ... FOR UPDATE`). Duplicate or concurrent webhook callbacks exit cleanly with `alreadyDone = true` without double-crediting.
3. **Atomic Quota Settlement**: `model.creditTopUpQuota` enforces the wallet quota ceiling (`WHERE quota <= maxCurrentQuota`) atomically inside the SQL `UPDATE` statement, preventing race conditions across concurrent nodes.
4. **PromptPay Support**: Supported natively through **Stripe Dynamic / Automatic Payment Methods**. Because `controller/topup_stripe.go` does not restrict `PaymentMethodTypes`, Stripe Checkout automatically presents PromptPay QR codes to Thai users whenever PromptPay is enabled in the Stripe Dashboard and the price currency is `THB`.
5. **Clear Trust Boundaries**: Webhooks remain strictly `SERVER_ONLY`. The Flutter mobile client never calls webhook routes and never treats checkout redirects or QR display as settlement. Settlement is authoritative only upon verified webhook delivery.

---

## 1. Codebase Architecture Tracing

### 1.1 Route & Controller Mapping

| Route | HTTP Method | Controller Handler | Purpose | Auth Requirement |
| :--- | :---: | :--- | :--- | :--- |
| `/api/user/stripe/amount` | POST | `controller.RequestStripeAmount` | Calculates exact billing charge for a given quota count | Authenticated User |
| `/api/user/stripe/pay` | POST | `controller.RequestStripePay` | Creates Stripe Checkout Session; returns `pay_link` | Authenticated User |
| `/api/subscription/stripe/pay` | POST | `controller.SubscriptionRequestStripePay` | Creates recurring Stripe Checkout Session for subscription | Authenticated User |
| `/api/stripe/webhook` | POST | `controller.StripeWebhook` | Authoritative settlement callback from Stripe | Public / Signed via `Stripe-Signature` |

### 1.2 Checkout Session Creation (`controller/topup_stripe.go`)
- **Reference Identifier**: Formats unique transaction `referenceId`:
  ```go
  reference := fmt.Sprintf("new-api-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
  referenceId := "ref_" + common.Sha1([]byte(reference))
  ```
- **Redirect Security**: Validates `SuccessURL` and `CancelURL` against trusted domain allowlists using `common.ValidateRedirectURL()`.
- **TopUp Ledger Insertion**: Creates a pending database record (`TopUp` row) with `Status: common.TopUpStatusPending` before returning the payment link.
- **Session Parameters**:
  ```go
  params := &stripe.CheckoutSessionParams{
      ClientReferenceID: stripe.String(referenceId),
      SuccessURL:        stripe.String(successURL),
      CancelURL:         stripe.String(cancelURL),
      LineItems: []*stripe.CheckoutSessionLineItemParams{
          {
              Price:    stripe.String(setting.StripePriceId),
              Quantity: stripe.Int64(amount),
          },
      },
      Mode:                stripe.String(string(stripe.CheckoutSessionModePayment)),
      AllowPromotionCodes: stripe.Bool(setting.StripePromotionCodesEnabled),
  }
  ```

---

## 2. Webhook Authenticity & Cryptographic Verification

In `controller/topup_stripe.go` (`StripeWebhook`):
```go
signature := c.GetHeader("Stripe-Signature")
event, err := webhook.ConstructEventWithOptions(payload, signature, setting.StripeWebhookSecret, webhook.ConstructEventOptions{
    IgnoreAPIVersionMismatch: true,
})
if err != nil {
    logger.LogWarn(ctx, fmt.Sprintf("Stripe webhook 验签失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
    c.AbortWithStatus(http.StatusBadRequest)
    return
}
```
- Forged payloads lacking valid HMAC-SHA256 signatures generated with the shared `StripeWebhookSecret` are rejected with HTTP 400.
- Disabled webhooks return HTTP 403 `Forbidden` before reading payloads.

---

## 3. Financial Integrity & Settlement Idempotency

### 3.1 Concurrency Protection
Settlement is guarded by two sequential locking mechanisms:
1. **In-Memory Order Lock (`LockOrder(referenceId)`)**: A reference-counted mutex map preventing goroutine races within a single process.
2. **Database Row Lock (`lockForUpdate(tx)`)**: Issues a database-level `SELECT ... FOR UPDATE` row lock on `TopUp` or `SubscriptionOrder`. This guarantees safety across multi-instance cluster deployments.

### 3.2 Idempotent Replay Handling
In `model/topup.go` (`Recharge`):
```go
if topUp.Status == common.TopUpStatusSuccess {
    alreadyDone = true
    return nil
}
```
If Stripe retries a webhook or concurrent deliveries arrive for the same event, the second invocation immediately detects `Status == common.TopUpStatusSuccess`, sets `alreadyDone = true`, and returns `nil`. No extra quota is credited.

### 3.3 Atomic Wallet Quota Ceiling
To prevent wallet balance manipulation or overflow, `creditTopUpQuota` performs the ceiling check and increment in a single atomic SQL statement:
```go
result := tx.Model(&User{}).
    Where("id = ? AND quota <= ?", userId, maxCurrentQuota).
    Updates(updateFields)
```
If the user's current quota exceeds `maxCurrentQuota`, zero rows are updated, and the transaction rolls back with `ErrTopUpQuotaLimitExceeded`.

---

## 4. PromptPay Capability Analysis

### 4.1 Stripe Mechanism
Per official Stripe Documentation:
- PromptPay is a single-use payment method based on dynamic Bank QR codes in Thailand.
- Stripe provides PromptPay via **Dynamic Payment Methods** on Checkout Sessions.
- Because `controller/topup_stripe.go` does not specify `payment_method_types`, Stripe automatically determines the eligible payment methods to display based on the merchant's Dashboard configuration.

### 4.2 Prerequisites for PromptPay
1. **Stripe Dashboard**: PromptPay must be enabled in the Stripe account settings (`https://dashboard.stripe.com/settings/payment_methods`).
2. **Currency Requirement**: PromptPay requires presentment and settlement in Thai Baht (`THB`). If `setting.StripePriceId` points to a price in `THB`, PromptPay will be displayed to eligible customers. If the price is in `USD`, Stripe displays USD-compatible methods (e.g. Credit/Debit Cards).
3. **One-Time vs Recurring**: PromptPay does not support recurring subscriptions (no automated recurring pull). It is fully supported for one-time top-ups (`CheckoutSessionModePayment`). For recurring subscriptions, customers use credit/debit cards.

---

## 5. Trust Boundary & Mobile Integration Architecture

### 5.1 Trust Model
```text
┌─────────────────────────────────────────────────────────────┐
│                       Flutter Client                        │
│                                                             │
│ 1. User selects top-up amount                               │
│ 2. Calls POST /api/user/stripe/pay                          │
│ 3. Receives {"pay_link": "https://checkout.stripe.com/..."} │
│ 4. Opens pay_link in System Browser via url_launcher        │
│ 5. Returns to app after payment                             │
│ 6. Refreshes quota via GET /api/user/self                   │
└──────────────────────────────┬──────────────────────────────┘
                               │
                       HTTPS (Auth Bearer)
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                      Tora / New-API                         │
│                                                             │
│ • Validates request quota caps                              │
│ • Inserts pending TopUp record (trade_no)                   │
│ • Generates Stripe Checkout Session                         │
│                                                             │
│ [WEBHOOK TRUST BOUNDARY]                                    │
│ • POST /api/stripe/webhook (SERVER ONLY)                    │
│ • Validates Stripe-Signature HMAC-SHA256                   │
│ • Atomically credits quota & marks TopUp status success     │
└──────────────────────────────▲──────────────────────────────┘
                               │
                      Stripe Event Delivery
                               │
┌──────────────────────────────┴──────────────────────────────┐
│                       Stripe Platform                       │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 Client-Side Invariants
1. **Never Call Webhooks**: `/api/stripe/webhook` is excluded from client APIs.
2. **No Cleartext Credentials in URLs**: `pay_link` is generated by Stripe; no Tora API keys or JWT session tokens are placed in query parameters.
3. **No Local Proof Assumption**: Returning to the application or viewing a payment success page is NEVER assumed to be settlement. Entitlement is only granted when the backend verifies the webhook and credits the account.

---

## 6. Audit Conclusion

The existing New-API Stripe / TopUp implementation meets all security, concurrency, and financial integrity requirements.

**Action Plan**:
- **Backend**: Preserve existing implementation without modification.
- **Mobile**: Integrate `requestStripeCheckout` into `ToraApiClient` using `url_launcher` for external browser execution.
