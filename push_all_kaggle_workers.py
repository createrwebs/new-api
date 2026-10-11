import urllib.request
import urllib.error
import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

worker_code = """# =======================================================
# TORA API GATEWAY - KAGGLE GPU WORKER
# Models: Qwen 2.5 7B & DeepSeek R1 7B on Tesla T4 GPU
# =======================================================
import subprocess
import time
import os
import re
import urllib.request

print(">>> [1/5] System setup & prerequisites...")
subprocess.run(["apt-get", "update", "-y"], capture_output=True)
subprocess.run(["apt-get", "install", "-y", "zstd", "curl"], capture_output=True)

# Install Cloudflared
cloudflared = "/usr/local/bin/cloudflared"
if not os.path.exists(cloudflared):
    print(">>> Installing cloudflared...")
    subprocess.run(["curl", "-fsSL", "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64", "-o", cloudflared], check=True)
    subprocess.run(["chmod", "+x", cloudflared], check=True)

# Install Ollama
print(">>> [2/5] Installing Ollama...")
subprocess.run(["curl", "-fsSL", "https://ollama.com/install.sh", "-o", "/tmp/install.sh"], check=True)
subprocess.run(["sh", "/tmp/install.sh"], check=True)

print(">>> [3/5] Starting Ollama server...")
ollama_proc = subprocess.Popen(["ollama", "serve"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
time.sleep(5)

print(">>> [4/5] Pulling Qwen 2.5 7B onto Tesla T4 GPU...")
t0 = time.time()
subprocess.run(["ollama", "pull", "qwen2.5:7b"], check=True)
print(f">>> Qwen 2.5 7B ready in {time.time()-t0:.2f}s!")

print(">>> [5/5] Establishing Cloudflare Tunnel to port 11434...")
tunnel_proc = subprocess.Popen(["cloudflared", "tunnel", "--url", "http://localhost:11434"], stderr=subprocess.PIPE, text=True)

tunnel_url = None
start_time = time.time()
while time.time() - start_time < 35:
    line = tunnel_proc.stderr.readline()
    match = re.search(r"https://[a-zA-Z0-9-]+\\.trycloudflare\\.com", line)
    if match:
        tunnel_url = match.group(0)
        print("="*60)
        print(f"PUBLIC WORKER URL: {tunnel_url}")
        print("="*60)
        break

if not tunnel_url:
    print("Warning: Tunnel URL not captured, keeping tunnel alive.")

print(">>> Worker is active and ready for requests. Entering keep-alive loop...")
loop_count = 0
while True:
    time.sleep(30)
    loop_count += 1
    if loop_count % 20 == 0:
        print(f">>> Worker heartbeat: running for {loop_count * 30 / 60:.1f} minutes")
"""

accounts = [
    {
        "id": 1,
        "alias": "Kaggle_websbringer",
        "username": "noppananthe",
        "token": "KGAT_690a2507e0c94e4538b6dcd1bc35dfbf",
        "slug": "noppananthe/tora-worker-t4-1"
    },
    {
        "id": 2,
        "alias": "Kaggle_codeathee",
        "username": "codeatheerachart",
        "token": "KGAT_c1b64a26e0d237d13ec99e10248831fe",
        "slug": "codeatheerachart/tora-worker-t4-2"
    },
    {
        "id": 3,
        "alias": "kaggle_softcreate9",
        "username": "weeblysdwebmagic",
        "token": "KGAT_554c5c65e6e6b8a953b38f3909647fb2",
        "slug": "weeblysdwebmagic/tora-worker-t4-3"
    },
    {
        "id": 4,
        "alias": "Kaggle_createrwebs",
        "username": "creatorwebs",
        "token": "KGAT_8ecae0ef31fbd25f8d867eb59efca686",
        "slug": "creatorwebs/tora-worker-t4-4"
    }
]

print("=== PUSHING KAGGLE GPU WORKERS ===")

for acc in accounts:
    title = f"tora-worker-t4-{acc['id']}"
    slug = acc["slug"]
    token = acc["token"]
    
    body = {
        "slug": slug,
        "newTitle": title,
        "text": worker_code,
        "language": "python",
        "kernelType": "script",
        "isPrivate": True,
        "enableGpu": True,
        "enableInternet": True
    }
    
    data = json.dumps(body).encode('utf-8')
    req = urllib.request.Request(
        "https://www.kaggle.com/api/v1/kernels/push",
        data=data,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json"
        }
    )
    
    try:
        with urllib.request.urlopen(req) as resp:
            res = json.loads(resp.read().decode('utf-8'))
            print(f"[+] Worker {acc['id']} ({acc['alias']}) PUSHED! URL: {res.get('url')}")
    except urllib.error.HTTPError as e:
        err_msg = e.read().decode('utf-8', errors='ignore')
        print(f"[-] Worker {acc['id']} Error HTTP {e.code}: {err_msg[:100]}")
    except Exception as e:
        print(f"[-] Worker {acc['id']} Error: {e}")
