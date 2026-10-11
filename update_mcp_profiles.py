import os
import re

files = [
    r"C:\Users\theer\.gemini\settings.json",
    r"C:\Users\theer\.claude.json",
    r"C:\Users\theer\.cline\mcp.json",
    r"C:\Users\theer\.cursor\mcp.json",
    r"C:\Users\theer\.kiro\settings\mcp.json",
    r"C:\Users\theer\.config\opencode\opencode.json"
]
for f in files:
    if os.path.exists(f):
        with open(f, "r", encoding="utf-8") as fp:
            content = fp.read()
        new_content = re.sub(r'"AWS_MCP_PROXY_PROFILES":\s*"[^"]*"', '"AWS_MCP_PROXY_PROFILES": "toraapi,toraapi-image"', content)
        with open(f, "w", encoding="utf-8") as fp:
            fp.write(new_content)
        print("Updated:", f)

codex = r"C:\Users\theer\.codex\config.toml"
if os.path.exists(codex):
    with open(codex, "r", encoding="utf-8") as fp:
        c = fp.read()
    c = re.sub(r'AWS_MCP_PROXY_PROFILES\s*=\s*"[^"]*"', 'AWS_MCP_PROXY_PROFILES = "toraapi,toraapi-image"', c)
    with open(codex, "w", encoding="utf-8") as fp:
        fp.write(c)
    print("Updated:", codex)
