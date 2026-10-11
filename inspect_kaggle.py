import urllib.request
import json

url = 'https://api.kaggle.com/v1/kernels.KernelsApiService/ListKernels'
tokens = [
    ('softcreate9', 'KGAT_554c5c65e6e6b8a953b38f3909647fb2'),
    ('createrwebs', 'KGAT_8ecae0ef31fbd25f8d867eb59efca686'),
]

for user, token in tokens:
    payload = json.dumps({'user': user, 'pageSize': 20}).encode('utf-8')
    req = urllib.request.Request(url, data=payload, headers={
        'Content-Type': 'application/json',
        'Authorization': f'Bearer {token}'
    })
    try:
        with urllib.request.urlopen(req) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            print(f"\nUser: {user} - Found {len(data.get('kernels', []))} kernels:")
            for k in data.get('kernels', []):
                print(f"  - Ref: {k.get('ref')}, Title: {k.get('title')}, URL: {k.get('url')}")
    except Exception as e:
        print(f"Error for {user}: {e}")
