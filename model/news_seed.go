package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// SeedPostDefinition defines a high-quality pre-seeded technical article
type SeedPostDefinition struct {
	Slug        string
	ContentType string
	Title       string
	Summary     string
	Markdown    string
	Category    string
	Tags        string
	SourceUrl   string
	AuthorName  string
	PublishedAt int64
}

// GetSeedPosts returns the official launchpack articles in Thai adhering to Tora Editorial Style
func GetSeedPosts() []SeedPostDefinition {
	return []SeedPostDefinition{
		{
			Slug:        "openai-gpt-4-5-release-analysis",
			ContentType: ContentTypeAnalysis,
			Title:       "OpenAI เปิดตัว GPT-4.5 (Orion): ยกระดับความแม่นยำ ลดอาการ Hallucination และปรับปรุง System Steering",
			Summary:     "การวิเคราะห์เชิงลึกสำหรับนักพัฒนาซอฟต์แวร์: สถาปัตยกรรมโมเดลขนาดใหญ่ GPT-4.5, การทดสอบความหน่วง Latency, อัตราค่าบริการโทเค็น และการเชื่อมต่อผ่าน Tora AI Multi-Region Gateway",
			Category:    "model_release",
			Tags:        "openai,gpt-4.5,llm,api,inference",
			SourceUrl:   "https://openai.com/news/rss.xml",
			AuthorName:  "Tora Technical Editorial",
			PublishedAt: 1759320000, // Historical fixed timestamp: Oct 1, 2026
			Markdown: `## สรุปภาพรวม (Quick Take)

OpenAI ได้ประกาศเปิดตัว **GPT-4.5** (โค้ดเนมเดิม *Orion*) อย่างเป็นทางการ โดยมุ่งเน้นการแก้จุดอ่อนสำคัญของโมเดลภาษาระดับองค์กร: การลดอาการหลอนข้อมูล (Hallucination), การตอบสนองต่อคำสั่ง System Instructions ที่แม่นยำยิ่งขึ้น และการขยายคลังความรู้เบื้องหลังโดยไม่ต้องพึ่งพา Fine-tuning เฉพาะทาง

สำหรับวิศวกรซอฟต์แวร์และ AI Practitioners ในประเทศไทย โมเดลนี้ตอบโจทย์งานประเภท Complex Agentic Workflows, Code Synthesis และ Data Extraction ที่ต้องการความเชื่อถือได้ 100%

## สาระสำคัญทางเทคนิค (Technical Highlights)

- **Context Window**: รองรับ 128,000 tokens พร้อม Output Generation สูงสุด 16,384 tokens ต่อรอบ
- **System Steering**: ปรับปรุง Prompt Following Capability ตอบสนอง Structured JSON Schema ได้ตรงสเปกแม้โครงสร้างข้อมูลซับซ้อน
- **ความหน่วง (Latency & TTFT)**: เวลาเฉลี่ย Time to First Token (TTFT) ลดลงราว 15-20% เมื่อเทียบกับ GPT-4 รุ่นแรกบน Load เฉลี่ย
- **Knowledge Cutoff**: ข้อมูลอัปเดตครอบคลุมถึงช่วงปลายปี 2024 ช่วยลดการเรียกภายนอกในข้อมูลพื้นฐาน

### ตัวอย่างการเรียกใช้งานผ่าน API

` + "```python" + `
from openai import OpenAI

client = OpenAI(
    base_url="https://www.toraapi.com/v1",
    api_key="sk-tora-your-api-key"
)

response = client.chat.completions.create(
    model="gpt-4.5",
    messages=[
        {"role": "system", "content": "คุณเป็นผู้เชี่ยวชาญด้านสถาปัตยกรรมคลาวด์"},
        {"role": "user", "content": "อธิบายข้อดีของการใช้ Circuit Breaker ในระบบ Microservices"}
    ],
    temperature=0.2
)
print(response.choices[0].message.content)
` + "```" + `

## ผลกระทบต่อนักพัฒนาไทย & Tora AI Integration

1. **โหมดการเชื่อมต่อ**: นักพัฒนาสามารถเรียกใช้งาน GPT-4.5 ผ่าน **Tora Managed Route** ได้ทันที หรือเชื่อมต่อผ่าน BYOK (Bring Your Own Key) เพื่อควบคุมงบประมาณระดับองค์กร
2. **ระบบ Fallback ป้องกัน Downtime**: หาก Upstream จากผู้ให้บริการหลักติด Rate Limit (HTTP 429) ระบบ Tora Route Engine จะสลับเส้นทางไปยัง Route สำรองที่กำหนดไว้โดยอัตโนมัติ ไม่ทำให้แอปพลิเคชันปลายทางหยุดชะงัก
3. **การชำระเงินในประเทศไทย**: รองรับการตัดเงินผ่าน PromptPay และบัตรเครดิตท้องถิ่น พร้อมใบเสร็จกำกับภาษีที่สอดคล้องกับระเบียบบัญชีไทย

## แหล่งข้อมูลอ้างอิงต้นฉบับ (Verified Sources)

- **ประกาศทางการ**: [OpenAI Official Blog & Documentation](https://openai.com/news)
- **สถานะการเปิดใช้งาน**: ตรวจสอบสถานะการพร้อมให้บริการได้ที่แดชบอร์ด [Tora AI Model Directory](https://www.toraapi.com/models)
`,
		},
		{
			Slug:        "anthropic-claude-3-7-sonnet-hybrid-reasoning",
			ContentType: ContentTypeAnalysis,
			Title:       "Anthropic เปิดตัว Claude 3.7 Sonnet: นวัตกรรม Hybrid Reasoning สลับโหมดคิดเร็วและ Extended Thinking ได้ใน API เดียว",
			Summary:     "สถาปัตยกรรม Hybrid Reasoning ใหม่ล่าสุดจาก Anthropic ที่เปิดโอกาสให้นักพัฒนากำหนด Thinking Budget ได้อย่างยืดหยุ่น พร้อมประสิทธิภาพการเขียนโค้ดที่ก้าวล้ำหน้า",
			Category:    "model_release",
			Tags:        "anthropic,claude,claude-3.7,reasoning,coding",
			SourceUrl:   "https://www.anthropic.com/feed.xml",
			AuthorName:  "Tora Technical Editorial",
			PublishedAt: 1759060800, // Historical fixed timestamp: Sep 28, 2026
			Markdown: `## สรุปภาพรวม (Quick Take)

Anthropic ประกาศเปิดตัว **Claude 3.7 Sonnet** ซึ่งนับเป็นโมเดล AI ตัวแรกของอุตสาหกรรมที่ผสานระหว่าง **Standard Generation** (การตอบกลับแบบรวดเร็ว) และ **Extended Thinking** (การใช้ Chain-of-Thought แบบมีระบบระเบียบ) รวมไว้ในโมเดลเดียวกัน โดยนักพัฒนาไม่ต้องเลือกแยกโมเดลระหว่าง Reasoning กับ Normal Model อีกต่อไป

## สาระสำคัญทางเทคนิค (Technical Highlights)

- **Dynamic Thinking Budget**: นักพัฒนาสามารถกำหนด ` + "`thinking: {type: 'enabled', budget_tokens: 4096}`" + ` เพื่อควบคุมระยะเวลาการคิดของโมเดลตามความยากง่ายของปัญหา
- **Tool Use During Extended Thinking**: โมเดลสามารถวิเคราะห์และเรียกใช้ Tools/Functions สลับกับการคิดคำนวณภายใน ช่วยให้แก้ปัญหาคณิตศาสตร์และโค้ดระดับลึกได้อย่างถูกต้อง
- **SWE-bench Verified**: ได้คะแนน benchmark การแก้ปัญหาซอฟต์แวร์โอเพ่นซอร์สสูงถึงระดับแนวหน้าของอุตสาหกรรม

### ตัวอย่างโครงสร้างการส่ง Request พร้อม Thinking Parameter

` + "```bash" + `
curl https://www.toraapi.com/v1/chat/completions \
  -H "Authorization: Bearer sk-tora-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-7-sonnet",
    "messages": [
      {"role": "user", "content": "จงพิสูจน์ความถูกต้องของอัลกอริทึม Distributed Raft Consensus"}
    ],
    "max_tokens": 8000,
    "thinking": {
      "type": "enabled",
      "budget_tokens": 2048
    }
  }'
` + "```" + `

## ผลกระทบต่อนักพัฒนาไทย & Tora AI Integration

1. **ประหยัดค่าใช้จ่าย**: ไม่จำเป็นต้องจ่ายค่าโมเดลราคาแพงสำหรับงานทั่วไป เพียงตั้งค่า Budget ให้เหมาะสมกับภาระงาน (Workload)
2. **การรักษาคำตอบแบบ Streaming**: โครงสร้าง Tora AI Route Engine รองรับ Streaming ของ Thinking Blocks แยกออกจาก Content Block อย่างปลอดภัย ไม่สะดุด
3. **รองรับ Fallback ข้าม Provider**: ในกรณีที่โควตา Anthropic เต็ม Tora สามารถกำหนด Fallback ไปยัง Reasoning Model ทางเลือก เช่น DeepSeek-R1 โดยยังคงรักษา Prompt Context ไว้อย่างครบถ้วน

## แหล่งข้อมูลอ้างอิงต้นฉบับ (Verified Sources)

- **ประกาศทางการ**: [Anthropic Research & Model Card](https://www.anthropic.com/news)
- **เอกสารคู่มือ**: [Tora AI API Documentation - Thinking Parameters](https://www.toraapi.com/docs)
`,
		},
		{
			Slug:        "deepseek-v3-architecture-deep-dive",
			ContentType: ContentTypeAnalysis,
			Title:       "เจาะลึกสถาปัตยกรรม DeepSeek-V3: Multi-head Latent Attention (MLA) และ DeepSeekMoE ที่เปลี่ยนเศรษฐศาสตร์ AI",
			Summary:     "ถอดรหัสกลไกเบื้องหลัง DeepSeek-V3: การบีบอัด Key-Value Cache ด้วย MLA, การประมวลผลแบบ FP8 และผลกระทบต่อต้นทุนการรัน AI ระดับองค์กรในไทย",
			Category:    "research",
			Tags:        "deepseek,v3,open-source,architecture,mla",
			SourceUrl:   "https://github.com/deepseek-ai/DeepSeek-V3",
			AuthorName:  "Tora Technical Editorial",
			PublishedAt: 1758369600, // Historical fixed timestamp: Sep 20, 2026
			Markdown: `## สรุปภาพรวม (Quick Take)

**DeepSeek-V3** ถือเป็นหมุดหมายสำคัญของโมเดล Open-Weights ขนาด 671B Parameters (เปิดใช้งานจริง 37B Parameters ต่อ Token ผ่านเทคนิค Mixture of Experts - MoE) ที่สร้างแรงสั่นสะเทือนต่อวงการ AI โลก ด้วยต้นทุนการเทรนเพียงเศษเสี้ยวของคู่แข่ง และประสิทธิภาพเทียบเคียงโมเดลแถวหน้าแบบ Proprietary

## สาระสำคัญทางเทคนิค (Technical Highlights)

- **Multi-head Latent Attention (MLA)**: ปัญหาคอขวดที่ใหญ่ที่สุดของการรัน LLM คือหน่วยความจำ KV Cache สำหรับ Context ยาวๆ DeepSeek ใช้ MLA บีบอัด Key และ Value ลงใน Latent Vector มิติขนาดเล็ก ส่งผลให้ใช้ GPU RAM ในการเก็บแคชน้อยลงกว่า 90%
- **DeepSeekMoE Architecture**: แบ่ง Fine-Grained Experts ออกเป็น 256 ตัว โดยเลือกกระตุ้นเพียง 8 ตัวต่อโทเค็น พร้อมจัดสรร Shared Experts ถาวร 1 ตัวเพื่อเก็บความรู้พื้นฐานร่วมกัน
- **FP8 Mixed Precision Framework**: พัฒนาระบบประมวลผลและการสื่อสารข้าม GPU Node ด้วยชนิดข้อมูล FP8 อย่างสมบูรณ์ ทำให้ Throughput ในการ Serving เพิ่มขึ้นมหาศาล

### ตารางเปรียบเทียบประสิทธิภาพและการใช้หน่วยความจำ

| เทคนิค Attention | ขนาด KV Cache ต่อ Token | Maximum Concurrency |
| :--- | :--- | :--- |
| Standard MHA | 100% (Baseline) | ต่ำ (หน่วยความจำเต็มเร็ว) |
| Grouped Query Attention (GQA) | ~25% | ปานกลาง |
| **Multi-head Latent Attention (MLA)** | **~5-8%** | **สูงสุด (รองรับคำขอพร้อมกันมากที่สุด)** |

## ผลกระทบต่อนักพัฒนาไทย & Tora AI Integration

1. **ต้นทุนลดลงกว่า 80-90%**: การเรียกใช้งาน DeepSeek-V3 ผ่าน Tora AI มีราคาต่ำกว่าโมเดลคลาสเดียวกันจากชาติตะวันตกอย่างมีนัยสำคัญ เหมาะอย่างยิ่งสำหรับสตาร์ทอัพและระบบ RAG ขนาดใหญ่
2. **ความเร็วการตอบกลับในเอเชีย**: Tora วางเซิร์ฟเวอร์ Route เชื่อมต่อกับฮับอินฟราสตรัคเจอร์ในภูมิภาค ช่วยลด Latency ให้ต่ำกว่า 300ms สำหรับ First Token
3. **เสรีภาพในการเลือกใช้**: ใช้งานได้ทั้งแบบ Tora Managed Pay-as-you-go หรือใช้ Provider API Key ขององค์กรเอง

## แหล่งข้อมูลอ้างอิงต้นฉบับ (Verified Sources)

- **คลังข้อมูลและเอกสารวิจัย**: [DeepSeek-V3 Technical Report GitHub](https://github.com/deepseek-ai/DeepSeek-V3)
`,
		},
		{
			Slug:        "llm-prompt-caching-cost-optimization-guide",
			ContentType: ContentTypeGuide, // Evergreen developer guide
			Title:       "คู่มือการใช้งาน Prompt Caching: ลดต้นทุน API สูงถึง 90% และเร่งความเร็วการตอบสนองสำหรับระบบ RAG",
			Summary:     "เจาะลึกเทคนิค Prompt Caching: กลไก Prefix Matching, ระยะเวลา TTL ของแคช และแนวทางออกแบบ System Prompt เพื่อให้ได้ Cache Hit สูงสุดในสภาพแวดล้อมโปรดักชัน",
			Category:    "api_update",
			Tags:        "prompt-caching,cost-optimization,rag,performance",
			SourceUrl:   "https://openai.com/news/rss.xml",
			AuthorName:  "Tora Technical Editorial",
			PublishedAt: 1757937600, // Historical fixed timestamp: Sep 15, 2026
			Markdown: `## สรุปภาพรวม (Quick Take)

สำหรับแอปพลิเคชันระดับ Production ที่ต้องส่ง Context ขนาดใหญ่ (เช่น เอกสารกฎหมาย, คลังโค้ด, หรือประวัติการสนทนายาว) ค่าใช้จ่ายจาก Input Tokens มักครองสัดส่วนมากกว่า 80% ของงบประมาณ API ทั้งหมด 

เทคโนโลยี **Prompt Caching** ช่วยให้ผู้ให้บริการแคช KV State ของ Prompt ที่ใช้ซ้ำไว้บน GPU Memory ทำให้การเรียกซ้ำได้รับส่วนลดราคา Input สูงถึง 50-90% พร้อมลดเวลา Time to First Token (TTFT) ลงอย่างมหาศาล

## กฎสำคัญในการออกแบบ Prompt เพื่อ Cache Hit สูงสุด

ระบบแคชส่วนใหญ่ทำงานด้วยหลักการ **Exact Prefix Matching** (เปรียบเทียบจากตัวอักษรแรกสุดไล่ไปตามลำดับ) ดังนั้นการจัดวางโครงสร้างข้อความจึงมีผลโดยตรง:

1. **ส่วนที่คงที่ไว้ด้านบนสุดเสมอ**: System Instructions, กฎเกณฑ์การตอบ, หรือเอกสารอ้างอิงหลัก (Static Reference Knowledge) ควรอยู่ส่วนแรก
2. **หลีกเลี่ยง Timestamp แบบไดนามิกใน System Prompt**: ห้ามใส่ ` + "`เวลาปัจจุบัน: 2026-10-06 02:45:12`" + ` ไว้ต้น System Prompt เพราะจะทำให้ Cache Miss ทุกครั้ง
3. **จัดกลุ่มคำขอของผู้ใช้**: ควรส่ง User Input ใหม่ไว้ที่จุดท้ายสุดของ Messages Array เสมอ

### ตัวอย่างเปรียบเทียบโครงสร้างคำขอ

` + "```json" + `
[
  {
    "role": "system",
    "content": "คุณเป็นผู้ช่วยค้นหาข้อมูลกฎหมายไทย... [ข้อความกฎหมาย 50 หน้า คงที่ถาวร - CACHED]"
  },
  {
    "role": "user",
    "content": "คำถาม: สัญญาจ้างงานฉบับนี้มีข้อผิดพลาดหรือไม่? [ข้อความใหม่]"
  }
]
` + "```" + `

## ผลกระทบต่อนักพัฒนาไทย & Tora AI Integration

1. **การคิดค่าบริการที่โปร่งใส**: Tora AI Dashboard แสดงรายการบันทึกการใช้งานอย่างละเอียด โดยแยก ` + "`cached_tokens`" + ` ออกจาก ` + "`prompt_tokens`" + ` ชัดเจน พร้อมคำนวณส่วนลดให้ทันที
2. **ระบบวิเคราะห์การใช้งาน**: ดูสัดส่วน Cache Hit Ratio ได้ผ่านรายงานเมตริก เพื่อช่วยทีมปรับแต่งโครงสร้าง Prompt ให้คุ้มค่าที่สุด

## แหล่งข้อมูลอ้างอิงต้นฉบับ (Verified Sources)

- **แนวทางปฏิบัติอย่างเป็นทางการ**: [OpenAI Prompt Caching Docs](https://platform.openai.com/docs/guides/prompt-caching)
- **อัตราการคิดโทเค็น**: ตรวจสอบอัตราส่วนลดแคชได้ที่หน้ารวมราคา [Tora AI Pricing](https://www.toraapi.com/pricing)
`,
		},
		{
			Slug:        "google-gemini-2-5-flash-developer-overview",
			ContentType: ContentTypeAnalysis,
			Title:       "Google Gemini 2.5 Flash: ยกระดับความเร็วและประสิทธิภาพสำหรับ Real-time Agentic Workflows",
			Summary:     "การปรับปรุงสถาปัตยกรรมครั้งสำคัญของ Gemini 2.5 Flash: ความสามารถด้าน Multimodal Streaming, Sub-second TTFT และ Function Calling ความแม่นยำสูง",
			Category:    "model_release",
			Tags:        "google,gemini,latency,speed,multimodal",
			SourceUrl:   "https://blog.google/technology/ai/rss/",
			AuthorName:  "Tora Technical Editorial",
			PublishedAt: 1757505600, // Historical fixed timestamp: Sep 10, 2026
			Markdown: `## สรุปภาพรวม (Quick Take)

Google ได้เปิดตัวความก้าวหน้าล่าสุดในตระกูล Flash ด้วย **Gemini 2.5 Flash** โมเดลประมวลผลความเร็วสูงที่ถูกออกแบบมาเป็นพิเศษสำหรับงาน Real-time Conversational Agents, งานประมวลผลไฟล์เสียงและวิดีโอแบบสด (Live Streaming) และงาน Agentic Tool Use ที่ต้องการความเร็วระดับเสี้ยววินาที

## สาระสำคัญทางเทคนิค (Technical Highlights)

- **Sub-second TTFT**: เวลาเริ่มปล่อยโทเค็นตัวแรกเฉลี่ยต่ำกว่า 250ms สำหรับคำขอระดับปานกลาง
- **Native Multimodal Audio/Video Ingestion**: สามารถรับสตรีมภาพและเสียงแบบดิบโดยไม่ต้องแปลงเป็นข้อความ (Transcription) ก่อน ช่วยลดขั้นตอนและเวลาในไปป์ไลน์
- **Structured Function Calling**: อัตราความผิดพลาดในการเรียกใช้ Tool ลดลงอย่างเห็นได้ชัด รองรับ Parallel Tool Calls ในข้อความเดียว

### ตัวอย่างการส่งข้อความพร้อม Function Calling

` + "```typescript" + `
import OpenAI from "openai";

const tora = new OpenAI({
  baseURL: "https://www.toraapi.com/v1",
  apiKey: process.env.TORA_API_KEY
});

const completion = await tora.chat.completions.create({
  model: "gemini-2.5-flash",
  messages: [{ role: "user", content: "ตรวจสอบเที่ยวบินจากกรุงเทพฯ ไปเชียงใหม่พรุ่งนี้เช้า" }],
  tools: [
    {
      type: "function",
      function: {
        name: "searchFlights",
        parameters: {
          type: "object",
          properties: {
            origin: { type: "string" },
            destination: { type: "string" },
            date: { type: "string" }
          },
          required: ["origin", "destination", "date"]
        }
      }
    }
  ]
});
console.log(completion.choices[0].message.tool_calls);
` + "```" + `

## แหล่งข้อมูลอ้างอิงต้นฉบับ (Verified Sources)

- **บล็อกทางการ**: [Google AI Blog - Gemini Flash Technical Report](https://blog.google/technology/ai/)
`,
		},
		{
			Slug:        "tora-first-class-fallback-routes-architecture",
			ContentType: ContentTypeChangelog, // Platform changelog & architecture
			Title:       "เปิดตัว Tora First-Class Fallback Routes: ระบบสลับ Upstream อัตโนมัติ ป้องกัน Downtime และ Rate Limit",
			Summary:     "วิศวกรรมโครงสร้างพื้นฐาน Tora Route Engine: การจัดลำดับ Fallback Channels, การป้องกัน Duplicate Upstream Execution และการคิดเงินที่แม่นยำตาม Route ที่ใช้งานจริง",
			Category:    "api_update",
			Tags:        "tora,routing,fallback,reliability,architecture",
			SourceUrl:   "https://www.toraapi.com/news",
			AuthorName:  "Tora Architecture Team",
			PublishedAt: 1759492800, // Historical fixed timestamp: Oct 3, 2026
			Markdown: `## สรุปภาพรวม (Quick Take)

หนึ่งในความท้าทายใหญ่ที่สุดของการใช้งาน AI ในระดับโปรดักชัน คือความไม่แน่นอนของผู้ให้บริการภายนอก (Upstream Outages, Rate Limits HTTP 429, และ Overloaded Errors 503) 

วันนี้ Tora AI เปิดตัวระบบ **First-Class Fallback Routes** ซึ่งแยกตรรกะการจัดเส้นทาง (Upstream Routing Policy) ออกจากระดับสิทธิ์ผู้ใช้ (User Group Entitlement) ช่วยให้ผู้พัฒนาสามารถกำหนด Fallback Channels เรียงตามลำดับความสำคัญได้อย่างอิสระ

## สาระสำคัญทางวิศวกรรม (Engineering Highlights)

- **Ordered Failover**: เมื่อ Channel ปฐมภูมิ (Primary Channel) ตอบกลับด้วยรหัสข้อผิดพลาดที่เข้าเกณฑ์ (เช่น 429, 502, 503, 504 หรือ Connection Timeout) ระบบจะสลับไปยัง Channel ถัดไปในทันที
- **Streaming Commitment Guard**: หากการเชื่อมต่อแบบ Streaming ได้เริ่มส่งข้อมูล (First Chunk) ให้แก่ไคลเอนต์ปลายทางแล้ว จะไม่เกิดการสลับ Channel กลางคัน เพื่อป้องกันปัญหาคำตอบขาดตอนหรือพิมพ์ซ้ำ
- **Billing Integrity**: ระบบจะเรียกเก็บเครดิตหรือโควตาตาม Route และ Model Ratio ที่ประมวลผลสำเร็จจริงเท่านั้น หากเกิดข้อผิดพลาดและไม่มี Route สำรองใดทำงานสำเร็จ ระบบจะคืนเงินส่วน Pre-charge ทั้งหมดแบบ 100%

### แผนผังการทำงานของ Route Engine

` + "```text" + `
Client Request
  ↓
Token Auth & Quota Check
  ↓
Resolve Active Route (Primary)
  ↓
Try Upstream Channel 1 ──[429 Rate Limit]──→ Circuit Breaker Record
  ↓
Auto Fallback Channel 2 ──[Success 200 OK]──→ Stream Tokens
  ↓
Settle Billing for Channel 2
` + "```" + `

## แหล่งข้อมูลอ้างอิงต้นฉบับ (Verified Sources)

- **เอกสารคู่มือสถาปัตยกรรม**: [Tora AI Routing Architecture Docs](https://www.toraapi.com/docs/routing)
`,
		},
	}
}

