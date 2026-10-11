#!/usr/bin/env python3
"""
Antigravity Multi-Profile Agent Pool (agent_pool.py)
===================================================
ระบบสำหรับ AI Agent ในการเรียกใช้งาน Google Antigravity (agy)
แยกตาม Profile บัญชี พร้อมกันได้หลาย Instance (Concurrent / Parallel Execution)

ความสามารถ:
1. Isolate Environment: แยก HOME, USERPROFILE, CLOUDSDK_CONFIG และ Data Dir 100%
2. Concurrency Management: จัดการคิวงานและจำกัดจำนวน agent ที่รันพร้อมกัน (Semaphore)
3. Headless Non-Interactive: สั่งงานผ่าน CLI flags และรับ JSON Response กลับมา
4. Batch Processing: รับรายการงานแล้วกระจายไปรันบน Profile ต่างๆ พร้อมกัน
"""

import asyncio
import json
import os
import shutil
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional

# ป้องกัน UnicodeEncodeError บน Windows stdout/stderr (cp1252)
if sys.platform == "win32":
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8")



def find_agy_executable() -> str:
    """ค้นหาตำแหน่งของ agy executable"""
    agy_path = shutil.which("agy") or shutil.which("antigravity")
    if agy_path:
        return agy_path

    # ตรวจสอบตำแหน่ง default เพิ่มเติมบน Windows / Unix
    candidates = [
        Path.home() / "AppData" / "Local" / "agy" / "bin" / "agy.exe",
        Path.home() / ".local" / "bin" / "agy",
        Path("/usr/local/bin/agy"),
    ]
    for c in candidates:
        if c.is_file():
            return str(c)

    raise FileNotFoundError(
        "ไม่พบคำสั่ง 'agy' หรือ 'antigravity' กรุณาตรวจสอบการติดตั้ง Antigravity CLI"
    )


@dataclass
class ProfileConfig:
    """คอนฟิกของแต่ละ Profile"""
    name: str
    base_dir: Path
    workspace: Path
    gemini_api_key: Optional[str] = None
    extra_env: Dict[str, str] = field(default_factory=dict)

    @property
    def home_dir(self) -> Path:
        return self.base_dir / "home"

    @property
    def gcloud_dir(self) -> Path:
        return self.base_dir / "gcloud"

    @property
    def gemini_data_dir(self) -> Path:
        return self.home_dir / ".gemini" / "antigravity"

    def ensure_directories(self) -> None:
        """สร้างโฟลเดอร์ที่จำเป็นทั้งหมด"""
        self.home_dir.mkdir(parents=True, exist_ok=True)
        self.gcloud_dir.mkdir(parents=True, exist_ok=True)
        self.gemini_data_dir.mkdir(parents=True, exist_ok=True)
        self.workspace.mkdir(parents=True, exist_ok=True)

    def get_isolated_env(self) -> Dict[str, str]:
        """สร้าง environment dictionary ที่แยก config ออกจาก profile อื่นเด็ดขาด"""
        self.ensure_directories()
        env = os.environ.copy()

        # Isolate paths
        home_str = str(self.home_dir)
        env["HOME"] = home_str
        env["USERPROFILE"] = home_str
        env["CLOUDSDK_CONFIG"] = str(self.gcloud_dir)
        env["ANTIGRAVITY_EXECUTABLE_DATA_DIR"] = str(self.gemini_data_dir)

        # Isolated API Key (ถ้ามีระบุเฉพาะ profile)
        if self.gemini_api_key:
            env["GEMINI_API_KEY"] = self.gemini_api_key

        # Extra environment variables
        for k, v in self.extra_env.items():
            env[k] = v

        return env


@dataclass
class AgentTask:
    """โครงสร้างงานที่ต้องการส่งให้ Agent รัน"""
    task_id: str
    profile_name: str
    prompt: str
    workspace: Optional[Path] = None
    timeout_seconds: Optional[int] = None
    dangerously_skip_permissions: bool = True
    effort: Optional[str] = None  # low | medium | high | xhigh | max
    model: Optional[str] = None


