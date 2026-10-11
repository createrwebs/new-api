import os
import subprocess
import json
import urllib.request
import urllib.error
import re
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import HTMLResponse, JSONResponse
from pydantic import BaseModel

app = FastAPI(title="Tora API - AI Workers Manager", docs_url=None, redoc_url=None)

# Configuration
CONFIG = {
    "ec2_text": {
        "id": "ec2_text",
        "name": "AWS EC2 #1 — Sydney Graviton2 (Text AI)",
        "ip": "54.206.183.194",
        "port": 11434,
        "key": os.path.expanduser("~/.ssh/toraapi-aws-key.pem"),
        "service": "ollama",
        "channel_id": 15,
        "models": ["qwen2.5:7b", "qwen2.5:3b", "llama3.2:3b", "deepseek-r1:1.5b"],
        "hardware": "2 vCPU Graviton2, 8GB RAM (ARM64)"
    },
    "ec2_image": {
        "id": "ec2_image",
        "name": "AWS EC2 #2 — Sydney Sapphire Rapids (Image AI)",
        "ip": "13.239.38.104",
        "port": 8000,
        "key": os.path.expanduser("~/.ssh/toraapi-image-key.pem"),
        "service": "sd-image",
        "channel_id": 16,
        "models": ["sd-turbo", "dall-e-2", "sd-image"],
        "hardware": "2 vCPU Intel Sapphire Rapids, 4GB RAM + 4GB Swap"
    },
    "kaggle_1": {
        "id": "kaggle_1",
        "name": "Kaggle Worker 1 (noppananthe)",
        "username": "noppananthe",
        "token": "KGAT_690a2507e0c94e4538b6dcd1bc35dfbf",
        "slug": "noppananthe/tora-worker-t4-1",
        "channel_id": 17,
        "models": ["qwen2.5:7b", "deepseek-r1:7b"],
        "hardware": "Tesla T4 x2 GPU (32GB VRAM)"
    },
    "kaggle_2": {
        "id": "kaggle_2",
        "name": "Kaggle Worker 2 (codeatheerachart)",
        "username": "codeatheerachart",
        "token": "KGAT_c1b64a26e0d237d13ec99e10248831fe",
        "slug": "codeatheerachart/tora-worker-t4-2",
        "channel_id": 18,
        "models": ["qwen2.5:7b", "deepseek-r1:7b"],
        "hardware": "Tesla T4 GPU (16GB VRAM)"
    },
    "kaggle_3": {
        "id": "kaggle_3",
        "name": "Kaggle Worker 3 (weeblysdwebmagic)",
        "username": "weeblysdwebmagic",
        "token": "KGAT_554c5c65e6e6b8a953b38f3909647fb2",
        "slug": "weeblysdwebmagic/tora-worker-t4-3",
        "channel_id": 19,
        "models": ["qwen2.5:7b", "deepseek-r1:7b"],
        "hardware": "Tesla T4 GPU (16GB VRAM)"
    },
    "kaggle_4": {
        "id": "kaggle_4",
        "name": "Kaggle Worker 4 (creatorwebs)",
        "username": "creatorwebs",
        "token": "KGAT_8ecae0ef31fbd25f8d867eb59efca686",
        "slug": "creatorwebs/tora-worker-t4-4",
        "channel_id": 20,
        "models": ["qwen2.5:7b", "deepseek-r1:7b"],
        "hardware": "Tesla T4 GPU (16GB VRAM)"
    }
}

def ssh_exec(ip, key_path, cmd, timeout=6):
    try:
        res = subprocess.run(
            ["ssh", "-n", "-i", key_path, "-o", "StrictHostKeyChecking=no", "-o", f"ConnectTimeout={timeout}", f"ubuntu@{ip}", cmd],
            capture_output=True, text=True, timeout=timeout+2
        )
        return res.returncode == 0, res.stdout.strip(), res.stderr.strip()
    except Exception as e:
        return False, "", str(e)

def update_db_channel(channel_id, status):
    cmd = f"docker exec -i postgres psql -U root -d new-api -c 'UPDATE channels SET status = {status} WHERE id = {channel_id};'"
    subprocess.run(cmd, shell=True, capture_output=True)