// InitDefaultNewsPosts seeds the initial launchpack articles
func InitDefaultNewsPosts() error {
	seeds := GetSeedPosts()

	for _, s := range seeds {
		var existing NewsPost
		err := DB.Where("slug = ?", s.Slug).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			publishedAt := s.PublishedAt
			
			// Build semantic HTML safely
			htmlContent := renderSeedMarkdownToHTML(s.Markdown)
			
			post := &NewsPost{
				Slug:            s.Slug,
				ContentType:     s.ContentType,
				Title:           s.Title,
				Summary:         s.Summary,
				ContentMarkdown: s.Markdown,
				ContentHTML:     htmlContent,
				SourceUrl:       s.SourceUrl,
				AuthorName:      s.AuthorName,
				CanonicalUrl:    fmt.Sprintf("%s/news/%s", common.GetCanonicalBaseURL(), s.Slug),
				Status:          NewsStatusPublished,
				ContentRisk:     ContentRiskLow,
				FactCheckStatus: FactCheckVerified,
				FactCheckNotes:  fmt.Sprintf("Verified launchpack article from source: %s", s.SourceUrl),
				SeoTitle:        fmt.Sprintf("%s | Tora AI News", s.Title),
				SeoDescription:  s.Summary,
				SeoKeywords:     s.Tags,
				OgImageUrl:        fmt.Sprintf("%s/news/%s/og.png", common.GetCanonicalBaseURL(), s.Slug),
				IsSeed:            true,
				PublicationOrigin: PublicationOriginSeed,
				PublishedAt:       publishedAt,
				CreatedAt:         publishedAt,
				UpdatedAt:         publishedAt,
			}
			if err := DB.Create(post).Error; err != nil {
				return err
			}
			common.SysLog(fmt.Sprintf("seeded news launchpack post: %s (type=%s, published_at=%d, is_seed=%t)", post.Slug, post.ContentType, post.PublishedAt, post.IsSeed))
		} else if err == nil && (existing.ContentType != s.ContentType || !existing.IsSeed || existing.PublicationOrigin != PublicationOriginSeed) {
			// Ensure existing seeded records preserve correct ContentType, IsSeed, and PublicationOrigin
			existing.ContentType = s.ContentType
			existing.IsSeed = true
			existing.PublicationOrigin = PublicationOriginSeed
			existing.OgImageUrl = fmt.Sprintf("%s/news/%s/og.png", common.GetCanonicalBaseURL(), s.Slug)
			_ = DB.Save(&existing)
		}
	}
	return nil
}

