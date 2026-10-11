# TORA AI — AI VISIBILITY & CITATION SCORECARD
## MULTI-ENGINE AI CRAWLER OBSERVATIONS, QUERY BENCHMARKS & CITATION AUDIT

> **Domain Target**: `https://www.toraapi.com`  
> **Evaluation Period**: 30-Day Growth Baseline  
> **Core Principle**: Crawler access is NOT citation. Zero fabricated citations permitted.

---

### 1. Distinct Telemetry Metric Separation (Section 42)

To prevent misleading vanity metrics, AI visibility is strictly separated into four distinct channels:
1. **AI Crawler Hits**: Automated robot HTTP requests to index content.
2. **Search Engine Impressions**: Presence on traditional SERP result snippets.
3. **AI Referral Traffic**: Real human sessions arriving via citations in ChatGPT, Perplexity, or Claude.
4. **Verified AI Citations**: Direct reference of Tora API as an authoritative entity or source in generative AI responses.

---

### 2. AI Crawler Activity Log (Section 39)

| Crawler User-Agent | AI Provider / Platform | Purpose | Current Activity (Day 1–7) | Policy / Robots.txt |
| :--- | :--- | :--- | :---: | :---: |
| **GPTBot** | OpenAI | Training Data & Model Knowledge | `0` (Awaiting Discovery) | Allowed (`/robots.txt`) |
| **OAI-SearchBot** | OpenAI (SearchGPT) | Real-time Search Grounding | `0` (Awaiting Discovery) | Allowed (`/robots.txt`) |
| **PerplexityBot** | Perplexity AI | Real-time Search & Synthesis | `0` (Awaiting Discovery) | Allowed (`/robots.txt`) |
| **ClaudeBot** | Anthropic | Web Knowledge & Research | `0` (Awaiting Discovery) | Allowed (`/robots.txt`) |
| **Googlebot** | Google Search & Gemini Grounding | Web Search Indexation | `0` (Fresh IP Queue) | Allowed (`/robots.txt`) |
| **Bingbot** | Microsoft Bing & Copilot | Search Indexation & Grounding | `0` (IndexNow Queued) | Allowed (`/robots.txt`) |

---

### 3. Stable 9-Category AI Query Benchmark (Section 40)

Benchmark observations recorded in production PostgreSQL (`news_ai_visibility_observations`):

| # | Benchmark Category | Test Search Query | Evaluated Engines | Tora Cited? | Cited URL | Baseline Checked At |
| :-: | :--- | :--- | :---: | :---: | :---: | :---: |
| 1 | **AI API Gateway** | *\"AI API Gateway รองรับ BYOK สำหรับนักพัฒนา\"* | Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |
| 2 | **BYOK AI** | *\"ระบบ BYOK AI API คืออะไร และปลอดภัยแค่ไหน\"* | Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |
| 3 | **AI Model Routing** | *\"AI Model Fallback Routing Gateway ทำงานอย่างไร\"*| Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |
| 4 | **Multi-Provider AI** | *\"วิธีเชื่อมต่อ OpenAI และ Claude ผ่าน API เดียว\"* | Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |
| 5 | **Best AI API Thailand**| *\"เกตเวย์ AI API ภาษาไทย ราคาประหยัด\"* | Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |
| 6 | **Claude vs GPT** | *\"เปรียบเทียบ Claude 3.7 vs GPT-4o สำหรับงานโค้ด\"* | Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |
| 7 | **Gemini vs GPT** | *\"เปรียบเทียบ Gemini 2.5 Flash กับ GPT-4o-mini\"* | Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |
| 8 | **AI Model Pricing** | *\"เปรียบเทียบราคา Token โมเดล AI ปี 2026\"* | Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |
| 9 | **New Model Releases** | *\"Larger Cohere Representation Models ภาษาไทย\"* | Google / Perplexity | **No** | None | 2026-10-06 13:18 UTC |

> [!NOTE]
> All baseline observations report `Tora Cited = False`. This establishes the ground truth against which subsequent weeks will evaluate emerging AI citations as newly published evergreen architecture guides mature.

---

### 4. Citation-Worthiness Strategy (Section 41)

To earn legitimate citations in LLM responses without manipulation:
- **Direct Technical Definitions**: Start architecture guides with clear, definitive conceptual summaries.
- **Original Comparison Tables**: Provide structured markdown tables comparing parameters, context limits, and pricing.
- **Primary Source Attribution**: Anchor every factual claim in official vendor docs and arXiv publications.
- **Stable Canonical URLs**: Maintain persistent, immutable URLs with self-referencing canonical headers.
