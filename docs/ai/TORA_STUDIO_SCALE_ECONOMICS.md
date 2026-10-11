# TORA STUDIO QUEUE 8: ECONOMICS, SCALE & SELF-HOST DECISION ENGINE

**Timestamp:** 2026-10-07T07:26:30+07:00  
**Repository:** `/Users/noppanan/new-api`  
**Branch:** `feat/formobile`  
**Authoritative Framework:** `service/studio_scale.go` & `service/studio_scale_test.go`

---

## 1. Executive Summary

Queue 8 establishes the quantitative decision framework governing when Tora Studio should consume commercial upstream APIs versus when it should graduate to serverless container GPUs or dedicated self-hosted clusters.

**Rule Zero:** Tora Studio will **NOT** purchase, reserve, or deploy dedicated GPU instances based on speculation or vanity metrics. All infrastructure decisions are driven by rolling, evidence-based economic calculations, true fully-loaded cost modeling (compute, idle, storage, egress, ops, and incidents), and rigid trigger conditions.

---

## 2. Rolling Telemetry & Operational Metrics

The telemetry engine (`ScaleDecisionEngine.ComputeRollingTelemetry`) continuously calculates 11 primary operational indicators over a 30-day sliding window:

| Metric | Calculation / Definition | Purpose |
| :--- | :--- | :--- |
| **jobs/day** | $\frac{\text{Total Jobs}}{\text{Window Days}}$ | Workload density & burstiness |
| **jobs/month** | $\text{jobs/day} \times 30$ | Monthly capacity requirement |
| **provider spend** | $\sum \text{Cost}_{\text{USD}}$ (Provider COGS) | Real external API expenditure |
| **Tora Credit sell value** | $\sum \text{ToraRevenue}_{\text{USD}}$ | Gross billing revenue |
| **gross profit** | $\text{Revenue}_{\text{USD}} - \text{ProviderSpend}_{\text{USD}}$ | Absolute cash generation |
| **gross margin** | $\frac{\text{GrossProfit}}{\text{Revenue}} \times 100\%$ | Unit economic health |
| **p50 latency** | Median duration $(\text{CompletedAt} - \text{CreatedAt})$ | Typical user experience |
| **p95 latency** | 95th percentile duration | Worst-case user wait time |
| **success rate** | $\frac{\text{Successful Jobs}}{\text{Total Jobs}} \times 100\%$ | Upstream pipeline reliability |
| **refund rate** | $\frac{\text{Failed Jobs}}{\text{Total Jobs}} \times 100\%$ | Wallet auto-refund frequency |
| **repeat use** | $\frac{\text{Users with } \ge 2\text{ jobs}}{\text{Total Unique Users}} \times 100\%$ | Retention & sticky customer base |

---

## 3. Candidate Open-Source Stacks

When evaluating the technical feasibility of self-hosting, current upstream API routes are benchmarked against specific open-source candidate architectures:

| Logical Tool | Candidate Open-Source Stack | Model Family | License Status | VRAM Required | Target GPU | Reference Cost (RunPod/Lambda) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **background-remove** | `rembg` / `BiRefNet` Container | BiRefNet / U-2-Net | MIT | 4GB | NVIDIA T4 / RTX 3060 | \$0.22/hr |
| **image-upscale** | `Real-ESRGAN` / Compact | Real-ESRGAN | BSD-3-Clause | 8GB | NVIDIA RTX 3090 / A4000 | \$0.44/hr |
| **image-generate** | `ComfyUI` + `Flux-schnell` | Flux.1 | Apache-2.0 | 16GB - 24GB | NVIDIA RTX 4090 / A5000 | \$0.79/hr |
| **product-photo** | `ComfyUI` + ControlNet + Inpainting | SDXL / Flux | OpenRAIL / Apache-2.0 | 24GB | NVIDIA A5000 | \$0.79/hr |
| **image-to-video** | `Wan 2.1` / `LTX-Video` | Wan 2.1 / LTX | Apache-2.0 | 48GB - 80GB | NVIDIA A100 80GB / H100 | \$2.19 - \$3.50/hr |
| **lip-sync** | `MuseTalk` Serverless | MuseTalk | Academic/Research | 16GB | NVIDIA A4000 | \$0.44/hr |
| **face-swap** | `FaceFusion` Pipeline | InsightFace / FF | GPL-3.0 (Ethics Review Req.) | 16GB | NVIDIA RTX 3090 | \$0.44/hr |

---

## 4. Full Cost Model: The "Hidden" Costs of Self-Hosting

Self-hosting is frequently miscalculated by comparing only raw per-second inference time against API prices. Tora's `SelfHostCostModel` factors in the complete operational reality:

$$\text{Total Monthly Self-Host Cost} = \text{Compute}_{\text{active}} + \text{Compute}_{\text{idle}} + \text{Egress} + \text{Storage} + \text{Maintenance}_{\text{Ops}} + \text{IncidentBurden}$$