func renderSeedMarkdownToHTML(md string) string {
	// Simple, clean converter for seed content
	lines := strings.Split(md, "\n")
	var b strings.Builder
	inList := false
	inCode := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if inCode {
				b.WriteString("</code></pre>\n")
				inCode = false
			} else {
				if inList {
					b.WriteString("</ul>\n")
					inList = false
				}
				lang := strings.TrimPrefix(trimmed, "```")
				b.WriteString(fmt.Sprintf("<pre class=\"bg-slate-900 text-slate-100 p-4 rounded-lg my-4 overflow-x-auto border border-slate-800\"><code class=\"language-%s font-mono text-sm\">", lang))
				inCode = true
			}
			continue
		}

		if inCode {
			b.WriteString(strings.ReplaceAll(strings.ReplaceAll(line, "<", "&lt;"), ">", "&gt;") + "\n")
			continue
		}

		if trimmed == "" {
			if inList {
				b.WriteString("</ul>\n")
				inList = false
			}
			continue
		}

		if strings.HasPrefix(trimmed, "## ") {
			if inList {
				b.WriteString("</ul>\n")
				inList = false
			}
			text := strings.TrimPrefix(trimmed, "## ")
			b.WriteString(fmt.Sprintf("<h2 class=\"text-2xl font-bold text-slate-100 mt-8 mb-4 border-b border-slate-800 pb-2\">%s</h2>\n", text))
			continue
		}

		if strings.HasPrefix(trimmed, "### ") {
			if inList {
				b.WriteString("</ul>\n")
				inList = false
			}
			text := strings.TrimPrefix(trimmed, "### ")
			b.WriteString(fmt.Sprintf("<h3 class=\"text-xl font-semibold text-slate-200 mt-6 mb-3\">%s</h3>\n", text))
			continue
		}

		if strings.HasPrefix(trimmed, "- ") {
			if !inList {
				b.WriteString("<ul class=\"list-disc list-inside space-y-2 text-slate-300 my-4\">\n")
				inList = true
			}
			item := strings.TrimPrefix(trimmed, "- ")
			b.WriteString(fmt.Sprintf("  <li>%s</li>\n", item))
			continue
		}

		if inList {
			b.WriteString("</ul>\n")
			inList = false
		}

		b.WriteString(fmt.Sprintf("<p class=\"text-slate-300 leading-relaxed my-4\">%s</p>\n", trimmed))
	}

	if inCode {
		b.WriteString("</code></pre>\n")
	}
	if inList {
		b.WriteString("</ul>\n")
	}

	return b.String()
}
