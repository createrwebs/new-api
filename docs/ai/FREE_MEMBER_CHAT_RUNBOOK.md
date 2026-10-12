# แช็ต AI ฟรีสำหรับสมาชิกใหม่ — toraapi.com

วันที่ 2026-10-12 (Asia/Bangkok); repository `/Users/noppanan/new-api`, branch `main`.

## ข้อกำหนดที่ผู้ใช้ยืนยัน

เว็บ toraapi.com ต้องมี AI สำหรับคุย เปิดให้สมาชิกใหม่ใช้งานฟรีหลังสมัครและเข้าสู่ระบบ โดยไม่ต้องซื้อแพ็กเกจ เติมเงิน หรือมี provider key ของตนเองก่อนแช็ตครั้งแรก ใช้หน้า `/playground` และระบบสมาชิก/relay เดิมของ New-API ไม่สร้างบริการแช็ตหรือระบบล็อกอินใหม่

- โมเดลหลัก: `scb10x/llama3.1-typhoon2-8b-instruct`
- โมเดลเพิ่มเติม: `tora-qwen`
- ทั้งสองโมเดลคิดราคา 0; คงราคาและสิทธิ์โมเดลอื่นตามเดิม
- ใช้ group `default` เดิม; สมาชิกเดิมในกลุ่มนี้เข้าถึงได้ด้วย ไม่จำกัดเฉพาะวันสมัครใหม่
- ใช้ตัวจำกัดฟรีเดิม 20 คำขอต่อบัญชีต่อวันตาม Asia/Bangkok ร่วมกันระหว่างหน้าเว็บและ API สำหรับบัญชีที่ไม่มี wallet quota และไม่มี subscription ที่ active บัญชีที่มี wallet quota, subscription หรือ role admin ข้ามตัวจำกัดนี้ตามนโยบายเดิม
- จำนวน 20 นับคำขอที่รับเข้ามาก่อนเลือก channel ไม่ใช่จำนวนคำตอบสำเร็จ; upstream/channel error ไม่คืนจำนวนคำขอ ตามพฤติกรรมเดิม

## ขอบเขตและการเปลี่ยนแปลง

เพิ่ม `/pg/chat/completions` ลงใน guard `middleware/free_tier.go` และเรียก `FreeTierQuota()` หลัง `UserAuth()`/concurrency ก่อน `Distribute()` ใน router เดิม หน้าเว็บจึงใช้ counter เดียวกับ `/v1/chat/completions` และ `/v1/responses`; Playground เรียก Relay ภายในไม่ได้นับซ้ำ

Regression tests ครอบคลุมเว็บ/API ใช้ counter ร่วมกันทั้งสองทิศทาง, dashboard session จริงถูกปฏิเสธ 429 ก่อนหา channel เมื่อใช้ครบแล้ว และผู้ไม่ล็อกอินยังถูกปฏิเสธ 401 ไม่มีการแก้ schema, paid billing, grant quota ตอนสมัคร, BYOK หรือระบบ mobile

## Upstream และ Azure ที่ผู้ใช้ให้มา

OpenAI-compatible API base: `https://self-begun-upgrading-retreat.trycloudflare.com/v1`

ค่าช่องทาง type 1 ใน New-API ใช้ **origin ที่ไม่มี `/v1`** เพราะ adaptor ต่อ `/v1/chat/completions` ให้แล้ว อย่าใส่ `/v1` ซ้ำใน `channel.base_url`

`FREE_MEMBER_CHAT_CHANNEL.json` เป็น payload สำหรับ `POST /api/channel/` ใช้ dummy key `toraapi` ที่ผู้ใช้ระบุว่าไม่ใช่ secret, logical model mapping ไปชื่อ `:latest` ที่ upstream ประกาศจริง และสร้าง channel ในสถานะ disabled (`status: 2`) เพื่อเปิดหลังตรวจ gate ด้านล่าง ห้ามส่ง dummy key นี้แทน member/session/relay credential ของ Tora

