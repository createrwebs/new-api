import urllib.request
import json

tokens = [
    ("websbringer", "KGAT_dd113GmLkaZvKbncgXkNj9hL8cQJCSCjXscK"),
    ("codeathee", "KGAT_f2113GmLkaZvKbncgXkNj9hL8cQJCSCjXscK"),
    ("softcreate9", "KGAT_554c5c65e6e6b8a953b38f3909647fb2"),
    ("createrwebs", "KGAT_8ecae0ef31fbd25f8d867eb59efca686"),
]

for name, tok in tokens:
    print(f"\n--- Testing {name} ---")
    for endpoint in [
        "https://api.kaggle.com/v1/models/list",
        "https://api.kaggle.com/v1/datasets/list",
        "https://api.kaggle.com/v1/kernels.KernelsApiService/ListKernels",
    ]:
        req = urllib.request.Request(
            endpoint,
            data=b"{}" if "Kernels" in endpoint else None,
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {tok}"
            }
        )
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                print(f"  {endpoint.split('/')[-1]}: SUCCESS (200)")
                break
        except urllib.error.HTTPError as e:
            print(f"  {endpoint.split('/')[-1]}: HTTP {e.code}")
        except Exception as e:
            print(f"  {endpoint.split('/')[-1]}: {e}")
