import subprocess

cmd = ["docker", "exec", "postgres", "psql", "-U", "root", "-d", "new-api", "-t", "-A", "-c", 
       "SELECT key, value FROM options WHERE key LIKE '%Group%' OR key LIKE '%Register%' OR key LIKE '%Quota%';"]
out = subprocess.check_output(cmd).decode("utf-8").strip()
for line in out.split("\n"):
    if "|" in line:
        k, v = line.split("|", 1)
        print(f"{k}: {v}")