@dataclass
class AgentResult:
    """ผลลัพธ์การทำงานของ Agent"""
    task_id: str
    profile_name: str
    exit_code: int
    success: bool
    output: str
    error: str
    duration_seconds: float


class AntigravityAgentPool:
    """
    Agent Pool Orchestrator
    จัดการการรัน Agent พร้อมกันตาม Profile
    """

    def __init__(
        self,
        base_profiles_dir: Optional[Path] = None,
        max_concurrent_agents: int = 4,
    ):
        self.base_profiles_dir = (
            base_profiles_dir or (Path.home() / ".antigravity-profiles")
        )
        self.max_concurrent_agents = max_concurrent_agents
        self.semaphore = asyncio.Semaphore(max_concurrent_agents)
        self.profiles: Dict[str, ProfileConfig] = {}
        self.agy_bin = find_agy_executable()

    def register_profile(
        self,
        name: str,
        workspace: Optional[Path] = None,
        gemini_api_key: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
    ) -> ProfileConfig:
        """ลงทะเบียน Profile ใหม่เข้า Pool"""
        base_dir = self.base_profiles_dir / name
        ws = workspace or (Path.home() / "projects" / name)
        config = ProfileConfig(
            name=name,
            base_dir=base_dir,
            workspace=ws,
            gemini_api_key=gemini_api_key,
            extra_env=extra_env or {},
        )
        config.ensure_directories()
        self.profiles[name] = config
        return config

    def get_or_create_profile(self, name: str) -> ProfileConfig:
        """ค้นหาหรือสร้าง Profile อัตโนมัติ"""
        if name not in self.profiles:
            return self.register_profile(name)
        return self.profiles[name]

    async def execute_task(self, task: AgentTask) -> AgentResult:
        """
        รันงาน 1 งานบน Profile ที่กำหนดแบบ Asynchronous Subprocess
        """
        async with self.semaphore:
            profile = self.get_or_create_profile(task.profile_name)
            env = profile.get_isolated_env()
            target_workspace = task.workspace or profile.workspace
            target_workspace.mkdir(parents=True, exist_ok=True)

            cmd = [
                self.agy_bin,
                "-p",
                task.prompt,
                "--output-format",
                "text",
            ]

            if task.dangerously_skip_permissions:
                cmd.append("--dangerously-skip-permissions")

            if task.effort:
                cmd.extend(["--effort", task.effort])

            if task.model:
                cmd.extend(["--model", task.model])

            start_time = asyncio.get_event_loop().time()

            try:
                proc = await asyncio.create_subprocess_exec(
                    *cmd,
                    cwd=str(target_workspace),
                    env=env,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE,
                )

                if task.timeout_seconds:
                    stdout, stderr = await asyncio.wait_for(
                        proc.communicate(), timeout=task.timeout_seconds
                    )
                else:
                    stdout, stderr = await proc.communicate()

                duration = asyncio.get_event_loop().time() - start_time
                exit_code = proc.returncode or 0

                return AgentResult(
                    task_id=task.task_id,
                    profile_name=task.profile_name,
                    exit_code=exit_code,
                    success=(exit_code == 0),
                    output=stdout.decode("utf-8", errors="replace"),
                    error=stderr.decode("utf-8", errors="replace"),
                    duration_seconds=duration,
                )

            except asyncio.TimeoutError:
                duration = asyncio.get_event_loop().time() - start_time
                if proc:
                    proc.kill()
                return AgentResult(
                    task_id=task.task_id,
                    profile_name=task.profile_name,
                    exit_code=-1,
                    success=False,
                    output="",
                    error=f"Task timed out after {task.timeout_seconds}s",
                    duration_seconds=duration,
                )
            except Exception as e:
                duration = asyncio.get_event_loop().time() - start_time
                return AgentResult(
                    task_id=task.task_id,
                    profile_name=task.profile_name,
                    exit_code=-2,
                    success=False,
                    output="",
                    error=str(e),
                    duration_seconds=duration,
                )

    async def execute_batch(self, tasks: List[AgentTask]) -> List[AgentResult]:
        """
        รันงานหลายรายการพร้อมกันในเวลาเดียวกัน (Concurrent Batch)
        """
        coroutines = [self.execute_task(t) for t in tasks]
        return await asyncio.gather(*coroutines)


