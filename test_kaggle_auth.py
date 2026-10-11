import os
import sys

ACCOUNTS = [
    ("websbringer", "KGAT_dd113GmLkaZvKbncgXkNj9hL8cQJCSCjXscK"),
    ("codeathee", "KGAT_f2113GmLkaZvKbncgXkNj9hL8cQJCSCjXscK"),
    ("softcreate9", "KGAT_554c5c65e6e6b8a953b38f3909647fb2"),
    ("createrwebs", "KGAT_8ecae0ef31fbd25f8d867eb59efca686"),
]

for username, key in ACCOUNTS:
    print(f"\n--- Testing Kaggle account: {username} ---")
    os.environ["KAGGLE_USERNAME"] = username
    os.environ["KAGGLE_KEY"] = key
    try:
        from kaggle.api.kaggle_api_extended import KaggleApi
        api = KaggleApi()
        api.authenticate()
        print(f"SUCCESS: Authenticated as {username}!")
        
        # Check existing kernels
        try:
            kernels = api.kernels_list(user=username)
            print(f"Kernels ({len(kernels)}): {[k.ref for k in kernels]}")
        except Exception as e:
            print(f"Kernels list error: {e}")
            
    except Exception as e:
        print(f"FAILED for {username}: {e}")
