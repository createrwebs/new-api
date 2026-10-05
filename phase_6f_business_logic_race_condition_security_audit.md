# Phase 6F — Business Logic, Race Condition & State Transition Security Audit

**Execution Date**: 2026-10-05  
**Audit Target**: SaaSCover / New-API Backend (`d:\LumenFlow\new-api`, branch `feat/formobile`)  
**Scope Mode**: STRICTLY READ-ONLY SECURITY AUDIT  
**Preceding Security Status**:  
- Phase 5C/5D (BYOK SSRF, DNS Rebinding, Credential Isolation): **PASS**  
- Phase 6B (Authentication, Authorization, StepUp, Token Security): **PASS**  
- Phase 6C (Billing, Balance, Quota, Pre-Consumption Ledger): **PASS**  
- Phase 6D (API Gateway Abuse, Concurrency Limiting, Timeouts): **PASS**  
- Phase 6E/6E-R (Data Exposure, Secret Leakage, Access Logs, Headers): **PASS**  

---

## 1. Executive Summary & Verdict

This security audit constitutes **Phase 6F** of the New API / SaaSCover security assurance roadmap. It provides an exhaustive, implementation-level examination of **business logic integrity, object ownership & privilege boundaries, database transaction isolation, state machine transitions, concurrent race conditions (TOCTOU), double-spending vectors, webhook replay/idempotency, and cache consistency**.

The inspection audited all state mutations across TopUp orders (Epay, Stripe, Creem, Waffo, Waffo Pancake), Subscriptions (purchase, renewal, reset, downgrade, cancellation), Quota & Balance Ledgers (pre-consumption reservations, wallet balances, token balances), Task Lifecycles (Midjourney, Suno, Video, asynchronous plugins), User Accounts (profile updates, role management, affiliate quota transfers, user deletion), and Channel Routing.

### Final Audit Verdict

```text
================================================================================
FINAL VERDICT: PASS WITH FINDINGS
================================================================================
Confirmed Vulnerabilities / Logic Flaws: 6
  - HIGH:         2
  - MEDIUM-HIGH:  1
  - MEDIUM:       2
  - LOW-MEDIUM:   1
Critical RCE / Direct Unauthorized Balance Injection: 0
Regression Against Preceding Phases (5C/5D, 6B, 6C, 6D, 6E): 0 (NONE)
================================================================================
```

### Summary of Confirmed Findings

| Finding ID | Severity | Impact Category | Primary File(s) | Description |
| :--- | :--- | :--- | :--- | :--- |
| **6F-01** | **HIGH** | Double-Spend / ACID Failure | `model/subscription.go` | Nested transaction boundary break in `RefundSubscriptionPreConsume` causes double refund on retry / outer rollback |
| **6F-02** | **HIGH** | Distributed State Corruption | `controller/topup_stripe.go` | Non-transactional race in Stripe delayed failure (`sessionAsyncPaymentFailed`) clobbers settled TopUp orders |
| **6F-03** | **MEDIUM-HIGH** | Quota Desync / Struct Clashing | `model/user.go`, `controller/user.go` | `TransferAffQuotaToQuota` omits Redis cache sync, bypasses wallet ceiling, and risks GORM struct overwrite |
| **6F-04** | **MEDIUM** | Partial Failure Quota Loss | `model/wallet_pre_consume.go` | `RefundUserWalletPreConsume` commits status `"refunded"` before crediting quota, leaving orphaned balances on crash |
| **6F-05** | **MEDIUM** | Resource Imbalance / Starvation | `model/channel.go` | `GetChannelKey` polling cursor freezes at index 0 when `MemoryCacheEnabled = true` due to uncommitted cache update |
| **6F-06** | **LOW-MEDIUM** | Webhook Idempotency Defect | `model/topup.go` | Stripe (`Recharge`) and Creem (`RechargeCreem`) return error on redelivered webhook instead of clean idempotent success |

---

## 2. Scope & Target Architecture Reconstruction

The audit traced all business logic flows through the complete backend architecture:

```text
                                  [ Client / Webhook ]
                                            │
                                            ▼
                                   [ HTTP Middleware ]
              (UserAuth, AdminAuth, RootAuth, RateLimit, CORS, StepUp)
                                            │
                                            ▼
                                    [ Controllers ]
   (TopUp, Subscription, User, Token, UserProvider, Channel, Task, Checkin)
                                            │
                                            ▼
                               [ Service / Domain Layer ]
      (BillingSession, FundingSource, TaskBilling, Polling, ProviderTester)
                                            │
                      ┌─────────────────────┴─────────────────────┐
                      ▼                                           ▼
          [ Redis Memory Cache ]                         [ Main Database ]
   (UserCache, TokenCache, Lua scripts)              (MySQL / PostgreSQL / SQLite)
   - Atomic Lua token reservations                   - ACID Transactions
   - Atomic Lua wallet reservations                  - FOR UPDATE Row Locks
   - Auth version & tombstone fences                 - Conditional CAS Queries
```

The audit verified every point of interaction between:
1. HTTP request parameter binding (`c.ShouldBindJSON`, `common.DecodeJson`) and mass assignment protections.
2. In-memory locks (`LockOrder`, `channelPollingLocks`, `channelStatusLock`) versus multi-node distributed database locks.
3. Multi-statement database operations and transaction boundaries (`tx *gorm.DB`).
4. Read-cache, write-cache, and database synchronization points (`syncCreditUserQuotaCache`, `cacheIncrUserQuota`, `updateUserCache`).

---

## 3. State Machine Reconstruction

### 3.1 TopUp & Payment Order Lifecycle

