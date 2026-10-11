#!/usr/bin/env python3
"""
Thai Chat Evaluation & Benchmark Script
Supports Novita AI, Tora API (New API), OpenAI, vLLM, and any OpenAI-compatible endpoint.
"""

import os
import sys
import time
import json
from typing import List, Dict, Any

# Ensure UTF-8 output on Windows console
if sys.platform == "win32":
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

try:
    from openai import OpenAI
except ImportError:
    print("Installing openai sdk...")
    os.system(f"{sys.executable} -m pip install openai")
    from openai import OpenAI

# ---------------------------------------------------------
# Test Cases for Thai Language Evaluation
# ---------------------------------------------------------
TEST_SUITES = [
    {
        "category": "1. คุยทั่วไป & สุภาพ (Casual & Politeness)",
        "prompt": "สวัสดีครับ แนะนำตัวหน่อยว่าคุณคือใคร มีความสามารถด้านไหนบ้าง ตอบเป็นภาษาไทยอย่างสุภาพและเป็นกันเอง",
        "system": "คุณคือ AI ผู้ช่วยอัจฉริยะที่พูดภาษาไทยได้อย่างเป็นธรรมชาติ สุภาพ และเข้าใจบริบทวัฒนธรรมไทย",
        "max_tokens": 800,
    },
    {
        "category": "2. การปรึกษาธุรกิจ & กลยุทธ์ (Business Strategy & SWOT)",
        "prompt": """ผมต้องการเปิดร้านกาแฟ Specialty ในย่านออฟฟิศใจกลางเมืองกรุงเทพฯ งบลงทุน 800,000 บาท 
ช่วยวิเคราะห์สั้นๆ:
1. จุดแข็งและจุดอ่อนหลัก (Strengths & Weaknesses)
2. แผนการตั้งราคากาแฟต่อแก้วและเมนูที่จะดึงดูดพนักงานออฟฟิศ
3. กลยุทธ์การตลาดเปิดร้านในสัปดาห์แรก""",
        "system": "คุณคือที่ปรึกษาด้านธุรกิจและการตลาดชั้นนำ มีความเชี่ยวชาญตลาดประเทศไทยและพฤติกรรมผู้บริโภคคนไทย ให้คำแนะนำที่กระชับ นำไปใช้ได้จริง (Actionable)",
        "max_tokens": 1200,
    },
    {
        "category": "3. การแก้ปัญหาเฉพาะหน้า & คำนวณต้นทุน (Financial & Operations)",
        "prompt": """คำนวณต้นทุนให้หน่อย:
- ค่าเช่าที่ 25,000 บาท/เดือน
- พนักงาน 2 คน รวมเงินเดือน 36,000 บาท/เดือน
- ค่าน้ำไฟ 8,000 บาท/เดือน
- ต้นทุนวัตถุดิบ (เมล็ดกาแฟ, นม, แก้ว) 20 บาท/แก้ว
- ถ้าขายแก้วละ 65 บาท ต้องขายวันละกี่แก้ว (คิด 30 วัน) ถึงจะถึงจุดคุ้มทุน (Breakeven)? แสดงวิธีคิดทีละขั้นตอน""",
        "system": "คุณคือผู้เชี่ยวชาญด้านการเงินและการบัญชีต้นทุน คิดเลขและแสดงสมการให้ชัดเจน ถูกต้อง",
        "max_tokens": 1200,
    },
    {
        "category": "4. การปฏิบัติตามคำสั่ง & กฎเกณฑ์ (Instruction Following & Format)",
        "prompt": """สรุปประโยชน์ของการทำ SEO สำหรับธุรกิจ SME ออกมาเป็นข้อๆ โดยมีเงื่อนไขดังนี้:
1. ต้องมีหัวข้อย่อยตรง 3 ข้อเท่านั้น ห้ามเกิน
2. แต่ละข้อต้องมีความยาวไม่เกิน 2 บรรทัด
3. ปิดท้ายด้วยประโยคคำคมธุรกิจ 1 ประโยค""",
        "system": "คุณคือ AI ผู้ปฏิบัติตามคำสั่งอย่างเคร่งครัด รักษากฎทุกข้ออย่างถูกต้อง 100%",
        "max_tokens": 800,
    }
]


