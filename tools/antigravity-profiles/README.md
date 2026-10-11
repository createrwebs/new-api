# Antigravity Multi-Profile System

ระบบแยกโปรไฟล์และรันหลาย Instances พร้อมกันสำหรับ **Google Antigravity (`agy`)** และ **Google Cloud SDK (`gcloud`)**
ออกแบบให้รองรับทั้งการใช้งานผ่าน Terminal (Interactive) และการเรียกใช้งานอัตโนมัติผ่าน AI Agent (Programmatic / Parallel)

---

## 📁 โครงสร้างไฟล์ในชุดเครื่องมือ

- [`launch-profiles.ps1`](file:///d:/new-api/tools/antigravity-profiles/launch-profiles.ps1) — สคริปต์ PowerShell สำหรับรันแบบแยก 2 หน้าต่างบน Windows / Windows Terminal
- [`launch-profiles.sh`](file:///d:/new-api/tools/antigravity-profiles/launch-profiles.sh) — สคริปต์ Bash สำหรับ macOS, Linux, WSL
- [`agent_pool.py`](file:///d:/new-api/tools/antigravity-profiles/agent_pool.py) — **Agent Orchestrator สำหรับให้ AI Agent สั่งการหลาย Profile พร้อมกัน**
- [`example_tasks.json`](file:///d:/new-api/tools/antigravity-profiles/example_tasks.json) — ตัวอย่างคำสั่ง Task สำหรับรันแบบ Concurrent Batch

---

## 🔑 หลักการแยก Profile อย่างเด็ดขาด (Isolation Mechanism)

1. **`CLOUDSDK_CONFIG`**: ชี้ไปยัง `~/.antigravity-profiles/<profile>/gcloud` เพื่อแยก Active Account, Token และการตั้งค่าของ `gcloud`
2. **`HOME` / `USERPROFILE`**: ชี้ไปยัง `~/.antigravity-profiles/<profile>/home` เพื่อแยกโฟลเดอร์ `~/.gemini/` ซึ่งเก็บ Auth Token, Sessions, Conversation Summaries, และ Cache ของ `agy`
3. **`ANTIGRAVITY_EXECUTABLE_DATA_DIR`**: กำหนด Path จัดเก็บข้อมูลและ Artifacts ให้แยกออกจากบัญชีหลัก
4. **`Workspace Isolation`**: แยก Root Directory การทำงาน ป้องกัน Git conflicts หรือการแย่งเขียนไฟล์

---

## 🛠️ ขั้นตอนการ Login ครั้งแรก (ป้องกัน Session ชนกัน)

เนื่องจากการ Login ด้วย OAuth มักจะเปิด Web Browser หลักขึ้นมา หากเบราว์เซอร์มี Google Session อยู่แล้วอาจล็อกอินผิดบัญชี:

### ตัวเลือกที่ 1: ใช้ Google Chrome แยก Profile
สร้าง Chrome Shortcut แยก Profile สำหรับแต่ละบัญชี:
```bash
chrome.exe --user-data-dir="C:\tmp\chrome-acc1"
chrome.exe --user-data-dir="C:\tmp\chrome-acc2"
```

### ตัวเลือกที่ 2: Login gcloud ก่อนผ่าน CLI โดยไม่เปิด Browser อัตโนมัติ
```bash
# สำหรับ acc1
export CLOUDSDK_CONFIG="$HOME/.antigravity-profiles/acc1/gcloud"
gcloud auth application-default login --no-launch-browser

# สำหรับ acc2
export CLOUDSDK_CONFIG="$HOME/.antigravity-profiles/acc2/gcloud"
gcloud auth application-default login --no-launch-browser
```
*(คัดลอก URL ไปเปิดใน Incognito Window ของแต่ละบัญชี)*

---

## 🤖 การนำไปใช้งานกับ AI Agent (Programmatic Usage)

### 1. เรียกใช้งานผ่าน Python Code (Multi-Agent Concurrent)
```python
import asyncio
from agent_pool import AntigravityAgentPool, AgentTask

async def main():
    pool = AntigravityAgentPool(max_concurrent_agents=4)

    tasks = [
        AgentTask(task_id="task-1", profile_name="acc1", prompt="วิเคราะห์ code ในโฟลเดอร์นี้"),
        AgentTask(task_id="task-2", profile_name="acc2", prompt="เขียน unit test สำหรับ auth service"),
    ]

    results = await pool.execute_batch(tasks)
    for r in results:
        print(f"[{r.profile_name}] Success: {r.success}, Output length: {len(r.output)}")

asyncio.run(main())
```

### 2. เรียกใช้งานผ่าน CLI (Non-interactive)
```bash
python agent_pool.py run --profile acc1 --prompt "ตรวจสอบ lint errors"
```

### 3. รันเป็นชุดผ่านไฟล์ JSON
```bash
python agent_pool.py batch --file example_tasks.json --max-concurrent 2
```
