import subprocess

sql = "UPDATE tokens SET key = 'toraapiNewUsersPoolKey2026' WHERE id = 5 RETURNING id, key, \"group\";"
out = subprocess.check_output(["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-c", sql]).decode("utf-8")
print(out)
