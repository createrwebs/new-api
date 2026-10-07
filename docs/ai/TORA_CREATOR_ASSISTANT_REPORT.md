# Tora Studio Queue 6: Creator Assistant & ToolPlan Report

> **[SUPERSEDED NOTICE]**  
> **Status:** SUPERSEDED by `docs/ai/TORA_STUDIO_FINAL_ACCEPTANCE_REPORT.md`.  
> **Audit Finding:** The Creator Assistant and ToolPlan engine architecture (natural language parsing, multi-step dependency DAG, authoritative quoting, and partial step-level settlement/refund) is fully verified in code and tests. However, live multi-step execution against external providers requires live provider API credentials (`FAL_KEY`). Under the strict acceptance policy, the true status is:  
> **`QUEUE 6 STATUS: IMPLEMENTATION VERIFIED — REAL WORKFLOW E2E PENDING (OPERATOR_BLOCKED)`**.

**Original Timestamp:** 2026-10-07T07:18:00+07:00
**Authoritative Ledger:** Single Tora Wallet (`model.PreConsumeUserWallet`, `model.SettleUserWalletPreConsume`)

---

## 1. Executive Summary & Architecture

Tora Studio Queue 6 delivers the **Creator Assistant** and the verifiable **ToolPlan** execution system. The assistant operates under a strict safety invariant:
> **The Assistant NEVER calls providers directly.**
> Architecture: Natural Language $\rightarrow$ ToolPlan $\rightarrow$ Server-Authoritative Quote $\rightarrow$ User Confirmation $\rightarrow$ Step-Level Studio Jobs.

Users express their desired outcome naturally in Thai or English without being exposed to backend models or provider endpoints. Tora automatically composes the optimal sequence of tools, obtains authoritative quotes for every step, and executes them with step-level billing protection.

---

## 2. ToolPlan Specification & Natural Language Resolution

When a user submits a creative outcome request, the `StudioAssistantEngine` resolves the prompt into discrete sequential steps with explicit dependency links:

### Example Canonical Workflow:
- **User Prompt:** `"ทำรูปสินค้านี้เป็นโฆษณา Instagram แล้วทำวิดีโอ 5 วิ"`
- **Initial Asset:** Raw product image URL (e.g. `raw_shoe.png`)

```
Step 1: background-remove
  - Tool: BiRefNet background isolation
  - DependsOn: [] (takes raw upload)
  - Authoritative Quote: 10 Credits (10,000 Quota)

Step 2: product-photo
  - Template: tpl-prod-instagram-4-5 (Instagram 4:5)
  - DependsOn: [Step 1] (takes cutout output from Step 1)
  - Authoritative Quote: 50 Credits (50,000 Quota)

Step 3: image-to-video
  - Tool: Wan 2.2 720p (5 seconds duration)
  - DependsOn: [Step 2] (takes composite photo output from Step 2)
  - Authoritative Quote: 125 Credits (125,000 Quota)

Total Authoritative Quote: 185 Tora Credits (185,000 Quota = $0.370 USD reference value)
```

---

## 3. Workflow Billing & Step-Level Safety Invariants

1. **Step-Level Reserve & Settle:**
   Each step reserves and settles independently through `SubmitJob`.
2. **Partial Failure Safety (Never Refund Successful Steps):**
   If Step 1 (10 Credits) and Step 2 (50 Credits) succeed, but Step 3 (125 Credits) encounters an upstream provider error:
   - Steps 1 & 2 remain settled in the ledger.
   - Only Step 3's 125 Credits are refunded.
   - The user has paid exactly 60 Credits for the completed assets.
3. **Step-Level Retry:**
   The user or system can retry *only* the failed Step 3 without repeating or re-charging for Steps 1 and 2.
4. **Insufficient Credit UX:**
   If a workflow requires 185 Credits and the user has 50 Credits:
   - System returns structured data: `Required: 185`, `Available: 50`, `Missing: 135`.
   - UI provides "Buy Credits" button redirecting to wallet and a "Requote" button to refresh quotes after top-up.
5. **Safety Guard (Identity / Face Swap):**
   Any identity-altering workflows (e.g. face swap) automatically flag `requires_consent: true` and are blocked from execution until the user explicitly confirms legal rights and consent.

---

## 4. Frontend Assistant UX

- **Routes:** Accessible via `/assistant` and `/studio` (Tab: "ผู้ช่วย AI (Assistant)").
- **Visual Step Progress:** Live step status indicators (`PENDING` $\rightarrow$ `RUNNING` $\rightarrow$ `SUCCEEDED` / `FAILED`), live previews of intermediate PNG/MP4 assets, and individual step retry buttons.

---

## 5. Automated Verification Results

```
=== RUN   TestQueue6_AssistantPlan_ThaiIntentResolution
--- PASS: TestQueue6_AssistantPlan_ThaiIntentResolution (0.01s)
=== RUN   TestQueue6_WorkflowBilling_StepLevelExecutionAndSafety
--- PASS: TestQueue6_WorkflowBilling_StepLevelExecutionAndSafety (0.01s)
=== RUN   TestQueue6_WorkflowBilling_PartialFailure_NeverRefundsSuccessfulSteps
--- PASS: TestQueue6_WorkflowBilling_PartialFailure_NeverRefundsSuccessfulSteps (0.01s)
=== RUN   TestQueue6_InsufficientCredit_UX_Data
--- PASS: TestQueue6_InsufficientCredit_UX_Data (0.01s)
=== RUN   TestQueue6_Safety_FaceSwapConsentRequired
--- PASS: TestQueue6_Safety_FaceSwapConsentRequired (0.01s)
PASS
ok  	github.com/QuantumNous/new-api/service	0.607s
```
