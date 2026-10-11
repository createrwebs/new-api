import os
import subprocess

accounts = [
    ("websbringer", "KGAT_dd113GmLkaZvKbncgXkNj9hL8cQJCSCjXscK"),
    ("codeathee", "KGAT_f2113GmLkaZvKbncgXkNj9hL8cQJCSCjXscK"),
    ("softcreate9", "KGAT_554c5c65e6e6b8a953b38f3909647fb2"),
    ("createrwebs", "KGAT_8ecae0ef31fbd25f8d867eb59efca686"),
]

for user, token in accounts:
    env = os.environ.copy()
    env["KAGGLE_API_TOKEN"] = token
    for k in ["KAGGLE_KEY", "KAGGLE_USERNAME"]:
        env.pop(k, None)
    try:
        out = subprocess.check_output(
            ["python", "-m", "kaggle", "kernels", "list", "--mine"],
            env=env, stderr=subprocess.STDOUT, timeout=10
        ).decode("utf-8")
        print(f"[{user}] SUCCESS:\n{out.strip()}\n")
    except subprocess.CalledProcessError as e:
        print(f"[{user}] FAILED: {e.output.decode('utf-8').strip()}\n")
    except Exception as e:
        print(f"[{user}] ERROR: {e}\n")
