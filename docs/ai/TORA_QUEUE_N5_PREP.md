# Tora Studio Queue N5 Preparation & Architecture Plan
**Queue:** `QUEUE N5 PREVIEW`  
**Milestone:** Seller Automation, High-Volume Batch Expansion & Brand Presets  
**Timestamp:** 2026-10-08T20:36:00+07:00  

---

## 1. Core Mission & Scope for Queue N5

Queue N5 builds directly on the proven foundation of **Seller Factory V2** and **Tora Native Engine**:
1. **Batch Scaling (10 $\to$ 25 $\to$ 50)**:
   - Queue N4 verified the 3-item chunking engine scales with linear latency ($130\text{ms}$ per item) and flat memory ($12.2\text{MB}$ heap delta).
   - In N5, test and unlock 25-item and 50-item enterprise merchant tiers under verified memory guardrails.
2. **Saved Seller Brand Profiles**:
   - Allow merchants to persist brand profiles containing:
     - Primary and secondary brand colors (hex codes).
     - Preferred marketplace channels (e.g. Shopee + TikTok Shop defaults).
     - Standard shadow preset preference (e.g. `MARKETPLACE` for white-label goods, `SOFT_STUDIO` for cosmetics).
     - Default watermark / logo asset placement.
3. **One-Click Seller Automation**:
   - "Repeat Last Pack" functionality.
   - Smart workflow presets: "Cosmetics Showcase", "Electronics Catalog", "Fashion Flat-Lay".
4. **Physical Mobile Device Canary Gate**:
   - As physical Android or iOS test devices become attached via USB, run the automated physical acceptance gate immediately without manual code modifications.
5. **Commercially Clean Retouching Upgrades**:
   - Continue research into clean-room, rights-audited texture synthesis and patch-based inpainting without relying on non-commercial research models (preserving the strict exclusion of LaMa and MAT).
   - Zero GPU purchases, zero new server infrastructure, and strict preservation of the canonical Tora Credit system ($1\text{ Credit} = 1,000\text{ Quota} = \$0.002$).