```text
                    ┌─────────────────────────┐
                    │      Created/Pending     │
                    └────────────┬────────────┘
                                 │
           ┌─────────────────────┼─────────────────────┐
           │ (Webhook Success)   │ (Webhook Failure)   │ (Session Expired)
           ▼                     ▼                     ▼
┌─────────────────────┐┌─────────────────────┐┌─────────────────────┐
│       Success       ││        Failed       ││       Expired       │
│  (Quota Credited)   ││  (Zero Balance Mod) ││  (Zero Balance Mod) │
└─────────────────────┘└─────────────────────┘└─────────────────────┘
```
- **Terminal States**: `Success`, `Failed`, `Expired`.
- **Invariants**: Once `Success`, an order MUST never transition to `Failed` or `Pending`. Quota must be credited exactly once.

### 3.2 Subscription Order & UserSubscription Lifecycle

```text
                    ┌─────────────────────────┐
                    │  Subscription Order     │
                    │       (Pending)         │
                    └────────────┬────────────┘
                                 │ (CompleteSubscriptionOrder)
                                 ▼
                    ┌─────────────────────────┐
                    │    UserSubscription     │
                    │        (Active)         │
                    └──────┬────────────┬─────┘
                           │            │
            (EndTime passed)            │ (Admin Invalidate)
                           ▼            ▼
                    ┌────────────┐┌────────────┐
                    │  Expired   ││ Cancelled  │
                    └────────────┘└────────────┘
                           │            │
                           └─────┬──────┘
                                 │ (downgradeUserGroupForSubscriptionTx)
                                 ▼
                    ┌─────────────────────────┐
                    │   Revert User Group     │
                    └─────────────────────────┘
```
- **Invariants**: `MaxPurchasePerUser` must be serialized per user. Downgrade must only revert when no other active upgraded subscription exists.

### 3.3 Task Lifecycle (Midjourney, Suno, Video, Plugins)

```text
    ┌──────────┐      CAS Win       ┌─────────────┐
    │ Pending  ├───────────────────►│ InProgress  │
    └──────────┘                    └──────┬──────┘
                                           │
                        ┌──────────────────┴──────────────────┐
                        │ CAS Win                             │ CAS Win
                        ▼                                     ▼
                 ┌──────────────┐                      ┌──────────────┐
                 │   Success    │                      │   Failure    │
                 │(Settle Final)│                      │(Refund Quota)│
                 └──────────────┘                      └──────────────┘
```
- **Invariants**: Status transitions guarded by `UpdateWithStatus(fromStatus)` (CAS). Only the CAS winner may finalize billing or refund quota.

---

## 4. Business Logic & Authorization Matrix

| Entity | Operation | Route / Function | Authorization Guard | Ownership Enforced? | Result |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **User** | Self Profile Update | `PUT /api/user/self` | `UserAuth()` | Bound to `c.GetInt("id")` | **PASS** |
| **User** | Self Password Change | `PUT /api/user/self` | `UserAuth()` + StepUp Proof | Bound to `c.GetInt("id")` | **PASS** |
| **User** | Self Account Delete | `DELETE /api/user/self`| `UserAuth()` | Bound to `c.GetInt("id")` | **PASS** |
| **User** | Admin Manage User | `POST /api/user/manage` | `AdminAuth()` + Root Proof | Role hierarchy enforced | **PASS** |
| **Token** | Create Token | `POST /api/token` | `UserAuth()` | Force `UserId = c.GetInt("id")` | **PASS** |
| **Token** | Get/Delete Token | `GET/DELETE /api/token/:id`| `UserAuth()` | `GetTokenByIds(id, userId)` | **PASS** |
| **Token** | Reveal Plaintext Key | `POST /api/token/:id/key` | `UserAuth()` + RateLimit | `GetTokenByIds(id, userId)` | **PASS** |
| **BYOK** | Create Provider | `POST /api/user/providers` | `UserAuth()` | Force `UserId = c.GetInt("id")` | **PASS** |
| **BYOK** | Update/Delete Provider| `PUT/DELETE /api/user/providers/:id` | `UserAuth()` | `GetUserProviderByID(userID, id)` | **PASS** |
| **BYOK** | Test Provider | `POST /api/user/providers/:id/test` | `UserAuth()` | `GetUserProviderByID(userID, id)` | **PASS** |
| **Task** | Query Task | `GET /api/task/:key` | `UserAuth()` | `GetByTaskId(userId, key)` | **PASS** |
| **Artifact**| Access Artifact | `GET /api/task/:key/artifacts` | Capability Token HMAC | `VerifyTaskArtifactAccess` | **PASS** |
| **Channel** | Admin Channel CRUD | `POST/PUT/DELETE /api/channel` | `AdminAuth()` + RBAC | RBAC `authz.Channel*` | **PASS** |
| **Channel** | Reveal Channel Key | `POST /api/channel/:id/key` | `RootAuth()` + StepUp | Restricted to Root role | **PASS** |

---

## 5. In-Depth Findings Breakdown

### Finding 6F-01: Broken Transaction Boundary in Subscription Pre-Consume Refund (`RefundSubscriptionPreConsume`) Leading to Double Refund / Connection Starvation

- **Vulnerability Type**: Transaction Scope Leak / Non-Atomic Nested Transaction / Double Spend
- **Severity**: **HIGH**
- **Affected File**: `model/subscription.go`, lines 1406–1424
- **Call Chain**:
  `BillingSession.Close()` / `BillingSession.Refund()`  
  `└── SubscriptionFunding.Refund()`  
  `    └── refundWithRetry(...)`  
  `        └── model.RefundSubscriptionPreConsume(requestId)`  
  `            └── DB.Transaction(func(tx *gorm.DB) { ... })`  
  `                └── model.PostConsumeUserSubscriptionDelta(...)`  
  `                    └── DB.Transaction(func(tx *gorm.DB) { ... })`  *(Independent DB connection!)*