def run_benchmark(base_url: str, api_key: str, model: str):
    print("=" * 70)
    print(f"🇹🇭 Starting Thai Chat Benchmark")
    print(f"Endpoint: {base_url}")
    print(f"Model   : {model}")
    print("=" * 70)

    client = OpenAI(
        base_url=base_url,
        api_key=api_key,
        default_headers={"User-Agent": "Mozilla/5.0"}
    )

    total_latency = 0.0
    passed_tests = 0

    for idx, tc in enumerate(TEST_SUITES, 1):
        print(f"\n[Test {idx}/{len(TEST_SUITES)}] {tc['category']}")
        print(f"คำถาม: {tc['prompt'][:60]}...")
        
        messages = [
            {"role": "system", "content": tc["system"]},
            {"role": "user", "content": tc["prompt"]}
        ]

        start_time = time.time()
        try:
            response = client.chat.completions.create(
                model=model,
                messages=messages,
                max_tokens=tc["max_tokens"],
                temperature=0.7,
            )
            elapsed = time.time() - start_time
            total_latency += elapsed

            choice = response.choices[0]
            msg = choice.message
            content = msg.content or ""
            reasoning = getattr(msg, "reasoning_content", None) or getattr(msg, "reasoning", None) or ""

            usage = response.usage
            prompt_tokens = usage.prompt_tokens if usage else 0
            completion_tokens = usage.completion_tokens if usage else 0
            speed = completion_tokens / elapsed if elapsed > 0 and completion_tokens > 0 else 0

            print(f"⏱️ เวลาที่ใช้: {elapsed:.2f}s | {speed:.1f} tokens/s (Out: {completion_tokens} tokens)")
            print("-" * 50)
            if reasoning:
                print("🧠 [ความคิด/Reasoning]:")
                print(reasoning.strip()[:400] + ("..." if len(reasoning) > 400 else ""))
                print("-" * 25)
            if content:
                print("💬 [คำตอบ/Content]:")
                print(content.strip())
            elif reasoning:
                print("💬 [คำตอบอยู่ในขั้นตอนคิด]:")
                print(reasoning.strip())
            print("-" * 50)
            passed_tests += 1

        except Exception as e:
            elapsed = time.time() - start_time
            print(f"❌ Error ({elapsed:.2f}s): {e}")

    print("\n" + "=" * 70)
    print(f"📊 สรุปผลการทดสอบ: ผ่าน {passed_tests}/{len(TEST_SUITES)} การทดสอบ")
    if passed_tests > 0:
        print(f"⚡ เวลาเฉลี่ยต่อข้อ: {total_latency / passed_tests:.2f} วินาที")
    print("=" * 70)


if __name__ == "__main__":
    # ค่าเริ่มต้น: สามารถ Override ผ่าน Environment Variables หรือ Arguments
    DEFAULT_BASE_URL = os.environ.get("OPENAI_BASE_URL", "https://api.novita.ai/v3/openai")
    DEFAULT_API_KEY = os.environ.get("OPENAI_API_KEY", "sk_ndBx7GQYhVJE1d5QvHCq2uE0JxvqhtcSXU0drjLd8XM")
    DEFAULT_MODEL = os.environ.get("OPENAI_MODEL", "meta-llama/llama-3.3-70b-instruct")

    # Command line arguments support
    # python test_thai_chat.py [base_url] [api_key] [model]
    base_url = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_BASE_URL
    api_key = sys.argv[2] if len(sys.argv) > 2 else DEFAULT_API_KEY
    model = sys.argv[3] if len(sys.argv) > 3 else DEFAULT_MODEL

    run_benchmark(base_url=base_url, api_key=api_key, model=model)