ข้อมูล Azure เป็น user-reported ยังไม่ได้ตรวจ Azure control plane:

| รายการ | ค่า |
| --- | --- |
| Public IP | `172.198.56.245` |
| Private IP | `172.16.0.4` |
| AI port | `11434` |
| Network interface | `ai192` |
| VNet / subnet | `vnet-indiasouthcentral-1` / `snet-indiasouthcentral-1` |
| NSG | `AI-nsg` |
| Rule 310 | Allow 11434 จาก Any ตามข้อมูลที่ผู้ใช้ส่ง |

อย่าให้ frontend ติดต่อ IP/Cloudflare upstream โดยตรง ใช้ same-origin Tora `/pg/chat/completions` เพื่อบังคับสมาชิกและ quota การเปิด 11434 จาก Any และ tunnel ที่ไม่ตรวจ key ทำให้บุคคลทั่วไปเรียก upstream ข้ามระบบสมาชิกได้ ต้องจำกัด NSG ให้รับจาก gateway ที่อนุญาต และให้ upstream HTTPS ตรวจ service credential/จำกัด caller ก่อนเปิดสู่สมาชิกจริง การมี HTTPS ไม่ใช่หลักฐานว่ามี access control

TryCloudflare เป็น URL ชั่วคราวสำหรับการทดสอบ ไม่ใช่ public origin ถาวร และ Quick Tunnel ไม่รองรับ SSE ตามเอกสารผู้ให้บริการ แม้ bounded SSE probe ครั้งนี้จะสำเร็จ ให้ใช้ HTTPS origin/tunnel ที่รองรับ SSE และจำกัด caller ได้ก่อน production streaming