#### Code Evidence

```go
// model/subscription.go:1406
func RefundSubscriptionPreConsume(requestId string) error {
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var record SubscriptionPreConsumeRecord
		if err := lockForUpdate(tx).
			Where("request_id = ?", requestId).First(&record).Error; err != nil {
			return err
		}
		if record.Status == "refunded" {
			return nil
		}
		if record.PreConsumed <= 0 {
			record.Status = "refunded"
			return tx.Save(&record).Error
		}
		// BUG: Calls PostConsumeUserSubscriptionDelta which opens a SEPARATE DB.Transaction
		// on the root connection pool rather than using tx!
		if err := PostConsumeUserSubscriptionDelta(record.UserSubscriptionId, -record.PreConsumed); err != nil {
			return err
		}
		record.Status = "refunded"
		return tx.Save(&record).Error
	})
}
```

```go
// model/subscription.go:1510
func PostConsumeUserSubscriptionDelta(userSubscriptionId int, delta int64) error {
	if userSubscriptionId <= 0 {
		return errors.New("invalid userSubscriptionId")
	}
	if delta == 0 {
		return nil
	}
	// Independent transaction on DB root:
	return DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).
			Where("id = ?", userSubscriptionId).
			First(&sub).Error; err != nil {
			return err
		}
		newUsed := max(sub.AmountUsed+delta, 0)
		if sub.AmountTotal > 0 && newUsed > sub.AmountTotal {
			return fmt.Errorf("subscription used exceeds total, used=%d total=%d", newUsed, sub.AmountTotal)
		}
		sub.AmountUsed = newUsed
		return tx.Save(&sub).Error
	})
}
```

#### Attack Sequence & Impact
1. `RefundSubscriptionPreConsume` begins an outer transaction `tx1` and locks `SubscriptionPreConsumeRecord`.
2. It calls `PostConsumeUserSubscriptionDelta`, which borrows connection 2 from the pool and starts `tx2`.
3. `tx2` executes `sub.AmountUsed = max(sub.AmountUsed - delta, 0)` and commits immediately to the database.
4. Back in `tx1`, if `tx.Save(&record)` fails (e.g., transient network glitch, database lock timeout, application crash, or serialization failure), `tx1` rolls back!
5. As a result, `record.Status` remains `"consumed"` in the database.
6. The caller (`service/funding_source.go:135` via `refundWithRetry`) catches the error and retries `RefundSubscriptionPreConsume` up to 3 times.
7. On attempt 2, `record.Status` is still `"consumed"`. `PostConsumeUserSubscriptionDelta` runs again, deducting `record.PreConsumed` a second time!
8. **Financial Impact**: A user's used subscription quota is discounted multiple times, granting unearned subscription credit.
9. **Deadlock / Starvation Impact**: In SQLite or connection-constrained pools, borrowing connection 2 while connection 1 holds a lock triggers immediate database lockouts.

---

### Finding 6F-02: Non-Transactional Distributed Race in Stripe Delayed Payment Failure (`sessionAsyncPaymentFailed`) Overwriting Settled TopUps

- **Vulnerability Type**: Distributed TOCTOU / Non-Atomic State Mutation / Webhook Collision
- **Severity**: **HIGH**
- **Affected File**: `controller/topup_stripe.go`, lines 239–273
- **Contrast With**: `controller/topup_stripe.go:333` (which uses `model.UpdatePendingTopUpStatus`)

#### Code Evidence

```go
// controller/topup_stripe.go:239
func sessionAsyncPaymentFailed(ctx context.Context, event stripe.Event, callerIp string) {
	referenceId := event.GetObjectValue("client_reference_id")
	logger.LogWarn(ctx, fmt.Sprintf("Stripe 异步支付失败 trade_no=%s client_ip=%s", referenceId, callerIp))

	if len(referenceId) == 0 {
		return
	}

	LockOrder(referenceId)
	defer UnlockOrder(referenceId)

	// BUG: Read-then-write without database transaction or row lock
	topUp := model.GetTopUpByTradeNo(referenceId)
	if topUp == nil {
		return
	}

	if topUp.PaymentProvider != model.PaymentProviderStripe {
		return
	}

	if topUp.Status != common.TopUpStatusPending {
		return
	}

	topUp.Status = common.TopUpStatusFailed
	if err := topUp.Update(); err != nil { // GORM Save(&topUp) outside transaction
		return
	}
	logger.LogInfo(ctx, fmt.Sprintf("Stripe 充值订单已标记为失败 trade_no=%s client_ip=%s", referenceId, callerIp))
}
```

Compare this with how `sessionExpired` correctly delegates to `UpdatePendingTopUpStatus`:
```go
// controller/topup_stripe.go:333
err := model.UpdatePendingTopUpStatus(referenceId, model.PaymentProviderStripe, common.TopUpStatusExpired)
```

And examine `model.UpdatePendingTopUpStatus`:
```go
// model/topup.go:155
return DB.Transaction(func(tx *gorm.DB) error {
	topUp := &TopUp{}
	if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
		return ErrTopUpNotFound
	}
	if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
		return ErrPaymentMethodMismatch
	}
	if topUp.Status != common.TopUpStatusPending {
		return ErrTopUpStatusInvalid
	}
	topUp.Status = targetStatus
	return tx.Save(topUp).Error
})
```

