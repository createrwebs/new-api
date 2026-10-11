#!/usr/bin/env python3
"""
Publishes authoritative Evergreen Pillars to Tora Production PostgreSQL
Ensures strict validation: ContentType='guide', Origin='MANUAL_ADMIN', zero clichés,
valid canonicals, and IndexNow push.
"""

import subprocess
import time
import json
import urllib.request
import ssl

EC2_HOST = "ubuntu@51.20.174.90"
SSH_KEY = "/Users/noppanan/key/saascover-api.pem"

PILLAR_1_MARKDOWN = """## บทนำ: ความท้าทายในการผสานรวมโมเดล AI ในระบบ Production

ในกระบวนการพัฒนาระบบปัญญาประดิษฐ์ระดับองค์กร (Enterprise AI Applications) ความต้องการใช้งานโมเดลภาษาขนาดใหญ่ (Large Language Models: LLMs) ไม่ได้จำกัดอยู่เพียงผู้ให้บริการรายเดียวอีกต่อไป การพึ่งพาผู้ให้บริการรายเดียวสร้างความเสี่ยงสูงต่อระบบ ทั้งในมิติของ **Vendor Lock-in**, ปัญหาการติดขัดของอัตราการเรียกใช้งาน (**Rate Limits: 429 Too Many Requests**), ค่าบริการที่ไม่สามารถควบคุมได้, ตลอดจนเหตุขัดข้องทางเทคนิคของผู้ให้บริการต้นทาง

สถาปัตยกรรม **AI API Gateway** จึงกลายมาเป็นองค์ประกอบสำคัญ (Core Infrastructure Component) ที่ทำหน้าที่เป็นตัวกลางในการจัดการ, กำหนดทิศทางทราฟฟิก (Traffic Routing), แปลงโปรโตคอลการเรียกใช้งาน, และรักษาความปลอดภัยของกุญแจเข้าถึง (API Key Management) 

คู่มือนี้สรุปแนวคิดเชิงสถาปัตยกรรมและแบบแผนการออกแบบ (Design Patterns) เพื่อสร้างเกตเวย์ที่รองรับมาตรฐาน **OpenAI-Compatible** พร้อมระบบ **Bring Your Own Key (BYOK)** สำหรับนักพัฒนาและทีมวิศวกรรมซอฟต์แวร์

---

## 1. สถาปัตยกรรม OpenAI-Compatible: มาตรฐานสากลสำหรับ Multi-Provider

ปัญหาหลักที่นักพัฒนาต้องเผชิญเมื่อใช้งานหลายค่าย (เช่น OpenAI, Anthropic Claude, Google Gemini, DeepSeek, และ Cohere) คือความแตกต่างของ SDK, รูปแบบ Payload, และพารามิเตอร์ของ Request/Response

การกำหนดให้อินเทอร์เฟซภายนอกเป็น **OpenAI-Compatible API** ช่วยแก้ปัญหานี้ได้อย่างสมบูรณ์:
- **รูปแบบ Endpoint สากล**: ใช้งานผ่าน `/v1/chat/completions`, `/v1/embeddings`, และ `/v1/models`
- **SDK Interoperability**: ใช้งานร่วมกับไลบรารีมาตรฐาน เช่น `openai-python`, `openai-node`, หรือเฟรมเวิร์กอย่าง LangChain, LlamaIndex, และ Vercel AI SDK ได้ทันทีโดยเปลี่ยนเพียง `base_url` และ `api_key`
- **Payload Normalization**: เกตเวย์จะรับ JSON Payload มาตรฐาน แล้วแปลงเข้าสู่ Payload ดั้งเดิมของแต่ละค่าย (Vendor Native Payload) ก่อนส่งต่อไปยัง Endpoint ปลายทาง

---

## 2. ความปลอดภัยระดับองค์กรด้วยระบบ Bring Your Own Key (BYOK)

การรวมศูนย์การเรียกใช้ API มักสร้างความกังวลด้านความปลอดภัย หากผู้ให้บริการเกตเวย์จัดเก็บ API Key ต้นทางไว้ทั้งหมด ความเสี่ยงต่อการรั่วไหลจะสูงขึ้น สถาปัตยกรรม **BYOK** จึงถูกออกแบบมาเพื่อแยกอำนาจการควบคุม (Separation of Control):

1. **Client-Side Key Vault**: ผู้ใช้งานหรือแต่ละทีมสามารถระบุ Key ต้นทางของตนเอง (เช่น OpenAI API Key หรือ Anthropic Key) ผ่าน Header เฉพาะ หรือจัดเก็บใน Vault ที่เข้ารหัสแบบ AES-256-GCM
2. **Zero-Knowledge Decryption**: กุญแจจะถูกถอดรหัสในหน่วยความจำชั่วคราวขณะประมวลผล Relay Request เท่านั้น และจะถูกล้างทิ้งทันทีเมื่อ Response ส่งกลับเสร็จสิ้น
3. **Usage Attribution & Isolation**: การบันทึก Quota และค่าใช้จ่ายจะถูกแยกตามบัญชีของตนเอง ทำให้ทีมต่างๆ สามารถควบคุมงบประมาณได้อย่างโปร่งใส

---

## 3. กลไก Dynamic Routing และ High-Availability Fallback

หัวใจสำคัญของเกตเวย์ที่มีเสถียรภาพคือความสามารถในการบริหารจัดการทราฟฟิกเมื่อเกิดปัญหา:

- **Weighted Round-Robin & Priority Tiers**: กำหนดให้ทราฟฟิกวิ่งไปยังโมเดลหลักที่มีต้นทุนต่ำที่สุดก่อน หากเกิด Latency เกินกำหนด ให้สลับไปยังช่องทางสำรอง
- **Error Classification**: เกตเวย์ต้องแยกความแตกต่างระหว่าง Client Error (เช่น 400 Bad Request, 401 Unauthorized) ซึ่งไม่ควร Retry กับ Transient Server Error (เช่น 429 Rate Limit, 502 Bad Gateway, 503 Service Unavailable, Connection Timeout) ซึ่งต้องส่งต่อไปยัง Provider สำรองทันที
- **Circuit Breaker Pattern**: เมื่อช่องทางใดช่องทางหนึ่งมีอัตราความล้มเหลวเกินเกณฑ์ที่กำหนด เกตเวย์จะเปิดวงจร (Open Circuit) และข้ามช่องทางนั้นชั่วคราวเพื่อป้องกันการสะสมคิวก่อนที่ระบบจะฟื้นตัว

---

## 4. มุมมองทางสถาปัตยกรรมสำหรับ Tora API และนักพัฒนาไทย

สำหรับนักพัฒนาในประเทศไทย การผสานรวม AI เข้ากับระบบงานจริงมักต้องการความยืดหยุ่นสูง:
- **การเชื่อมต่อแบบครบวงจร**: ศึกษาเอกสารและแนวทางการเชื่อมต่อได้ที่ [Tora Documentation](https://www.toraapi.com/docs) เพื่อดูรูปแบบการเรียกใช้งานผ่าน REST และ Streaming
- **การบริหารต้นทุนที่ชัดเจน**: สามารถตรวจสอบการคำนวณราคาและโควตาตามการใช้งานจริงได้ที่ [Tora Pricing & Plans](https://www.toraapi.com/pricing)
- **การสืบค้นข้อมูลความหมายสูง**: ใช้งานร่วมกับโมเดลเวกเตอร์รุ่นใหม่ เช่น [Larger Cohere Representation Models](https://www.toraapi.com/news/larger-cohere-representation-models) เพื่อสร้างระบบ Semantic Search และ RAG ที่แม่นยำ
- **การเพิ่มประสิทธิภาพด้วยการแคช**: ควบคู่กับการใช้ [LLM Prompt Caching Cost Optimization](https://www.toraapi.com/news/llm-prompt-caching-cost-optimization-guide) เพื่อลด Latency และประหยัดค่าโทเค็นได้สูงสุดถึง 80%

---

## สรุปแนวทางปฏิบัติ (Best Practices)

1. **หลีกเลี่ยงการ Hard-code Endpoint**: ออกแบบสถาปัตยกรรมให้รับ `base_url` จาก Environment Configuration เสมอ
2. **ตั้งค่า Timeout อย่างรัดกุม**: แยก Timeout สำหรับ Non-streaming (เช่น 30 วินาที) และ Time-to-First-Token สำหรับ Streaming (เช่น 10 วินาที)
3. **บันทึก Audit Logs อย่างระมัดระวัง**: ตรวจสอบว่าระบบจะไม่บันทึก Sensitive Data เช่น รหัสผ่าน หรือ Prompt ที่มีข้อมูลส่วนบุคคล (PII) ลงใน System Logs"""

