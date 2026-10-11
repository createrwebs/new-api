import urllib.request
import json
import sys

if sys.platform == "win32":
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    sys.stderr.reconfigure(encoding="utf-8", errors="replace")

base_url = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:3000"
model = sys.argv[2] if len(sys.argv) > 2 else "qwen2.5:3b"
api_key = sys.argv[3] if len(sys.argv) > 3 else "sk-V9udfhPBGv7M14NF0gwpqplBhJGO7KF11253ZDJgJTk2Icct"

req = urllib.request.Request(
    f"{base_url}/v1/chat/completions",
    headers={
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}"
    },
    data=json.dumps({
        "model": model,
        "messages": [{"role": "user", "content": "สวัสดีครับ แนะนำตัวสั้นๆ ใน 1 ประโยค"}],
        "max_tokens": 50,
        "temperature": 0.7
    }).encode("utf-8")
)

try:
    print(f"Sending request to {base_url} for model {model}...")
    with urllib.request.urlopen(req, timeout=90) as resp:
        result = json.loads(resp.read().decode("utf-8"))
        print("\n=== SUCCESS ===")
        print(f"Status: {resp.status}")
        print(f"Model returned: {result.get('model')}")
        content = result["choices"][0]["message"]["content"]
        print(f"Reply: {content}")
        print(f"Usage: {result.get('usage')}")
except Exception as e:
    print(f"\n=== ERROR ===")
    if hasattr(e, "read"):
        print("Response body:", e.read().decode("utf-8"))
    print("Exception:", e)