#### Attack Sequence & Impact
1. `LockOrder(referenceId)` is an in-memory process-local mutex (`sync.Map`). In any multi-instance or Kubernetes deployment, different pods do not share `orderLocks`.
2. Stripe sends both `checkout.session.completed` (or delayed `sessionAsyncPaymentSucceeded`) and a misrouted/delayed `checkout_session.async_payment_failed`.
3. Node A receives `sessionCompleted` and begins `model.Recharge` (which locks the row via `SELECT ... FOR UPDATE`).
4. Concurrently on Node B, `sessionAsyncPaymentFailed` reads `model.GetTopUpByTradeNo` before Node A commits. It observes `topUp.Status == "pending"`.
5. Node A completes: sets `topUp.Status = "success"`, credits user wallet quota, and commits.
6. Node B now executes: `topUp.Status = common.TopUpStatusFailed; topUp.Update()`. It overwrites the database row with `status = "failed"`.
7. **Business Impact**:
   - The order shows as `"failed"` in audit logs and management consoles, while the user was actually credited.
   - Or in the reverse ordering: Node B marks `"failed"` without atomic verification, and Node A's legitimate `Recharge` sees `Status == "failed"`, rejecting the top-up (`充值订单状态错误`). The user is charged by Stripe but receives NO quota.

---

### Finding 6F-03: Wallet Quota Cache Desynchronization and Struct Overwrite Hazard in Affiliate Quota Transfer (`TransferAffQuotaToQuota`)

- **Vulnerability Type**: Cache Inconsistency / Unbounded Wallet Quota / GORM Field Overwrite
- **Severity**: **MEDIUM-HIGH**
- **Affected File**: `model/user.go`, lines 587–622
- **Call Chain**:
  `POST /api/user/aff_transfer`  
  `└── controller.TransferAffQuota`  
  `    └── user.TransferAffQuotaToQuota(tran.Quota)`

#### Code Evidence

```go
// model/user.go:587
func (user *User) TransferAffQuotaToQuota(quota int) error {
	if float64(quota) < common.QuotaPerUnit {
		return fmt.Errorf("转移额度最小为%s！", logger.LogQuota(common.QuotaFromFloat(common.QuotaPerUnit)))
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	err := lockForUpdate(tx).First(user, user.Id).Error
	if err != nil {
		return err
	}

	if user.AffQuota < quota {
		return errors.New("邀请额度不足！")
	}

	user.AffQuota -= quota
	user.Quota += quota

	// BUG 1: tx.Save(user) writes all struct fields back to DB,
	// risking clobbering asynchronous changes to other user attributes.
	if err := tx.Save(user).Error; err != nil {
		return err
	}

	// BUG 2: Omits common.ValidateWalletQuota(user.Quota) and ValidateTopUpQuotaCapacity.
	// Can overflow MaxWalletQuota or cause int wrapping.

	if err := tx.Commit().Error; err != nil {
		return err
	}

	// BUG 3: syncCreditUserQuotaCache(user.Id, quota, "aff transfer") is NEVER called!
	return nil
}
```

Compare with `model/user_cache.go:155`:
```go
// syncCreditUserQuotaCache 在授信事务（充值/兑换等）提交后同步把增量补进缓存
// 余额。预扣以缓存值为准（存在期间），授信不能绕过它，否则新到账的额度在
// 缓存过期前不可用；缓存未命中无需处理，下次读取会从已提交的数据库余额水合。
func syncCreditUserQuotaCache(userId int, quota int, operation string) {
	if quota <= 0 {
		return
	}
	if err := cacheIncrUserQuota(userId, int64(quota)); err != nil {
		common.SysLog(fmt.Sprintf("failed to sync %s credit to user quota cache: %s", operation, err.Error()))
	}
}
```

#### Attack Sequence & Impact
1. User transfers 500,000 aff quota to wallet quota via `/api/user/aff_transfer`.
2. The DB row is updated, but Redis `user:<id>` cache is NOT incremented.
3. User immediately attempts to send AI relay requests.
4. Relay calls `TryReserveUserQuota`, which evaluates the Redis cache `Quota`.
5. The request is rejected as having insufficient quota (`403 Forbidden` / `insufficient_quota`) because the newly transferred affiliate quota is completely invisible to Redis until the cache key TTL expires (up to 60+ seconds).
6. Furthermore, `tx.Save(user)` writes the whole struct back, creating race conditions against concurrent login timestamp updates (`last_login_at`) or batch usage reconciliations.

---

### Finding 6F-04: Partial Failure Quota Leak in Wallet Pre-Consume Refund (`RefundUserWalletPreConsume`) Due to Non-Atomic State Commit

- **Vulnerability Type**: Non-Atomic Multi-Phase State Commit / Permanent Credit Deprivation
- **Severity**: **MEDIUM**
- **Affected File**: `model/wallet_pre_consume.go`, lines 100–135

#### Code Evidence

```go
// model/wallet_pre_consume.go:100
func RefundUserWalletPreConsume(requestId string) error {
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is empty")
	}
	ensureWalletPreConsumeTable(DB)
	var userId int
	var preConsumed int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var records []WalletPreConsumeRecord
		if err := lockForUpdate(tx).
			Where("request_id = ?", requestId).Limit(1).Find(&records).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		record := records[0]
		if record.Status != "pending" {
			return nil
		}
		// Record marked "refunded" in transaction 1:
		record.Status = "refunded"
		if err := tx.Save(&record).Error; err != nil {
			return err
		}
		userId = record.UserId
		preConsumed = record.PreConsumed
		return nil
	})
	if err != nil {
		return err
	}
	// BUG: Executed OUTSIDE the transaction!
	if preConsumed > 0 {
		return IncreaseUserQuota(userId, preConsumed, false)
	}
	return nil
}
```

