You are working inside a fork of `QuantumNous/new-api` on branch `tora-api`.

Goal: implement a production-safe Budget V1 for Tora API on top of New API's existing billing system.

Current implementation status:

1. Added `model/budget.go`
2. Added `BudgetRule`
3. Added `BudgetUsage`
4. Registered both models in `model/main.go` inside `DB.AutoMigrate(...)`
5. Added helper functions:
   - `GetEnabledBudgetRules`
   - `GetBudgetPeriodStart`
   - `GetBudgetUsage`
   - `CheckBudget`
   - `AddBudgetUsage`
   - `ReserveBudget`
   - `ReleaseBudget`
   - `SettleBudget`
6. Added `BudgetReservation`
7. Added budget reservation lifecycle support to `BillingSession`
8. Updated `PreConsumeBilling()` to reserve user/token budgets before creating the billing session
9. Updated billing session refund/settle paths so budget reservations are released or settled
10. Added `model/budget_test.go`
11. Current model budget tests all pass:
   - `TestReserveBudgetWithinLimit`
   - `TestReserveBudgetRejectsOverLimit`
   - `TestReleaseBudgetReturnsReservedQuota`
   - `TestSettleBudgetAdjustsToActualQuota`
   - `TestSettleBudgetIncreasesWhenActualExceedsReservation`
   - `TestDailyAndMonthlyBudgetsBothReserve`

Important architecture:

New API currently uses `BillingSession` as the main billing lifecycle.

Relevant flow:

```text
PreConsumeBilling()
    ↓
Reserve user budget
    ↓
Reserve token budget
    ↓
NewBillingSession()
    ↓
BillingSession holds budget reservations
    ↓
upstream request
    ↓
BillingSession.Settle(actualQuota)
    ↓
SettleBudget(...)
```

Failure flow:

```text
request failure
    ↓
BillingSession.Refund()
    ↓
refund funding
    ↓
refund token quota
    ↓
ReleaseBudget(...)
```

Relevant source locations:

- `service/billing.go`
- `service/billing_session.go`
- `model/budget.go`
- `model/main.go`
- `model/budget_test.go`

`NewBillingSession` is currently located around:

```text
service/billing_session.go:407
```

Existing compile checks pass:

```bash
go test ./model -run TestNonExistent
go test ./service -run TestNonExistent
go test ./model -run Budget -v
```

Do NOT deploy anything to production yet.

Next task:

1. Inspect `NewBillingSession()` and the surrounding billing code carefully.
2. Add integration/unit tests for the budget lifecycle through the real `BillingSession` flow where practical.
3. Tests should cover at minimum:
   - budget reserved during `PreConsumeBilling`
   - user budget + token budget both reserved
   - successful settle adjusts reservation to actual quota
   - failed/refunded request releases budget
   - billing-session creation failure releases previously reserved budget
   - token budget reservation failure releases user reservation
   - playground requests do not reserve token budget
4. Reuse existing New API test patterns and SQLite test setup where possible.
5. Avoid invasive refactors.
6. Keep all existing New API billing behavior unchanged when no BudgetRule exists.
7. Do not duplicate usage accounting. Budget usage must be counted exactly once.
8. Preserve existing token quota, wallet, subscription, trusted-credit, refund, and settle behavior.
9. Run targeted tests after each change.
10. Then run broader tests for affected packages.

Important production-safety concerns:

- Budget checks must be concurrency-safe.
- Budget reservation should be atomic.
- If a user budget reservation succeeds but token reservation fails, user reservation must be rolled back.
- If `NewBillingSession()` fails after budget reservation, all budget reservations must be released.
- If billing succeeds but budget bookkeeping fails, do not incorrectly convert a successful upstream/billing request into a client-visible failure unless absolutely necessary.
- Existing funding/token billing remains the source of truth for actual money/quota charging.
- Budget is only a usage cap layer, not a replacement billing system.

Before changing code, inspect:

```bash
sed -n '407,620p' service/billing_session.go
```

Also inspect related functions used by `NewBillingSession()`.

Work directly in the repo, make the code changes, run tests, and report:
- files changed
- tests added
- tests run
- any edge cases or risks found
- recommended next step after Budget V1 backend is stable

Do not stop after analysis; make the code changes and validate them.