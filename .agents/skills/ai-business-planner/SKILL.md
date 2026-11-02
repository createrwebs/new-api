---
name: ai-business-planner
description: >-
  Framework and methodology for evaluating, analyzing, and executing AI SaaS business models,
  unit economics, pricing strategies, dual-brand positioning, and international go-to-market
  (GTM) motions for developer and creator platforms (such as toraapi.com and createai.site).
  Use whenever analyzing business viability, calculating margins/COGS, structuring pricing
  tiers, evaluating upstream provider agreements, planning user acquisition, or assessing
  operational and compliance risks.
---

# AI SaaS Business Planner & GTM Execution Skill

This skill provides the authoritative framework for evaluating and scaling AI businesses within the Tora ecosystem, specifically across developer infrastructure (`toraapi.com`) and business automation (`createai.site`).

---

## 1. Unit Economics & Pricing Governance

### 1.1 Canonical Ledger Anchors
- **Unified Quota Engine**: `1.00 USD = 500,000 Quota Units`.
- **Display Credit Rate**: `1 Tora Credit = 1,000 Quota Units = $0.0020 USD`.
- **Margin Floor Invariant**: Server-side gross margin MUST NOT drop below **60.0%** on managed routes.
- **Stripe & Payment Overhead**: Model all consumer retail pricing with at least 3.5% + $0.30 payment processing cost buffer.

### 1.2 Dual Pricing Architecture
1. **Pay-As-You-Go (Prepaid Top-Up)**:
   - Target: Developers and creators who reject monthly recurring commitments ("Subscription Fatigue").
   - Multipliers: Standard 1.0x rate.
   - Top-up increments: $5, $15, $50.
2. **Hybrid Subscription (Tiered + Daily/Weekly Limits)**:
   - Target: Active power users requiring predictable cost ceilings and priority concurrency.
   - Discount: 10% lower credit burn rate (0.9x multiplier) on quota usage.
   - Quota Enforcement: Hard daily/weekly governor to prevent runaway agentic loops or account sharing.

---

## 2. Dual-Brand Separation (Clean Boundary Invariant)

To eliminate foreign buyer skepticism and compliance risks, preserve strict separation between developer infrastructure and business marketing tools:

| Dimension | `toraapi.com` | `createai.site` |
| :--- | :--- | :--- |
| **Primary Audience** | Software engineers, AI builders, technical creators | Small business owners, marketers, e-commerce sellers, agencies |
| **Core Product Lines** | VS Code Extension, Mobile IDE, High-Speed Model Relay, Short Video/Design Studio | AI Website Builder, Managed Hosting, WhatsApp/FB Chatbots, MCP Bulk Emailer |
| **Trust Vector** | Privacy-first, zero code logging, TLS streaming, low TTFT | All-in-one simplicity, no-code visual flow, unified inbox management |
| **Marketing Angle** | "No $20/mo lock-in. Pay per token for Claude, GPT, Grok & Kling at 50-80% off official." | "Replace 5 expensive subscriptions (Hosting, Webflow, ManyChat, Mailchimp) in one dashboard." |
| **Internal Synergy** | Acts as the upstream wholesale token engine | Consumes tokens via internal service account, creating baseline volume |

---

## 3. International GTM & Acquisition Matrix

### 3.1 Developer Hub (`toraapi.com`)
1. **VS Code Marketplace**:
   - Optimize title and tags: `Pay-as-you-go Copilot alternative`, `Claude 3.5`, `Grok`, `Multi-LLM`.
   - Embed starter incentive: "$1-$2 Free Trial Credits on GitHub Sign-In".
2. **Hacker News (Show HN)**:
   - Frame strictly as a technical, transparent tool: "Show HN: A Pay-as-you-go VS Code AI Assistant with Claude, GPT & Grok".
   - Highlight zero-knowledge BYOK option and transparent cost-per-prompt ledger.
3. **Targeted Subreddits**:
   - `r/vscode`, `r/SideProject`, `r/LocalLLaMA`, `r/programming`.
   - Focus on solving subscription fatigue for part-time coders.

### 3.2 Business Suite (`createai.site`)
1. **AppSumo & Lifetime/Launch Deals**:
   - Launch an introductory tiered deal to rapidly acquire 1,000+ international paying customers.
2. **Product Hunt**:
   - Launch under *No-Code / Marketing Automation / AI Website Builders*.
3. **E-Commerce & Dropshipping Communities**:
   - Target Shopify and TikTok Shop sellers with WhatsApp auto-inbox and short-form video generation.

---

## 4. Supply Chain Resilience & Anti-Fragility

1. **Vendor Agnosticism**:
   - Never build hard dependency on a single upstream provider (e.g. Modelflare, fal.ai, WaveSpeed, KIE).
   - Use dynamic multi-candidate routing with automatic health checks and latency benchmarks.
2. **Sponsorship & Grant Maximization**:
   - Leverage aggregated token demand from both sites to negotiate custom tiered volume pricing.
   - Secure upfront sandbox credits to validate latency from SEA/US/EU before production traffic routing.

---

## 5. Compliance, Security & Abuse Prevention

1. **Anti-Spam & Deliverability Guardrails**:
   - Enforce CAN-SPAM, GDPR, and Meta Business verification policies on `createai.site`.
   - Never allow unverified domains or anonymous accounts to trigger bulk mail loops.
2. **Prompt & Code Data Integrity**:
   - Guarantee zero model retraining on user inputs on `toraapi.com`.
   - Explicitly publish a clear Data Processing Agreement (DPA) and Privacy Policy.