And examine `ReconcileOrphanedWalletPreConsumes`:
```go
// model/wallet_pre_consume.go:145
if err := DB.Where("status = ? AND created_at < ?", "pending", cutoff).
	Limit(100).
	Find(&records).Error; err != nil {
	return 0, err
}
```

#### Attack Sequence & Impact
1. An upstream relay request fails, and `BillingSession.Refund()` calls `RefundUserWalletPreConsume`.
2. The transaction completes, marking `WalletPreConsumeRecord.Status = "refunded"`.
3. Before `IncreaseUserQuota` executes (or if `IncreaseUserQuota` fails due to a database connection glitch or server shutdown):
   - The user has NOT received their quota back.
   - The ledger record is now permanently marked `"refunded"`.
4. Subsequent calls to `RefundUserWalletPreConsume` see `record.Status != "pending"` and return `nil`.
5. The background cleaner `ReconcileOrphanedWalletPreConsumes` filters only on `status = "pending"`, completely missing this record.
6. The user's pre-consumed quota is permanently lost.

---

### Finding 6F-05: Silent Polling Cursor Freeze in Channel Multi-Key Polling Mode (`GetChannelKey`) When Memory Cache Enabled

- **Vulnerability Type**: Dead Cursor / Concurrency Update Omission / Load Imbalance
- **Severity**: **MEDIUM**
- **Affected File**: `model/channel.go`, lines 265–294
- **Call Chain**:
  `channel.GetChannelKey()`  
  `└── case constant.MultiKeyModePolling:`

#### Code Evidence

```go
// model/channel.go:265
	case constant.MultiKeyModePolling:
		channelInfo, err := CacheGetChannelInfo(channel.Id)
		if err != nil {
			return "", 0, types.NewError(err, types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
		}
		defer func() {
			if common.DebugEnabled {
				logger.LogDebug(nil, "channel %d polling index: %d", channel.Id, channel.ChannelInfo.MultiKeyPollingIndex)
			}
			if !common.MemoryCacheEnabled {
				_ = channel.SaveChannelInfo()
			} else {
				// BUG: CacheUpdateChannel is COMMENTED OUT!
				// CacheUpdateChannel(channel)
			}
		}()
		start := channelInfo.MultiKeyPollingIndex
		if start < 0 || start >= len(keys) {
			start = 0
		}
		for i := range keys {
			idx := (start + i) % len(keys)
			if getStatus(idx) == common.ChannelStatusEnabled {
				// Updates local struct copy only:
				channel.ChannelInfo.MultiKeyPollingIndex = (idx + 1) % len(keys)
				return keys[idx], idx, nil
			}
		}
```

#### Mechanism & Impact
1. `channelInfo` is fetched from `channelsIDM[id]` in `CacheGetChannelInfo`.
2. Inside the polling loop, `channel.ChannelInfo.MultiKeyPollingIndex = (idx + 1) % len(keys)` modifies the caller's transient local struct.
3. In production, `common.MemoryCacheEnabled` is `true`. The code in `defer` comments out the cache update: `// CacheUpdateChannel(channel)`.
4. `channelsIDM[id].ChannelInfo.MultiKeyPollingIndex` in memory is NEVER incremented.
5. Every single relay request reading from cache always gets `MultiKeyPollingIndex: 0`.
6. Key 0 receives 100% of all traffic, leading to rapid upstream rate limits or quota depletion on Key 0 while all remaining keys sit completely unused.

---

### Finding 6F-06: Idempotency Error Inconsistency in Stripe (`Recharge`) and Creem (`RechargeCreem`) Webhook Retries

- **Vulnerability Type**: API Contract / Webhook Retrial Fault Tolerance
- **Severity**: **LOW-MEDIUM**
- **Affected File**: `model/topup.go`, lines 258–260 (`Recharge`), lines 547–549 (`RechargeCreem`)

#### Code Evidence

Compare:
```go
// model/topup.go:195 (RechargeEpay) - PROPER IDEMPOTENCY
if topUp.Status == common.TopUpStatusSuccess {
	alreadyDone = true
	return nil
}
```
And:
```go
// model/topup.go:619 (RechargeWaffo) - PROPER IDEMPOTENCY
if topUp.Status == common.TopUpStatusSuccess {
	return nil // 幂等：已成功直接返回
}
```
With:
```go
// model/topup.go:258 (Recharge - Stripe) - CONTRACT INCONSISTENCY
if topUp.Status != common.TopUpStatusPending {
	return errors.New("充值订单状态错误")
}
```
And:
```go
// model/topup.go:547 (RechargeCreem) - CONTRACT INCONSISTENCY
if topUp.Status != common.TopUpStatusPending {
	return errors.New("充值订单状态错误")
}
```

#### Mechanism & Impact
- In Stripe, webhooks are guaranteed to be delivered at least once and may be retried across network delays.
- When `Recharge` returns `errors.New("充值订单状态错误")`, `fulfillOrder` logs an error:  
  `Stripe 充值处理失败 trade_no=... error="充值失败，请稍后重试"`.
- Although double crediting is prevented by rejecting non-pending status, returning an error breaks the idempotent success semantic and pollutes operational error telemetry with false positives.

---

## 6. Detailed Trace of Finding 6F-01: Nested Transaction Isolation Failure

