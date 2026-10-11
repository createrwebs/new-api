import subprocess
import torch
print('=== GPU DIAGNOSTIC START ===')
print('Torch Version:', torch.__version__)
print('CUDA Available:', torch.cuda.is_available())
if torch.cuda.is_available():
    print('Device Name:', torch.cuda.get_device_name(0))
    print('Device Count:', torch.cuda.device_count())

res = subprocess.run(['nvidia-smi'], capture_output=True, text=True)
print('NVIDIA-SMI output:')
print(res.stdout)
print('=== GPU DIAGNOSTIC END ===')