# ==============================================================================
# CLI Entry Point
# ==============================================================================
async def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Antigravity Multi-Profile Agent Runner"
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    # Subcommand: run
    run_parser = subparsers.add_parser("run", help="รันคำสั่ง 1 prompt บน Profile ที่กำหนด")
    run_parser.add_argument("--profile", required=True, help="ชื่อโปรไฟล์ (เช่น acc1, acc2)")
    run_parser.add_argument("--workspace", help="Path workspace โฟลเดอร์งาน")
    run_parser.add_argument("--prompt", required=True, help="ข้อความ prompt สั่งงาน")
    run_parser.add_argument("--effort", choices=["low", "medium", "high", "xhigh", "max"])
    run_parser.add_argument("--model", help="โมเดลที่ต้องการใช้")
    run_parser.add_argument("--timeout", type=int, default=300, help="Timeout วินาที")

    # Subcommand: batch
    batch_parser = subparsers.add_parser("batch", help="รันรายการงานแบบ JSON หลายโปรไฟล์พร้อมกัน")
    batch_parser.add_argument("--file", required=True, help="Path ไปยัง JSON ไฟล์ที่เก็บรายการ tasks")
    batch_parser.add_argument("--max-concurrent", type=int, default=3, help="จำนวน concurrent agents สูงสุด")

    # Subcommand: init
    init_parser = subparsers.add_parser("init", help="เตรียมสร้างโฟลเดอร์ config ให้ Profile")
    init_parser.add_argument("--profile", required=True, help="ชื่อโปรไฟล์ที่ต้องการสร้าง")

    args = parser.parse_args()
    pool = AntigravityAgentPool()

    if args.command == "init":
        p = pool.register_profile(args.profile)
        print(f"✅ Initialized profile '{args.profile}' successfully.")
        print(f"  Config Base: {p.base_dir}")
        print(f"  Isolated Home: {p.home_dir}")
        print(f"  Isolated gcloud: {p.gcloud_dir}")

    elif args.command == "run":
        ws = Path(args.workspace) if args.workspace else None
        task = AgentTask(
            task_id="single-run",
            profile_name=args.profile,
            prompt=args.prompt,
            workspace=ws,
            timeout_seconds=args.timeout,
            effort=args.effort,
            model=args.model,
        )
        print(f"🚀 Running task on profile '{args.profile}'...")
        res = await pool.execute_task(task)
        print(f"Status: {'SUCCESS' if res.success else 'FAILED'} (Exit code: {res.exit_code}, Duration: {res.duration_seconds:.2f}s)")
        if res.output:
            print("\n--- Output ---")
            print(res.output)
        if res.error:
            print("\n--- Errors/Logs ---", file=sys.stderr)
            print(res.error, file=sys.stderr)
        sys.exit(res.exit_code)

    elif args.command == "batch":
        with open(args.file, "r", encoding="utf-8") as f:
            data = json.load(f)

        tasks = []
        for idx, item in enumerate(data):
            tasks.append(
                AgentTask(
                    task_id=item.get("id", f"task-{idx+1}"),
                    profile_name=item["profile"],
                    prompt=item["prompt"],
                    workspace=Path(item["workspace"]) if item.get("workspace") else None,
                    timeout_seconds=item.get("timeout", 300),
                    effort=item.get("effort"),
                    model=item.get("model"),
                )
            )

        pool.max_concurrent_agents = args.max_concurrent
        pool.semaphore = asyncio.Semaphore(args.max_concurrent)

        print(f"🚀 Running {len(tasks)} tasks concurrently (Max {args.max_concurrent} at a time)...")
        results = await pool.execute_batch(tasks)

        summary = [
            {
                "task_id": r.task_id,
                "profile": r.profile_name,
                "success": r.success,
                "exit_code": r.exit_code,
                "duration": round(r.duration_seconds, 2),
                "output_snippet": r.output[:200] if r.output else "",
            }
            for r in results
        ]
        print("\n=== Batch Summary ===")
        print(json.dumps(summary, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    asyncio.run(main())