```text
Time   Thread / Goroutine                      Database State
─────────────────────────────────────────────────────────────────────────────
T1     Request A fails -> BillingSession.Refund()
       Calls RefundSubscriptionPreConsume("req-123")
       tx1 = DB.Begin()
       lockForUpdate(tx1).Where("request_id = 'req-123'")
       record.Status == "consumed", PreConsumed = 50000

T2     Calls PostConsumeUserSubscriptionDelta(subId, -50000)
       tx2 = DB.Begin()  <-- SEPARATE CONNECTION FROM ROOT POOL
       lockForUpdate(tx2).Where("id = subId")
       sub.AmountUsed = max(sub.AmountUsed - 50000, 0)
       tx2.Commit()      <-- PERMANENTLY COMMITTED TO DB!

T3     tx1 attempts: record.Status = "refunded"; tx1.Save(&record)
       Network drop / DB lock timeout / crash occurs!
       tx1.Rollback()    <-- RECORD STATUS REMAINS "consumed"!

T4     refundWithRetry fires attempt 2
       Calls RefundSubscriptionPreConsume("req-123") again
       tx3 = DB.Begin()
       lockForUpdate(tx3).Where("request_id = 'req-123'")
       record.Status is STILL "consumed"!
       Calls PostConsumeUserSubscriptionDelta(subId, -50000) again!
       tx4 = DB.Begin()
       sub.AmountUsed = max(sub.AmountUsed - 50000, 0) <-- DEDUCTED TWICE!
       tx4.Commit()
       record.Status = "refunded"; tx3.Commit()
```

---

## 7. Detailed Trace of Finding 6F-02: Multi-Node Race on Stripe Delayed Failure

```text
Node A (Stripe checkout.session.completed)      Node B (Stripe async_payment_failed)
─────────────────────────────────────────────   ───────────────────────────────────────
[LockOrder(refId) - in-memory Pod A]            [LockOrder(refId) - in-memory Pod B]

                                                topUp = model.GetTopUpByTradeNo(refId)
                                                Checks: topUp.Status == "pending" (TRUE)

model.Recharge(refId) begins
txA = DB.Begin()
lockForUpdate(txA).First(topUp)
topUp.Status = "success"
creditTopUpQuota(txA, user.Id, quota)
txA.Commit()
                                                // Node B was delayed / preempted
                                                topUp.Status = common.TopUpStatusFailed
                                                topUp.Update()  <-- DB.Save(topUp)!
                                                OVERWRITES STATUS BACK TO "failed"!
```

---

## 8. Detailed Trace of Finding 6F-03: AffQuota Transfer Flaws

```text
Step   Component                          Action / State Change
─────────────────────────────────────────────────────────────────────────────
1      Client                             POST /api/user/aff_transfer {"quota": 500000}
2      controller.TransferAffQuota        Loads user, validates compliance, calls TransferAffQuotaToQuota
3      model.TransferAffQuotaToQuota      tx = DB.Begin()
                                          lockForUpdate(tx).First(user, user.Id)
                                          user.AffQuota -= 500000
                                          user.Quota += 500000
                                          tx.Save(user)  <-- writes ALL struct fields
                                          tx.Commit()
4      POST-COMMIT GAP                    NO syncCreditUserQuotaCache() called!
                                          DB has: user.quota = 500000
                                          Redis has: user:<id> -> Quota: 0
5      Client Relay Call                  Client immediately issues Chat Completions
6      service.BillingSession             Checks TryReserveUserQuota -> queries Redis Lua
7      Redis Lua Script                   tonumber(HGET user:<id> Quota) == 0 < needed
                                          RETURNS 0 (Insufficient Quota)
8      Client Result                      403 Forbidden! Transferred quota unusable.
```

---

## 9. Detailed Trace of Finding 6F-04: Two-Phase Quota Refund Commit Flaw

```text
Step   Component                          Action
─────────────────────────────────────────────────────────────────────────────
1      RefundUserWalletPreConsume         tx = DB.Begin()
                                          lockForUpdate(tx) finds pending record
                                          record.Status = "refunded"
                                          tx.Save(&record)
                                          tx.Commit() <-- Database records status = 'refunded'
2      SYSTEM CRASH / OOM / DB OUTAGE     Process killed before line 132 executes!
3      IncreaseUserQuota                  NEVER EXECUTED!
4      Reconciliation Sweep               ReconcileOrphanedWalletPreConsumes runs:
                                          WHERE status = 'pending' AND created_at < cutoff
5      Outcome                            Record is ignored because status is 'refunded'.
                                          User never receives pre-consumed quota back.
```

---

## 10. Detailed Trace of Finding 6F-05: Multi-Key Polling Index Isolation

```text
Request   Execution Path                             Outcome
─────────────────────────────────────────────────────────────────────────────
Req 1     CacheGetChannelInfo(1)                     Reads channelsIDM[1].MultiKeyPollingIndex (0)
          start = 0                                  Selects key[0]
          channel.ChannelInfo.MultiKeyPollingIndex=1 Modifies transient struct copy
          defer: MemoryCacheEnabled == true          Cache update COMMENTED OUT
Req 2     CacheGetChannelInfo(1)                     Reads channelsIDM[1].MultiKeyPollingIndex (0)
          start = 0                                  Selects key[0] again!
Req N     CacheGetChannelInfo(1)                     Reads channelsIDM[1].MultiKeyPollingIndex (0)
          start = 0                                  Key 0 overloaded, Keys 1..N starved
```

---

## 11. Detailed Trace of Finding 6F-06: Webhook Idempotency Contract Mismatch

```text
Gateway           Method                    Duplicate Delivery Behavior
─────────────────────────────────────────────────────────────────────────────
Epay              RechargeEpay              Returns (true, nil), logs "重复回调幂等忽略" (Clean 200 OK)
Waffo             RechargeWaffo             Returns nil immediately (Clean 200 OK)
Waffo Pancake     RechargeWaffoPancake      Returns nil immediately (Clean 200 OK)
Stripe            Recharge                  Returns "充值订单状态错误", logs "Stripe 充值处理失败"
Creem             RechargeCreem             Returns "充值订单状态错误", controller absorbs error
```