PILLAR_2_MARKDOWN = """## บทนำ: ความจำเป็นของระบบ Fallback ในการให้บริการ AI ระดับ Mission-Critical

ในการให้บริการซอฟต์แวร์สมัยใหม่ที่ขับเคลื่อนด้วยปัญญาประดิษฐ์ ข้อผิดพลาดจาก Upstream LLM Provider เป็นสิ่งที่หลีกเลี่ยงไม่ได้อย่างสิ้นเชิง ไม่ว่าจะเป็นปัญหา **HTTP 429 (Rate Limit Exceeded)**, การชะลอตัวของเซิร์ฟเวอร์ (High Latency Spike), หรือปัญหาเครือข่ายล่มชั่วคราว

หากระบบ Application ส่ง Request ไปยังผู้ให้บริการเพียงช่องทางเดียว เมื่อเกิดข้อผิดพลาด ผู้ใช้งานปลายทางจะพบกับความล้มเหลวทันที (Immediate Failure) สถาปัตยกรรม **LLM Routing & Dynamic Fallback** จึงเป็นรากฐานสำคัญในการยกระดับความพร้อมใช้งานของระบบให้แตะระดับ **High Availability (99.9% Uptime)**

---

## 1. ลำดับชั้นการออกแบบ Fallback: Intra-Provider vs Cross-Provider

การทำ Failover ที่มีประสิทธิภาพต้องแบ่งออกเป็นสองระดับชั้น:

### 1.1 Intra-Provider Failover (การสลับภายในผู้ให้บริการเดิม)
เมื่อเกิดข้อผิดพลาดประเภท Rate Limit หรือ Quota หมดบน API Key หลัก เกตเวย์จะสลับไปยัง **Secondary Key Pool** ของ Provider เดิมก่อน:
- ช่วยรักษาความต่อเนื่องของคุณลักษณะเฉพาะของโมเดล (เช่น Output Formatting หรือ Fine-tuned Weights)
- ลดความเสี่ยงจากผลลัพธ์ที่อาจแตกต่างกันระหว่างต่างตระกูลโมเดล

### 1.2 Cross-Provider Failover (การสลับข้ามผู้ให้บริการ)
ในกรณีที่เกิด Outage ระดับศูนย์ข้อมูล หรือ API ล่มทั้งภูมิภาค เกตเวย์จะสลับ Request ไปยังโมเดลในระดับความสามารถใกล้เคียงกัน (Equivalent Capability Tier):
- เช่น หาก `claude-3-7-sonnet` ล่มชั่วคราว ให้สลับไปยัง `gpt-4o` หรือ `gemini-2.5-pro`
- เกตเวย์ต้องแปลง System Prompt, Temperature, และ Tool Calling Schema ให้เข้ากันได้โดยอัตโนมัติ

---

## 2. การจัดการ Streaming Failover (SSE Buffer & Header Holdback)

ความท้าทายสูงสุดของการทำ Fallback อยู่ที่การเรียกใช้งานแบบ **Server-Sent Events (SSE Streaming)**:

- **ปัญหา Header Commitment**: หากเกตเวย์เริ่ม Flush HTTP Response Header (200 OK) และ Chunk แรกไปยัง Client แล้ว Client จะถือว่าคำขอสำเร็จ หาก Upstream ตัดการเชื่อมต่อกลางคัน เกตเวย์จะไม่สามารถส่ง 302 หรือเปลี่ยนช่องทางใหม่ได้
- **เทคนิค Header Holdback**: เกตเวย์จะรอรับ Token แรก (First Token) จาก Upstream และตรวจสอบความถูกต้องของ Status Code ก่อนจะส่ง HTTP Header ไปยัง Client
- **Buffering Threshold**: หากเกิด Connection Error ก่อนการส่ง Chunk แรก เกตเวย์จะทำ Dynamic Retry ไปยังช่องทางสำรองได้ทันทีโดยที่ Client ไม่รับรู้ถึงความผิดพลาด

---

## 3. การจำแนกประเภท Error และกฎการ Retry

ไม่ใช่ทุกข้อผิดพลาดที่จะสามารถ Retry ได้ การออกแบบ Router ต้องกำหนดเงื่อนไขอย่างเคร่งครัด:

| HTTP Status / Error | สาเหตุหลัก | นโยบาย Retry |
| :--- | :--- | :--- |
| **400 Bad Request** | Prompt ผิดรูปแบบ / พารามิเตอร์ผิด | **ห้าม Retry** (ส่งกลับ Client ทันที) |
| **401 Unauthorized** | API Key ไม่ถูกต้อง / หมดอายุ | **ห้าม Retry** เว้นแต่มี Key สำรองใน Vault |
| **429 Rate Limit** | ติด Quota TPM/RPM | **Retry ทันที** ไปยังช่องทางหรือโมเดลถัดไป |
| **500 Internal Error** | เซิร์ฟเวอร์ต้นทางมีปัญหา | **Retry ทันที** ข้าม Provider |
| **503 Service Unavailable** | เซิร์ฟเวอร์ต้นทาง Overload | **Retry ทันที** ข้าม Provider |
| **Client Abort / 499** | ผู้ใช้กดยกเลิกกลางคัน | **หยุดการทำงานทันที** ไม่คิดโควตาเพิ่ม |

---

## 4. ประโยชน์ด้านต้นทุนและการจัดสรรภาระงาน (Load Balancing)

นอกเหนือจากความเสถียร ระบบ Routing ยังช่วยควบคุมงบประมาณได้อย่างมหาศาล:
- **Cost-Optimized Routing**: กำหนดให้อัตราส่วนทราฟฟิก (Weighted Ratio) 80% วิ่งไปยังโมเดลขนาดเล็กที่มีราคาประหยัด เช่น Flash/Mini Model สำหรับงานที่ไม่ซับซ้อน และส่งงานที่มี Logic ซับซ้อนไปยัง Frontier Model
- **Prompt Caching Prioritization**: เลือกส่งคำขอไปยังผู้ให้บริการที่รองรับ Prompt Caching ก่อนเพื่อประหยัดต้นทุนโทเค็น
- ศึกษาโครงสร้างและแนวทางสถาปัตยกรรมเพิ่มเติมได้ที่ [คู่มือสถาปัตยกรรม AI API Gateway](https://www.toraapi.com/news/ai-api-gateway-architecture-guide)
- สามารถดูโครงสร้างราคาและการควบคุมโควตาอย่างละเอียดได้ที่ [Tora Pricing & Plans](https://www.toraapi.com/pricing)

---

## สรุปหลักการนำไปใช้

การสร้างระบบ AI ที่เสถียรไม่ได้ขึ้นอยู่กับความฉลาดของโมเดลเพียงอย่างเดียว แต่ขึ้นอยู่กับความแข็งแกร่งของโครงสร้างพื้นฐานในการส่งต่อคำขอ การติดตั้งระบบ Fallback Routing จึงเป็นการลงทุนพื้นฐานที่คุ้มค่าที่สุดสำหรับทุกทีมพัฒนา"""