อ้างอิง: [Cloudflare Quick Tunnels](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/), [Ollama OpenAI compatibility](https://docs.ollama.com/api/openai-compatibility)

## เปิดใช้งานด้วยระบบ Admin เดิม

1. Build และติดตั้ง revision ที่มี guard `/pg` นี้ด้วย deployment flow ของโครงการเดิม ตรวจ hash ของ artifact และ rollback artifact ก่อนดำเนินการ ห้ามใช้คำสั่ง deploy/DB credentials จากเอกสารเก่าโดยไม่ตรวจ runtime
2. เข้าสู่ Admin ด้วยสิทธิ์ที่มีอยู่ ตรวจ channel ชื่อ `Tora Free Thai Chat` ก่อน เพื่อไม่สร้างซ้ำ ถ้ามีแล้วแก้เฉพาะ channel นั้น; ถ้าไม่มี ใช้ payload JSON ผ่าน `POST /api/channel/` หรือกรอกในหน้า Channels โดยเก็บ disabled ไว้ก่อน การสร้าง channel ต้องมี `channel.sensitive.write`
3. เปิด Models/Pricing ตั้งเฉพาะ logical model สองชื่อเป็น billing mode `ratio`, `ModelRatio: 0`, `CompletionRatio: 1` และลบ fixed `ModelPrice`/billing expression ของสองชื่อหากมี อย่าเปลี่ยน `GroupRatio.default` เป็น 0 และอย่า replace pricing map ทั้งระบบ
4. ถ้าใช้ API: root-authenticated `GET /api/option/model_pricing?model=scb10x%2Fllama3.1-typhoon2-8b-instruct&model=tora-qwen` อ่าน `entries[].version` และ `configured` เก็บ backup เฉพาะสองชื่อ จากนั้น `PATCH /api/option/model_pricing` ด้วย `changes[]` ที่มี `model_name`, `expected_version` ที่เพิ่งอ่านจริง และ `pricing: {"ModelRatio":0,"CompletionRatio":1,"billing_setting.billing_mode":"ratio"}` ถ้า 409 ต้อง reload/review ไม่ retry ด้วย version เก่า
5. Root ตั้ง `quota_setting.enable_free_model_pre_consume` เป็น `false` ผ่าน Settings หรือ `PUT /api/option/` ด้วย `{"key":"quota_setting.enable_free_model_pre_consume","value":"false"}` หลังบันทึกค่าเดิม นี่เป็น option **ระดับระบบ** กระทบทุกโมเดลที่ราคา/ratio เป็น 0; โมเดลที่คิดเงินยังใช้ billing เดิม ต้องตรวจโมเดลฟรีอื่นด้วย การเปลี่ยนนี้จำเป็นเพราะค่า default true ทำให้ zero-wallet ยังถูก billing session ปฏิเสธแม้ ratio 0
6. เปลี่ยน upstream เป็น origin ที่มีการจำกัด caller และรองรับ SSE, ทดสอบ channel ทั้งสองโมเดล แล้วเปิด channel นี้จาก disabled เป็น enabled
7. ใช้บัญชีทดสอบที่สมัครใหม่ group default, wallet 0, ไม่มี subscription: หน้า `/playground` เห็นสองโมเดล, ส่งข้อความภาษาไทยแล้วรับ stream จนจบ, wallet ยังคง 0, ไม่มีการพาไปเติมเงิน ใช้บัญชีทดสอบโดยเฉพาะพิสูจน์คำขอที่ 21 ได้ 429 `FREE_DAILY_LIMIT` และข้ามวัน Bangkok เริ่ม counter ใหม่ ตรวจว่า user ที่ไม่ล็อกอินเข้า inference ไม่ได้
8. ตรวจ paid/BYOK regression และ route mapping ไม่เปิดโมเดล Pro ผ่าน alias ฟรีก่อนรับรอง live member chat

Rollback: disable เฉพาะ channel นี้, คืน pricing เฉพาะสองชื่อด้วย current CAS version และ snapshot `configured` ก่อนเปลี่ยน, คืนค่า global free preconsume เดิม ไม่ลบสมาชิก, quota counter, subscriptions หรือข้อมูลลูกค้า

## สถานะหลักฐาน

- `GET upstream /v1/models`: HTTP/2 200 แสดง `tora-qwen:latest`, `scb10x/llama3.1-typhoon2-8b-instruct:latest`, `qwen2.5:14b`; ไม่เปิด raw qwen2.5 ให้ free channel
- Typhoon bounded SSE probe (`max_tokens: 16`): HTTP/2 200, `text/event-stream`, คำตอบจริง `สวัสดี`, `finish_reason: stop`, `[DONE]` ไม่ใช้ mock
- Qwen probe, build และ regression results: บันทึกใน `FREE_MEMBER_CHAT_VERIFICATION.md` หลังคำสั่งจบ
- toraapi.com/www.toraapi.com HTTPS: curl exit 35 จากเครือข่ายเครื่องนี้; HTTP และ Azure IP ได้หน้า `AXONS - Access Blocked`/503 เป็น network-policy blocker ไม่ใช่หลักฐานว่า origin ล่ม ไม่ปิด TLS validation หรือเลี่ยง network policy
- ไม่มี Sites/Hostinger website ที่เชื่อมกับโดเมนนี้ใน connector ที่ตรวจ ไม่ค้น credential/SSH key/production env หรือ DB จากนอกโปรเจกต์
- Production deployment, admin channel/pricing mutation และ signup-to-chat บน public site: **NOT RUN / BLOCKED** จนกว่า coordinator จะเข้าถึง deployment/admin ผ่านช่องทางที่อนุญาตได้ ไม่ต้องสร้างระบบใหม่เพื่อแก้ access blocker

คำสั่ง build ถัดไปจาก `/Users/noppanan/new-api/web`: `rtk proxy bun run build:check`

หลัง frontend ผ่าน จาก `/Users/noppanan/new-api`: `rtk proxy go build -o bin/new-api-free-chat .`