---

## 12. Object Ownership & IDOR Verification Analysis

A exhaustive IDOR audit was conducted across all resource identifiers:

1. **Tokens (`/api/token`)**:
   - `GetAllTokens`: Filters strictly by `c.GetInt("id")`.
   - `GetToken` / `DeleteToken` / `UpdateToken`: Uses `model.GetTokenByIds(id, userId)`. Impossible to manipulate other users' tokens.
2. **BYOK Providers (`/api/user/providers`)**:
   - `GetUserProviderByID(userID, id)` and `DeleteUserProvider(userID, id)` strictly enforce `user_id = ? AND id = ?`.
3. **Tasks (`/api/task`)**:
   - `GetTask`: Uses `model.GetByTaskId(c.GetInt("id"), c.Param("key"))`.
   - Artifact downloads use capability tokens with purpose, expiration, and HMAC signatures.
4. **Subscriptions (`/api/subscription`)**:
   - `GetSubscriptionSelf`: Evaluates `c.GetInt("id")`.
   - Admin management (`/api/subscription/admin/*`) strictly protected by `middleware.AdminAuth()`.
5. **Redemptions (`/api/redemption`)**:
   - Management routes protected by `middleware.AdminAuth()`.
   - Redemption execution (`/api/user/topup`) binds the code to `c.GetInt("id")`.

---

## 13. Mass Assignment & Data Binding Security Analysis

- **`UpdateSelf` (`controller/user.go:778`)**:
  - Decodes input, but explicitly instantiates:
    ```go
    cleanUser := model.User{
        Id:          c.GetInt("id"),
        Username:    user.Username,
        Password:    user.Password,
        DisplayName: user.DisplayName,
    }
    ```
  - `role`, `quota`, `group`, `admin_permissions` cannot be mass-assigned.
- **`UpdateUser` (`controller/user.go:649`)**:
  - Restricts `updatedUser.Role = originUser.Role`.
  - Enforces `canManageTargetRole(myRole, originUser.Role)`.
  - Requires StepUp security proof when updating password or admin permissions.
- **`AddToken` (`controller/token.go:287`)**:
  - Sanitizes user input and forces `cleanToken.UserId = c.GetInt("id")`.
- **`AddRedemption` (`controller/redemption.go:65`)**:
  - Validates `quota > 0`, `ValidateWalletQuota`, and sets `UserId = c.GetInt("id")`.

---

## 14. Concurrency & TOCTOU Audit

| Resource / Action | Mechanism Used | Multi-Node Safe? | Verdict |
| :--- | :--- | :--- | :--- |
| **Token Quota Deduction** | Redis Lua script `tokenQuotaReserveScript` / DB conditional update | **YES** | **PASS** |
| **Wallet Quota Deduction** | Redis Lua script `userQuotaReserveScript` / DB conditional update | **YES** | **PASS** |
| **TopUp Quota Credit** | `creditTopUpQuota` atomic `Where("id = ? AND quota <= ?", ...)` | **YES** | **PASS** |
| **Redemption Code Use** | DB transaction + `lockForUpdate` + CAS on `status == enabled` | **YES** | **PASS** |
| **Subscription Completion**| DB transaction + `lockForUpdate(order)` + lock user row `FOR UPDATE` | **YES** | **PASS** |
| **Task Status Transition** | `task.UpdateWithStatus(fromStatus)` conditional CAS UPDATE | **YES** | **PASS** |
| **Stripe TopUp Failure** | `sessionAsyncPaymentFailed` non-transactional read-then-write | **NO** | **FINDING 6F-02** |
| **Subscription Refund** | Nested `DB.Transaction` inside `RefundSubscriptionPreConsume` | **NO** | **FINDING 6F-01** |
| **Wallet Refund Commit** | Two-phase non-atomic status commit in `RefundUserWalletPreConsume`| **NO** | **FINDING 6F-04** |
| **Multi-Key Cursor** | Local variable write with commented-out cache update | **NO** | **FINDING 6F-05** |

---

## 15. Webhook Replay & Idempotency Audit Matrix

| Gateway | Webhook Endpoint | Verification Mechanism | Idempotent Handling | Multi-Node Safe |
| :--- | :--- | :--- | :--- | :--- |
| **Epay** | `/api/user/epay/notify` | Epay MD5 / HMAC signature | Row lock + Status check + CAS | **YES** |
| **Stripe** | `/api/user/stripe/webhook` | Stripe HMAC-SHA256 signature | Row lock in `Recharge` (logs error on retry) | Partial (Finding 6F-02, 6F-06) |
| **Creem** | `/api/user/creem/webhook` | Creem HMAC-SHA256 signature | Controller checks status == pending | Partial (Finding 6F-06) |
| **Waffo** | `/api/user/waffo/notify` | Waffo RSA signature | Row lock + returns nil on success | **YES** |
| **Waffo Pancake** | `/api/user/waffo-pancake/webhook` | Waffo Pancake HMAC signature | Row lock + returns nil on success | **YES** |

---

## 16. Soft Delete Lifecycle & Auth Version Fencing Audit

- **User Deletion**:
  - `model/user.go:1000` executes hard deletion within a transaction (`tx.Unscoped().Delete(user)`).
  - Advances `auth_version` before deletion and publishes tombstone to Redis.
  - Revokes all access tokens and purges all user sessions.
  - Invalidates in-memory and Redis caches.
