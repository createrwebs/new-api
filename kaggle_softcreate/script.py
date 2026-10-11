import subprocess
import torch

print("=== GPU CHECK START ===")
print("Torch:", torch.__version__)
print("CUDA Available:", torch.cuda.is_available())
if torch.cuda.is_available():
    print("Device Name:", torch.cuda.get_device_name(0))
    print("Memory (GB):", torch.cuda.get_device_properties(0).total_memory / 1e9)
try:
    res = subprocess.run(["nvidia-smi"], capture_output=True, text=True)
    print("nvidia-smi:\n", res.stdout)
except Exception as e:
    print("nvidia-smi error:", e)

# Test internet connectivity
import urllib.request
try:
    with urllib.request.urlopen("https://www.google.com", timeout=5) as r:
        print("Internet connectivity: OK, status", r.status)
except Exception as e:
    print("Internet connectivity error:", e)

print("=== GPU CHECK END ===")