def get_kaggle_status(u, slug, tok):
    url = f"https://www.kaggle.com/api/v1/kernels/status?userName={u}&kernelSlug={slug.split('/')[-1]}"
    try:
        req = urllib.request.Request(url, headers={"Authorization": f"Bearer {tok}"})
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            return data.get("status", "unknown")
    except Exception as e:
        return "offline"

def get_kaggle_output(u, slug, tok):
    url = f"https://www.kaggle.com/api/v1/kernels/output?userName={u}&kernelSlug={slug.split('/')[-1]}"
    try:
        req = urllib.request.Request(url, headers={"Authorization": f"Bearer {tok}"})
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode())
            log_str = data.get("logNullable", "")
            try:
                entries = json.loads(log_str)
                return "\n".join([e.get("data", "").strip() for e in entries if e.get("data")])
            except Exception:
                return log_str
    except Exception as e:
        return f"Error fetching output: {e}"

@app.get("/workers/api/status")
@app.get("/api/status")
def api_status():
    results = {}
    
    # 1. EC2 Text
    cfg1 = CONFIG["ec2_text"]
    ok1, out1, _ = ssh_exec(cfg1["ip"], cfg1["key"], f"sudo systemctl is-active {cfg1['service']}")
    status1 = "running" if out1 == "active" else "stopped"
    # CPU / RAM
    _, top1, _ = ssh_exec(cfg1["ip"], cfg1["key"], "free -m | awk '/Mem:/ {print $3\"/\"$2\" MB\"}'")
    results["ec2_text"] = {
        "status": status1,
        "detail": top1 if ok1 else "Unreachable",
        "ip": cfg1["ip"],
        "models": cfg1["models"],
        "hardware": cfg1["hardware"]
    }

    # 2. EC2 Image
    cfg2 = CONFIG["ec2_image"]
    ok2, out2, _ = ssh_exec(cfg2["ip"], cfg2["key"], f"sudo systemctl is-active {cfg2['service']}")
    status2 = "running" if out2 == "active" else "stopped"
    _, top2, _ = ssh_exec(cfg2["ip"], cfg2["key"], "free -m | awk '/Mem:/ {print $3\"/\"$2\" MB\"}'")
    results["ec2_image"] = {
        "status": status2,
        "detail": top2 if ok2 else "Unreachable",
        "ip": cfg2["ip"],
        "models": cfg2["models"],
        "hardware": cfg2["hardware"]
    }

    # 3. Kaggle Workers
    for k in ["kaggle_1", "kaggle_2", "kaggle_3", "kaggle_4"]:
        cfg = CONFIG[k]
        k_status = get_kaggle_status(cfg["username"], cfg["slug"], cfg["token"])
        results[k] = {
            "status": k_status,
            "detail": f"Kaggle Cloud ({cfg['username']})",
            "models": cfg["models"],
            "hardware": cfg["hardware"]
        }

    return results

class ActionRequest(BaseModel):
    target: str
    action: str

@app.post("/workers/api/action")
@app.post("/api/action")
def api_action(req: ActionRequest):
    t = req.target
    a = req.action
    
    if t not in CONFIG:
        raise HTTPException(status_code=400, detail="Unknown target")
        
    cfg = CONFIG[t]
    
    if t in ["ec2_text", "ec2_image"]:
        srv = cfg["service"]
        if a == "start":
            ok, _, err = ssh_exec(cfg["ip"], cfg["key"], f"sudo systemctl start {srv}")
            update_db_channel(cfg["channel_id"], 1)
            return {"success": ok, "message": f"Started {srv}" if ok else err}
        elif a == "stop":
            ok, _, err = ssh_exec(cfg["ip"], cfg["key"], f"sudo systemctl stop {srv}")
            update_db_channel(cfg["channel_id"], 2)
            return {"success": ok, "message": f"Stopped {srv}" if ok else err}
        elif a == "restart":
            ok, _, err = ssh_exec(cfg["ip"], cfg["key"], f"sudo systemctl restart {srv}")
            update_db_channel(cfg["channel_id"], 1)
            return {"success": ok, "message": f"Restarted {srv}" if ok else err}
            
    elif t.startswith("kaggle_"):
        if a == "start":
            # Re-push worker code to start
            cmd = f"python3 /home/ubuntu/push_all_kaggle_workers.py"
            subprocess.run(cmd, shell=True)
            update_db_channel(cfg["channel_id"], 1)
            return {"success": True, "message": f"Triggered GPU Worker {cfg['name']}"}
        elif a == "stop":
            # Cancel kernel
            url = f"https://www.kaggle.com/api/v1/kernels/cancel?userName={cfg['username']}&kernelSlug={cfg['slug'].split('/')[-1]}"
            try:
                r = urllib.request.Request(url, method="POST", headers={"Authorization": f"Bearer {cfg['token']}"})
                urllib.request.urlopen(r, timeout=5)
            except Exception:
                pass
            update_db_channel(cfg["channel_id"], 2)
            return {"success": True, "message": f"Stopped {cfg['name']}"}
            
    return {"success": False, "message": "Unknown action"}

