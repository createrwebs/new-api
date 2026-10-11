# Tora AI — Editorial Style Guide & Content Policy

**Document**: `TORA_EDITORIAL_STYLE.md`  
**Purpose**: Authoritative guideline for AI-generated and human-edited technical news, developer updates, and SEO articles published on Tora AI (`https://api.tora.ai/news`).

---

## 1. Core Editorial Philosophy

Tora AI exists to empower software engineers, AI developers, and tech builders in Thailand and Southeast Asia. Our news is:
- **Developer-First**: We care about APIs, token pricing, context lengths, SDK changes, and latency, not corporate PR jargon.
- **Natural, Professional Thai**: Written in modern, natural technical Thai with standard English technical terms preserved where appropriate (e.g. `Context Window`, `Token`, `Inference`, `Function Calling`, `Embeddings`, `Prompt Caching`).
- **Fact-Checked & Sourced**: Every material claim must link directly to the primary announcement or research paper.
- **Anti-Clickbait**: Headlines summarize the actual event. We never use vague curiosity gaps like "สิ่งนี้จะเปลี่ยนโลกไปตลอดกาล..." or "เปิดตัวแล้วสุดอึ้ง!".
- **Actionable**: Every article must explain: *What does this mean for developers? How do I use this in my code?*

---

## 2. Article Structure Standard

Every published article must follow this layout:

### 2.1 Headline (พาดหัว)
- Concise, factual, informative.
- Example: **OpenAI เปิดตัว GPT-4.5: รองรับ 128k Context พร้อม Prompt Caching ลดค่าใช้จ่ายลง 50%**
- Bad: *OpenAI ปล่อยของใหม่อีกแล้ว มาดูกันว่าจะเจ๋งแค่ไหน!*

### 2.2 Quick Take (สรุปสั้นใน 1 ประโยค)
- Highlighted lead paragraph giving the core essence in under 25 words.

### 2.3 What Changed (สาระสำคัญทางเทคนิค)
- Key parameters: Context window, pricing per 1M tokens (Input/Output), benchmark comparison (MMLU, HumanEval, Math), multimodal capabilities.
- Concrete bullet points.

### 2.4 Developer & Thailand Impact (ผลกระทบต่อนักพัฒนาไทย)
- Comparison of pricing and latency when accessing from Thailand / Southeast Asia.
- Availability on Tora Managed API vs. Direct API.
- Migration instructions: changes to model parameters or request format.

### 2.5 Code Example (ตัวอย่างการเรียกใช้งาน)
- Provide practical code snippet (cURL, Python `openai-python`, or Dart/Flutter).

### 2.6 Verified Source Attribution (แหล่งอ้างอิงต้นฉบับ)
- Markdown links to primary announcement (e.g. OpenAI Blog, Anthropic Research, GitHub Release).

---

## 3. Thai Language & Terminology Guidelines

| English Concept | Preferred Thai Usage | Avoid |
| :--- | :--- | :--- |
| Token | โทเค็น (Token) | คำ / สัญลักษณ์ย่อย |
| Prompt Caching | แคชคีย์ / Prompt Caching | การจำคำสั่ง |
| Inference | การประมวลผลโมเดล / Inference | การอนุมาน (overly academic) |
| Latency | ความหน่วง / Latency (TTFT) | ความช้า |
| Function Calling | การเรียกฟังก์ชัน / Function Calling | การเรียกคำสั่งภายนอก |
| Context Window | ความยาวบริบท (Context Window) | หน้าต่างบริบท |
| Rate Limit | ขีดจำกัดการเรียกใช้งาน (Rate Limit / RPM / TPM) | การจำกัดความเร็ว |

---

## 4. Content Risk & Fact-Checking Framework

### Level 1: LOW RISK (Auto-Publish Permitted)
- Official model releases with public API documentation and clear pricing.
- SDK updates and minor library versions.
- Standard benchmark publications from official research labs.

### Level 2: MEDIUM RISK (Editorial Corroboration Required)
- Third-party benchmarks claiming model superiority.
- Pricing analysis or price drops across competing platforms.
- Performance comparisons requiring multi-source corroboration.

### Level 3: HIGH RISK (Manual Operator Approval Required)
- Security vulnerabilities, leaks, API key exposure reports.
- Legal, regulatory, or policy sanctions.
- Unconfirmed leaks or rumors from unofficial social accounts.
- Any claim involving disputed commercial pricing.

---

## 5. SEO & Structured Data Rules

- **Title Tag**: Under 65 characters, includes primary keyword (e.g. `GPT-4.5`, `Claude 3.7 Sonnet`, `DeepSeek V3`).
- **Meta Description**: 130–155 characters summarizing the developer benefit.
- **Canonical URL**: `https://api.tora.ai/news/{slug}`.
- **Schema.org**: Valid `NewsArticle` JSON-LD with `datePublished`, `author`, `publisher`, `headline`, and `image`.
- **Internal Links**: Must link naturally to Tora product pages (`/pricing`, `/v1/models`, `/settings/byok`) without keyword stuffing.