### Detailed Breakdown for a Dedicated A4000 Node:
1. **Compute Base (720 hours @ \$0.44/hr):** \$316.80/month.
2. **Idle Capacity Waste:** Without 24/7 sustained queue load, 25%–50% of GPU cycles burn during Thai off-peak hours (01:00–07:00).
3. **Storage & Volume Persistence:** \$20.00/month (Fast NVMe volume for model weights and checkpoints).
4. **Bandwidth / Egress:** \$30.00/month (transferring high-resolution images/videos to Cloudflare R2 / client CDN).
5. **Maintenance & DevOps Hours:** \$400.00/month (8 hours/month @ \$50/hr internal rate for patching, CUDA updates, driver compatibility, ComfyUI node breaks).
6. **Incident Burden & On-Call Risk Buffer:** \$150.00/month (compensation for unexpected crashes, disk fullness, provider node preemption).
7. **SLA & Cold Start Tradeoff:** 
   - Managed API: **99.95% availability**, 0s cold start.
   - Self-hosted serverless: **99.2% availability**, 12–30s container cold starts.

$$\text{True Minimum Fully-Loaded Self-Host Basis} \approx \mathbf{\$916.80 / \text{month}}$$

---

## 5. Decision Engine Taxonomy

Every tool is dynamically classified into one of five states:

1. **`KEEP_API`:**
   Current API spend is below threshold, margins are healthy (> 50%–70%), and the API delivers zero ops burden and 99.95% reliability.
2. **`ADD_PROVIDER`:**
   Spend is moderate, but API success rate is degraded (< 95%). Adding a secondary commercial API with automated safe fallback provides resilience without server management.
3. **`SERVERLESS_GPU_CANDIDATE`:**
   Meets economic spend and margin criteria, but workload is spiky/bursty. Moving to pay-per-second containerized serverless (e.g. RunPod Serverless, Modal) captures cost savings without paying for idle hours.
4. **`SELF_HOST_CANDIDATE`:**
   Tool spend exceeds \$1,000/month, gross margin is compressed below 50%, self-host net savings exceed 50% *after* all ops costs, and job density is high (> 1,000 jobs/day).
5. **`NOT_ECONOMIC`:**
   Gross margin is negative (sell price is lower than upstream COGS). Immediate pricing reconfiguration or tool retirement required.

---

## 6. The Authoritative Trigger Rule

Tora Studio enforces an immutable rule before considering self-hosting any tool:

```
IF:
    (tool_spend > $1,000 / month)
    AND (gross_margin < 50%)
    AND (self_host_savings > 50%)
    AND (maintenance_costs < projected_savings)
THEN:
    Evaluate Graduating to Serverless GPU / Self-Host
ELSE:
    KEEP API-FIRST.
```

### Why High-Spend With High-Margin Must Remain API-First:
As proven in unit test `TestQueue8_TriggerRule_HighSpendButHighMargin_RemainsKeepAPI`:
- If a tool generates \$10,000/month revenue on \$2,500/month API spend, its gross margin is **75.0%**, yielding **\$7,500/month net profit**.
- Spending engineering hours managing GPU infrastructure to save \$800 while introducing downtime risk, cold starts, and maintenance drag is an operational anti-pattern.

---

## 7. Current Tora Studio Production Status

Applying the decision engine to current Tora Studio production tools:

| Tool | Current Route | Provider Spend | Gross Margin | Decision Classification | Justification |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **background-remove** | Replicate (Fast) / Fal (Quality) | < \$50/mo | **75% – 85%** | **KEEP_API** | High margin, \$0.003–\$0.005 unit cost. |
| **image-upscale** | Replicate (Fast) / Fal (Premium) | < \$50/mo | **70% – 83%** | **KEEP_API** | High margin, dual-provider redundancy. |
| **image-generate** | Fal / Replicate Flux | < \$50/mo | **68% – 81%** | **KEEP_API** | Flux models rapidly evolving; zero GPU lock-in. |
| **product-photo** | Fal Flux LoRA / Inpaint | < \$100/mo | **75.0%** | **KEEP_API** | High perceived user value (50 Credits = \$0.100). |
| **image-to-video** | Fal Kling / Minimax | < \$200/mo | **68.0%** | **KEEP_API** | Video GPUs require A100s (\$2.50–\$3.50/hr). API is far cheaper. |

---

## 8. Verified Test Proof

All 7 test specifications pass in `service/studio_scale_test.go`:
```
=== RUN   TestQueue8_RollingTelemetryCalculation
--- PASS: TestQueue8_RollingTelemetryCalculation (0.00s)
=== RUN   TestQueue8_CandidateStacks
--- PASS: TestQueue8_CandidateStacks (0.00s)
=== RUN   TestQueue8_DecisionEngine_KeepAPI
--- PASS: TestQueue8_DecisionEngine_KeepAPI (0.00s)
=== RUN   TestQueue8_DecisionEngine_AddProvider
--- PASS: TestQueue8_DecisionEngine_AddProvider (0.00s)
=== RUN   TestQueue8_DecisionEngine_ServerlessGPUCandidate
--- PASS: TestQueue8_DecisionEngine_ServerlessGPUCandidate (0.00s)
=== RUN   TestQueue8_DecisionEngine_SelfHostCandidate
--- PASS: TestQueue8_DecisionEngine_SelfHostCandidate (0.00s)
=== RUN   TestQueue8_TriggerRule_HighSpendButHighMargin_RemainsKeepAPI
--- PASS: TestQueue8_TriggerRule_HighSpendButHighMargin_RemainsKeepAPI (0.00s)
PASS
ok  	github.com/QuantumNous/new-api/service	0.568s
```

---

QUEUE 8 STATUS: API-FIRST REMAINS OPTIMAL