- **Token Deletion**:
  - Soft deleted (`gorm.DeletedAt`), but `Delete()` calls `invalidateTokenCacheForMutation(token.Key)`.
  - Relay authentication checks `deleted_at IS NULL` via standard GORM queries.
- **Redemption Code Deletion**:
  - Soft deleted; queries automatically apply `deleted_at IS NULL`. Soft-deleted codes cannot be redeemed.

---

## 17. Threat Modeling & Attack Scenario Simulations

### Scenario 1: Double Redemption Race
- **Attacker Goal**: Redeem a single $10 gift code simultaneously across 20 parallel threads to obtain $200.
- **Simulation**:
  20 threads hit `POST /api/user/topup` with the same key.
  All threads enter `model.Redeem`.
  Thread 1 acquires `lockForUpdate(tx)` on the key row, checks `status == 1`, runs CAS update to `status = 2`, credits quota, commits.
  Threads 2–20 unblock; their CAS queries observe `status == 2` (`RowsAffected == 0`), fail the check, and rollback without crediting quota.
- **Outcome**: **PREVENTED** (Single credit of $10).

### Scenario 2: Concurrent Wallet Depletion Overdraw
- **Attacker Goal**: User with $1.00 balance issues 50 concurrent requests costing $0.50 each to spend $25.00.
- **Simulation**:
  50 requests enter `TryReserveUserQuota`.
  Requests hit Redis Lua script `userQuotaReserveScript`.
  Requests 1 and 2 decrement quota from $1.00 to $0.50, then $0.50 to $0.00.
  Requests 3–50 evaluate `quota < amount` (0 < 0.50) in Lua and return `0` (insufficient).
  Requests 3–50 abort before reaching upstream providers.
- **Outcome**: **PREVENTED** (Max spent: $1.00).

### Scenario 3: Stripe Asynchronous Webhook Glitch (Finding 6F-02)
- **Attacker Goal**: Exploit delayed bank transfer failure event arriving after payment success to invalidate payment status in backend.
- **Simulation**:
  User pays via Stripe SEPA.
  `checkout.session.completed` arrives at Node A -> sets `status = success`, credits wallet.
  Late `async_payment_failed` arrives at Node B -> executes un-locked `GetTopUpByTradeNo` and `Update()`.
- **Outcome**: **VULNERABLE** (Status reverted to `failed` despite credited balance).

---

## 18. Prior Phase Non-Regression Audit

| Prior Phase | Core Controls Checked | Status |
| :--- | :--- | :--- |
| **Phase 5C/5D** | BYOK SSRF, Private IP filters, DNS rebinding, Credential encryption | **PRESERVED** (No regression) |
| **Phase 6B** | StepUp Verification, Token masking, MJ Capability token HMAC | **PRESERVED** (No regression) |
| **Phase 6C** | Pre-consumption reservation ledger, Wallet quota ceiling, Midjourney pricing | **PRESERVED** (No regression) |
| **Phase 6D** | Distributed relay concurrency leases, relay timeouts, response ceiling | **PRESERVED** (No regression) |
| **Phase 6E/6E-R** | Sensitive data redaction, secure headers, CORS, cookie security | **PRESERVED** (No regression) |

---

## 19. Actionable Remediation Guidance Matrix

| Finding ID | Priority | Remediation Target | Recommended Fix |
| :--- | :--- | :--- | :--- |
| **6F-01** | P1 (High) | `model/subscription.go` | Implement `postConsumeUserSubscriptionDeltaWithTx(tx *gorm.DB, ...)` and call it within `RefundSubscriptionPreConsume` to preserve the outer transaction boundary. |
| **6F-02** | P1 (High) | `controller/topup_stripe.go` | Replace manual read-and-update in `sessionAsyncPaymentFailed` with `model.UpdatePendingTopUpStatus(referenceId, model.PaymentProviderStripe, common.TopUpStatusFailed)`. |
| **6F-03** | P2 (Med-High)| `model/user.go` | In `TransferAffQuotaToQuota`: replace `tx.Save(user)` with atomic updates, validate `ValidateTopUpQuotaCapacity`, and invoke `syncCreditUserQuotaCache(user.Id, quota, "aff transfer")` upon commit. |
| **6F-04** | P2 (Medium) | `model/wallet_pre_consume.go` | Perform `IncreaseUserQuota` (or atomic user quota increment) inside the same database transaction as `record.Status = "refunded"`. |
| **6F-05** | P2 (Medium) | `model/channel.go` | Fix `MultiKeyModePolling` to update the cached `channelsIDM[id].ChannelInfo.MultiKeyPollingIndex` atomically under channel lock when `MemoryCacheEnabled = true`. |
| **6F-06** | P3 (Low-Med) | `model/topup.go` | In `Recharge` and `RechargeCreem`, return idempotent success (`alreadyDone = true` / `nil`) when `topUp.Status == common.TopUpStatusSuccess`. |

---

## 20. Final Audit Verdict & Readiness Certification

The Phase 6F Business Logic, Race Condition & State Transition Security Audit of the SaaSCover / New-API backend is complete.

The backend demonstrates robust defenses against unauthorized elevation, direct object takeover (IDOR), mass assignment, gift code race conditions, and relay balance overdraw. However, six concrete business logic and state transition issues were identified, including two high-severity concurrency/transaction boundary vulnerabilities.

### Readiness Declaration

- **Security Posture**: **STRONG WITH ISOLATED LOGIC GAPS**
- **Audit Verdict**: **PASS WITH FINDINGS**
- **Production Status**: Ready for Phase 6F Remediation (6F-R) to address the six identified findings prior to final mobile production launch.

---
*Report certified by Antigravity Autonomous Security Pair Programmer on 2026-10-05.*
