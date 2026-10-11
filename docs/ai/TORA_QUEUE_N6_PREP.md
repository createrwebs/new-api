# Tora Studio — Queue N6 Preparation & Transition
**File:** `docs/ai/TORA_QUEUE_N6_PREP.md`  
**Timestamp:** 2026-10-08T22:21:00+07:00  

---

## 1. Baseline State Inherited from Queue N5

1. **Backend Engine**:
   - Marketplace Compliance Registry (`GET /api/studio/marketplace/rules`, `POST /api/studio/marketplace/validate`)
   - Seller Brand Profiles (`/api/studio/seller/profiles`)
   - Reusable Workflow Presets (`/api/studio/seller/presets`)
   - Repeat Last Pack (`GET /api/studio/seller/repeat-last`)
   - Resumable Batch Items (`StudioWorkflowItem`, `ResumableWorkflowJob`)
   - Batch 25 Volume Tier (15% discount for 25 items; flat heap profile verified)
2. **Infrastructure**:
   - Production Host: AWS EC2 `51.20.174.90` (t4g.small ARM64 Graviton2)
   - Zero additional servers or GPU infrastructure added.
   - Preserved rollback targets.
3. **Mobile Client Condition**:
   - `ANDROID_PHYSICAL_DEVICE = PENDING`
   - `IOS_PHYSICAL_DEVICE = PENDING`
   - Android release APK built and verified.
   - Flutter repository local backup bundles secured at `/Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/`.

---

## 2. Priority Themes for Queue N6

1. **Physical Mobile Device Lab Acceptance**:
   - When physical Android or iOS hardware is attached to the workstation, execute real USB adb/xcode installation.
   - Run real on-device model inference (CoreML / NNAPI / XNNPACK) on camera photos.
   - Verify zero network exfiltration of raw input images during local inference.
2. **Seller Factory UI Parity (Flutter & Web)**:
   - Wire Seller Brand Profiles and Workflow Presets directly into the mobile Flutter UI (`LumenFlow`).
   - Implement the "Repeat Last Pack" one-click action button on the Home / Factory tab.
   - Add the interactive Marketplace Compliance Badge indicating compliance status (Shopee Mall pass, TikTok Shop pass) before batch export.
3. **Resumable Batch UI Streaming**:
   - Wire item-level progress indicators to `StudioWorkflowItem` states.
   - Allow sellers to retry failed items with zero credit deduction.