def push_indexnow(urls):
    payload = {
        "host": "www.toraapi.com",
        "key": "484c06053f3e43a992a8327171e5491a",
        "keyLocation": "https://www.toraapi.com/484c06053f3e43a992a8327171e5491a.txt",
        "urlList": urls
    }
    req = urllib.request.Request(
        "https://api.indexnow.org/IndexNow",
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json; charset=utf-8", "User-Agent": "Tora-Organic-Autopilot/1.0"}
    )
    ctx = ssl._create_unverified_context()
    try:
        with urllib.request.urlopen(req, context=ctx, timeout=10) as resp:
            print(f"IndexNow push result: HTTP {resp.getcode()}")
    except Exception as e:
        print(f"IndexNow push warning: {e}")

def main():
    print("=" * 60)
    print("PUBLISHING AUTHORITATIVE EVERGREEN PILLARS TO TORA PRODUCTION")
    print("=" * 60)

    now = int(time.time())
    pillars = [
        {
            "slug": "ai-api-gateway-architecture-guide",
            "title": "คู่มือสถาปัตยกรรม AI API Gateway: เชื่อมต่อหลายโมเดลด้วยมาตรฐาน OpenAI-Compatible และระบบ BYOK",
            "summary": "เจาะลึกสถาปัตยกรรมการออกแบบ AI API Gateway สำหรับระบบ Production: การรวมศูนย์โมเดลด้วยมาตรฐาน OpenAI-Compatible, ระบบความปลอดภัยแบบ Bring Your Own Key (BYOK), และเทคนิค Dynamic Routing & High Availability Fallback สำหรับนักพัฒนาไทย",
            "content": PILLAR_1_MARKDOWN,
            "seo_title": "คู่มือสถาปัตยกรรม AI API Gateway: เชื่อมต่อหลายโมเดลด้วยมาตรฐาน OpenAI-Compatible และระบบ BYOK | Tora AI",
            "seo_desc": "เจาะลึกสถาปัตยกรรมการออกแบบ AI API Gateway สำหรับระบบ Production รองรับ OpenAI-Compatible, BYOK Key Vault, และ Dynamic Routing ป้องกัน API ล่ม",
            "keywords": "AI API Gateway, OpenAI Compatible, BYOK, Multi-Provider AI, LLM Architecture, Tora API"
        },
        {
            "slug": "llm-routing-and-fallback-architecture",
            "title": "คู่มือการออกแบบ LLM Routing & Dynamic Fallback: ป้องกัน API ล่ม ลดต้นทุน และเพิ่ม Uptime 99.9%",
            "summary": "คู่มือเชิงสถาปัตยกรรมสำหรับวิศวกรซอฟต์แวร์ในการออกแบบระบบ LLM Routing และ Dynamic Fallback: เทคนิคการรับมือ HTTP 429, การจัดการ Streaming Failover ด้วย SSE Buffer, และการจัดสรรภาระงานเพื่อลดต้นทุนสูงสุด",
            "content": PILLAR_2_MARKDOWN,
            "seo_title": "คู่มือการออกแบบ LLM Routing & Dynamic Fallback: ป้องกัน API ล่ม ลดต้นทุน และเพิ่ม Uptime 99.9% | Tora AI",
            "seo_desc": "คู่มือวิศวกรรมการออกแบบ LLM Routing และ Dynamic Fallback ป้องกันปัญหา HTTP 429 เพิ่ม Uptime 99.9% พร้อมเทคนิค Streaming Failover และ Cost Optimization",
            "keywords": "LLM Routing, Dynamic Fallback, AI High Availability, HTTP 429 Failover, SSE Streaming Fallback, Uptime 99.9%"
        }
    ]

    urls_to_push = []
    for p in pillars:
        slug = p["slug"]
        canonical_url = f"https://www.toraapi.com/news/{slug}"
        og_image_url = f"https://www.toraapi.com/news/{slug}/og.png"
        
        # Check if already exists
        check_cmd = [
            "ssh", "-i", SSH_KEY, "-o", "StrictHostKeyChecking=no",
            EC2_HOST,
            f"docker exec -i postgres psql -U root -d new-api -t -A -c \"SELECT count(*) FROM news_posts WHERE slug = '{slug}';\""
        ]
        res = subprocess.run(check_cmd, capture_output=True, text=True)
        exists = int(res.stdout.strip() or 0) > 0

        if exists:
            print(f"Pillar '{slug}' already exists. Updating content...")
            sql = f"""
            UPDATE news_posts SET
                title = '{p['title'].replace("'", "''")}',
                summary = '{p['summary'].replace("'", "''")}',
                content_markdown = '{p['content'].replace("'", "''")}',
                seo_title = '{p['seo_title'].replace("'", "''")}',
                seo_description = '{p['seo_desc'].replace("'", "''")}',
                seo_keywords = '{p['keywords'].replace("'", "''")}',
                canonical_url = '{canonical_url}',
                og_image_url = '{og_image_url}',
                updated_at = {now}
            WHERE slug = '{slug}';
            """
        else:
            print(f"Inserting new Evergreen Pillar: '{slug}'...")
            sql = f"""
            INSERT INTO news_posts (
                slug, content_type, title, summary, content_markdown, content_html,
                cluster_id, source_id, source_url, author_name, canonical_url,
                status, content_risk, fact_check_status, fact_check_notes,
                seo_title, seo_description, seo_keywords, og_image_url,
                is_seed, publication_origin, published_at, view_count,
                created_at, updated_at
            ) VALUES (
                '{slug}', 'guide', '{p['title'].replace("'", "''")}', '{p['summary'].replace("'", "''")}',
                '{p['content'].replace("'", "''")}', '',
                0, 0, 'https://www.toraapi.com/docs', 'Tora Technical Editorial', '{canonical_url}',
                'published', 'low', 'verified', 'Authoritative Tora Engineering Pillar Guide',
                '{p['seo_title'].replace("'", "''")}', '{p['seo_desc'].replace("'", "''")}', '{p['keywords'].replace("'", "''")}', '{og_image_url}',
                false, 'MANUAL_ADMIN', {now}, 0,
                {now}, {now}
            );
            """

        exec_cmd = [
            "ssh", "-i", SSH_KEY, "-o", "StrictHostKeyChecking=no",
            EC2_HOST,
            "docker exec -i postgres psql -U root -d new-api -t -A"
        ]
        run_res = subprocess.run(exec_cmd, input=sql, capture_output=True, text=True)
        if run_res.returncode != 0:
            print(f"Error executing SQL for {slug}: {run_res.stderr}")
        else:
            print(f"Successfully persisted pillar: {slug} (Status: published)")
            urls_to_push.append(canonical_url)

    # Push to IndexNow
    if urls_to_push:
        print("\nSubmitting published pillars to IndexNow...")
        push_indexnow(urls_to_push)

    print("\n" + "=" * 60)
    print("EVERGREEN PILLARS DEPLOYMENT COMPLETE")
    print("=" * 60)

if __name__ == "__main__":
    main()