@app.get("/workers/api/logs/{target}")
@app.get("/api/logs/{target}")
def api_logs(target: str):
    if target not in CONFIG:
        raise HTTPException(status_code=400, detail="Unknown target")
    cfg = CONFIG[target]
    
    if target in ["ec2_text", "ec2_image"]:
        srv = cfg["service"]
        ok, out, err = ssh_exec(cfg["ip"], cfg["key"], f"sudo journalctl -u {srv} -n 40 --no-pager")
        return {"logs": out if ok else err}
    else:
        out = get_kaggle_output(cfg["username"], cfg["slug"], cfg["token"])
        return {"logs": out or "No output available yet"}

@app.get("/workers", response_class=HTMLResponse)
@app.get("/workers/", response_class=HTMLResponse)
@app.get("/", response_class=HTMLResponse)
def index_page():
    return HTML_CONTENT

HTML_CONTENT = """<!DOCTYPE html>
<html lang="en" class="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Tora API — AI Workers Control Panel</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script>
    tailwind.config = {
      darkMode: 'class',
      theme: {
        extend: {
          colors: {
            brand: { 500: '#6366f1', 600: '#4f46e5', 700: '#4338ca' }
          }
        }
      }
    }
  </script>
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
  <style>
    @keyframes pulse-fast { 0%, 100% { opacity: 1; } 50% { opacity: .4; } }
    .animate-pulse-fast { animation: pulse-fast 1.5s cubic-bezier(0.4, 0, 0.6, 1) infinite; }
  </style>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen font-sans antialiased">
  
  <!-- Header -->
  <header class="border-b border-slate-800 bg-slate-900/60 backdrop-blur sticky top-0 z-40">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
      <div class="flex items-center space-x-3">
        <div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-indigo-500 to-purple-500 flex items-center justify-center font-bold text-white shadow-lg shadow-indigo-500/20">
          <i class="fa-solid fa-server"></i>
        </div>
        <div>
          <h1 class="font-bold text-lg leading-tight flex items-center gap-2">
            Tora API <span class="text-xs bg-indigo-500/20 text-indigo-400 border border-indigo-500/30 px-2 py-0.5 rounded-full font-medium">Workers Hub</span>
          </h1>
          <p class="text-xs text-slate-400">AWS EC2 & Kaggle GPU Fleet Controller</p>
        </div>
      </div>
      
      <div class="flex items-center space-x-3">
        <button onclick="refreshAll()" id="refresh-btn" class="flex items-center gap-2 text-xs font-medium px-3.5 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg border border-slate-700 transition">
          <i class="fa-solid fa-rotate" id="refresh-icon"></i> Refresh Now
        </button>
        <span class="inline-flex items-center gap-1.5 text-xs text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-1.5 rounded-lg">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse-fast"></span> Live Auto (10s)
        </span>
      </div>
    </div>
  </header>

  <!-- Main Content -->
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
    
    <!-- Top Summary Banner -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="bg-slate-900/80 border border-slate-800/80 rounded-2xl p-4 shadow-sm">
        <div class="flex items-center justify-between text-slate-400 text-xs mb-1">
          <span>Total Fleet Workers</span>
          <i class="fa-solid fa-microchip text-slate-500"></i>
        </div>
        <div class="text-2xl font-bold text-white">6 <span class="text-xs font-normal text-slate-400">(2 AWS + 4 Kaggle)</span></div>
      </div>
      
      <div class="bg-slate-900/80 border border-slate-800/80 rounded-2xl p-4 shadow-sm">
        <div class="flex items-center justify-between text-slate-400 text-xs mb-1">
          <span>Active / Running</span>
          <i class="fa-solid fa-circle-check text-emerald-400"></i>
        </div>
        <div class="text-2xl font-bold text-emerald-400" id="stat-active">-</div>
      </div>

      <div class="bg-slate-900/80 border border-slate-800/80 rounded-2xl p-4 shadow-sm">
        <div class="flex items-center justify-between text-slate-400 text-xs mb-1">
          <span>Queued / Transition</span>
          <i class="fa-solid fa-clock text-amber-400"></i>
        </div>
        <div class="text-2xl font-bold text-amber-400" id="stat-queued">-</div>
      </div>

      <div class="bg-slate-900/80 border border-slate-800/80 rounded-2xl p-4 shadow-sm">
        <div class="flex items-center justify-between text-slate-400 text-xs mb-1">
          <span>Gateway Failover</span>
          <i class="fa-solid fa-shield-halved text-indigo-400"></i>
        </div>
        <div class="text-sm font-semibold text-indigo-300 mt-1.5">Auto Failover Active</div>
      </div>
    </div>

    <!-- Quick Actions Banner -->
    <div class="flex flex-wrap items-center justify-between bg-gradient-to-r from-slate-900 to-indigo-950/40 border border-slate-800 rounded-2xl p-4 gap-4">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-xl bg-indigo-500/20 text-indigo-400 flex items-center justify-center">
          <i class="fa-solid fa-bolt"></i>
        </div>
        <div>
          <h2 class="text-sm font-semibold text-slate-200">Fleet Quick Actions</h2>
          <p class="text-xs text-slate-400">Control all workers simultaneously</p>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <button onclick="controlAll('start')" class="text-xs font-medium px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg transition shadow-sm shadow-emerald-600/30 flex items-center gap-1.5">
          <i class="fa-solid fa-play"></i> Start All Workers
        </button>
        <button onclick="controlAll('stop')" class="text-xs font-medium px-4 py-2 bg-rose-600/80 hover:bg-rose-600 text-white rounded-lg transition shadow-sm flex items-center gap-1.5">
          <i class="fa-solid fa-stop"></i> Stop All Workers
        </button>
      </div>
    </div>

    <!-- Worker Cards Grid -->
    <div>
      <h3 class="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
        <i class="fa-brands fa-aws text-amber-400"></i> AWS Dedicated EC2 Instances
      </h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6" id="aws-grid">
        <!-- Rendered via JS -->
      </div>
    </div>

    <div>
      <h3 class="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
        <i class="fa-brands fa-kaggle text-sky-400"></i> Kaggle GPU Worker Fleet (Tesla T4)
      </h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6" id="kaggle-grid">
        <!-- Rendered via JS -->
      </div>
    </div>

  </main>

  <!-- Log Modal -->
  <div id="log-modal" class="fixed inset-0 bg-slate-950/80 backdrop-blur-sm z-50 hidden flex items-center justify-center p-4">
    <div class="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-4xl max-h-[85vh] flex flex-col shadow-2xl">
      <div class="flex items-center justify-between p-4 border-b border-slate-800">
        <div class="flex items-center gap-2.5">
          <i class="fa-solid fa-terminal text-indigo-400"></i>
          <h4 class="font-semibold text-sm" id="modal-title">Worker Live Logs</h4>
        </div>
        <button onclick="closeModal()" class="text-slate-400 hover:text-white p-1 text-lg">
          <i class="fa-solid fa-xmark"></i>
        </button>
      </div>
      <div class="p-4 flex-1 overflow-auto bg-black/60 font-mono text-xs text-slate-300 whitespace-pre-wrap leading-relaxed select-text" id="modal-content">
        Loading logs...
      </div>
      <div class="p-3 border-t border-slate-800 flex justify-end">
        <button onclick="closeModal()" class="text-xs px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg">Close</button>
      </div>
    </div>
  </div>

  <script>
    const WORKERS = [
      { id: 'ec2_text', name: 'AWS EC2 #1 — Qwen & Llama (Sydney)', type: 'aws', icon: 'fa-brain', badge: 'Graviton2', models: ['qwen2.5:7b', 'qwen2.5:3b', 'llama3.2:3b', 'deepseek-r1:1.5b'] },
      { id: 'ec2_image', name: 'AWS EC2 #2 — SD-Turbo Image (Sydney)', type: 'aws', icon: 'fa-palette', badge: 'Sapphire Rapids', models: ['sd-turbo', 'dall-e-2', 'sd-image'] },
      { id: 'kaggle_1', name: 'Kaggle Worker 1 (noppananthe)', type: 'kaggle', icon: 'fa-bolt', badge: 'Tesla T4 x2', models: ['qwen2.5:7b', 'deepseek-r1:7b'] },
      { id: 'kaggle_2', name: 'Kaggle Worker 2 (codeatheerachart)', type: 'kaggle', icon: 'fa-bolt', badge: 'Tesla T4', models: ['qwen2.5:7b', 'deepseek-r1:7b'] },
      { id: 'kaggle_3', name: 'Kaggle Worker 3 (weeblysdwebmagic)', type: 'kaggle', icon: 'fa-bolt', badge: 'Tesla T4', models: ['qwen2.5:7b', 'deepseek-r1:7b'] },
      { id: 'kaggle_4', name: 'Kaggle Worker 4 (creatorwebs)', type: 'kaggle', icon: 'fa-bolt', badge: 'Tesla T4', models: ['qwen2.5:7b', 'deepseek-r1:7b'] }
    ];

    function renderCards() {
      const awsEl = document.getElementById('aws-grid');
      const kaggleEl = document.getElementById('kaggle-grid');
      awsEl.innerHTML = '';
      kaggleEl.innerHTML = '';

      WORKERS.forEach(w => {
        const card = document.createElement('div');
        card.id = `card-${w.id}`;
        card.className = "bg-slate-900 border border-slate-800 rounded-2xl p-5 flex flex-col justify-between hover:border-slate-700 transition shadow-sm relative overflow-hidden";
        card.innerHTML = `
          <div>
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-xl bg-slate-800 flex items-center justify-center text-slate-300">
                  <i class="fa-solid ${w.icon}"></i>
                </div>
                <div>
                  <h4 class="font-semibold text-sm text-slate-100">${w.name}</h4>
                  <div class="text-xs text-slate-400 mt-0.5 flex items-center gap-2">
                    <span class="bg-slate-800 px-2 py-0.5 rounded text-[11px]">${w.badge}</span>
                    <span id="detail-${w.id}" class="text-slate-400">-</span>
                  </div>
                </div>
              </div>
              <span id="status-${w.id}" class="text-xs px-2.5 py-1 rounded-full font-medium inline-flex items-center gap-1.5 bg-slate-800 text-slate-400">
                <span class="w-1.5 h-1.5 rounded-full bg-slate-500"></span> Checking
              </span>
            </div>

            <!-- Model Pills -->
            <div class="mt-3 mb-5">
              <div class="text-[11px] uppercase tracking-wider text-slate-500 font-semibold mb-1.5">Served Models</div>
              <div class="flex flex-wrap gap-1.5">
                ${w.models.map(m => `<span class="text-xs bg-slate-800/80 text-slate-300 border border-slate-700/60 px-2 py-0.5 rounded-md font-mono">${m}</span>`).join('')}
              </div>
            </div>
          </div>

          <!-- Controls -->
          <div class="flex items-center justify-between pt-4 border-t border-slate-800/80">
            <div class="flex items-center gap-2">
              <button onclick="triggerAction('${w.id}', 'start')" class="px-3 py-1.5 bg-emerald-600/90 hover:bg-emerald-500 text-white rounded-lg text-xs font-medium transition flex items-center gap-1 shadow-sm">
                <i class="fa-solid fa-play text-[10px]"></i> Start
              </button>
              <button onclick="triggerAction('${w.id}', 'stop')" class="px-3 py-1.5 bg-rose-600/80 hover:bg-rose-600 text-white rounded-lg text-xs font-medium transition flex items-center gap-1">
                <i class="fa-solid fa-stop text-[10px]"></i> Stop
              </button>
              ${w.type === 'aws' ? `
              <button onclick="triggerAction('${w.id}', 'restart')" class="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg text-xs font-medium transition flex items-center gap-1">
                <i class="fa-solid fa-rotate-right text-[10px]"></i> Restart
              </button>` : ''}
            </div>

            <button onclick="openLogs('${w.id}', '${w.name}')" class="text-xs text-slate-400 hover:text-indigo-400 px-2.5 py-1.5 hover:bg-slate-800 rounded-lg transition flex items-center gap-1.5">
              <i class="fa-solid fa-terminal text-[11px]"></i> Logs
            </button>
          </div>
        `;

        if (w.type === 'aws') awsEl.appendChild(card);
        else kaggleEl.appendChild(card);
      });
    }

    async function refreshAll() {
      const btnIcon = document.getElementById('refresh-icon');
      btnIcon.classList.add('fa-spin');
      try {
        const res = await fetch('/workers/api/status');
        const data = await res.json();
        
        let activeCount = 0;
        let queuedCount = 0;

        Object.keys(data).forEach(k => {
          const item = data[k];
          const badge = document.getElementById(`status-${k}`);
          const detail = document.getElementById(`detail-${k}`);
          if (!badge) return;

          if (item.status === 'running') {
            activeCount++;
            badge.className = "text-xs px-2.5 py-1 rounded-full font-medium inline-flex items-center gap-1.5 bg-emerald-500/10 text-emerald-400 border border-emerald-500/20";
            badge.innerHTML = '<span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span> Active';
          } else if (item.status === 'queued') {
            queuedCount++;
            badge.className = "text-xs px-2.5 py-1 rounded-full font-medium inline-flex items-center gap-1.5 bg-amber-500/10 text-amber-400 border border-amber-500/20";
            badge.innerHTML = '<span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span> Queued';
          } else {
            badge.className = "text-xs px-2.5 py-1 rounded-full font-medium inline-flex items-center gap-1.5 bg-rose-500/10 text-rose-400 border border-rose-500/20";
            badge.innerHTML = '<span class="w-1.5 h-1.5 rounded-full bg-rose-400"></span> Stopped';
          }

          if (detail && item.detail) {
            detail.innerText = item.detail;
          }
        });

        document.getElementById('stat-active').innerText = activeCount;
        document.getElementById('stat-queued').innerText = queuedCount;
      } catch (e) {
        console.error("Refresh failed", e);
      } finally {
        setTimeout(() => btnIcon.classList.remove('fa-spin'), 400);
      }
    }

    async function triggerAction(target, action) {
      const badge = document.getElementById(`status-${target}`);
      if (badge) badge.innerText = "Applying...";
      try {
        const res = await fetch('/workers/api/action', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ target, action })
        });
        const d = await res.json();
        setTimeout(refreshAll, 1200);
      } catch (e) {
        alert("Action failed: " + e);
        refreshAll();
      }
    }

    async function controlAll(action) {
      if (!confirm(`Are you sure you want to ${action.toUpperCase()} all fleet workers?`)) return;
      for (const w of WORKERS) {
        await fetch('/workers/api/action', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ target: w.id, action })
        });
      }
      setTimeout(refreshAll, 1500);
    }

    async function openLogs(target, name) {
      document.getElementById('modal-title').innerText = name + ' — Logs';
      document.getElementById('modal-content').innerText = 'Fetching logs from remote node...';
      document.getElementById('log-modal').classList.remove('hidden');
      try {
        const res = await fetch(`/workers/api/logs/${target}`);
        const d = await res.json();
        document.getElementById('modal-content').innerText = d.logs || 'No log output returned.';
      } catch (e) {
        document.getElementById('modal-content').innerText = 'Error fetching logs: ' + e;
      }
    }

    function closeModal() {
      document.getElementById('log-modal').classList.add('hidden');
    }

    // Init
    renderCards();
    refreshAll();
    setInterval(refreshAll, 10000);
  </script>
</body>
</html>
"""
