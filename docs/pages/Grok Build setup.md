# Grok Build setup

Configure the official Grok Build CLI with the Responses API endpoint.

## 1. Create an API key

Create a ToraApi API key that can access `grok-4.5`.

## 2. Install Grok Build

**Windows PowerShell**

~~~powershell
irm https://x.ai/cli/install.ps1 | iex
grok version
~~~

**Linux or macOS**

~~~bash
curl -fsSL https://x.ai/cli/install.sh | bash
grok version
~~~

## 3. Create `config.toml`

**Windows PowerShell**

~~~powershell
New-Item -ItemType Directory -Force "$HOME\.grok" | Out-Null
Set-Location "$HOME\.grok"
$config = @'
[models]
default = "ToraApi-grok"

[model.ToraApi-grok]
model = "grok-4.5"
base_url = "https://toraapi.com/v1"
name = "ToraApi Grok 4.5"
env_key = "ToraApi_API_KEY"
api_backend = "responses"
'@
$utf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
[System.IO.File]::WriteAllText(
  (Join-Path (Get-Location) "config.toml"),
  $config,
  $utf8NoBom
)
~~~

**Linux or macOS**

~~~bash
mkdir -p ~/.grok
cd ~/.grok
cat > config.toml << 'EOF'
[models]
default = "ToraApi-grok"

[model.ToraApi-grok]
model = "grok-4.5"
base_url = "https://toraapi.com/v1"
name = "ToraApi Grok 4.5"
env_key = "ToraApi_API_KEY"
api_backend = "responses"
EOF
chmod 600 config.toml
~~~

## 4. Save your API key

Replace `YOUR_ToraApi_API_KEY` with your complete ToraApi API key.

**Windows PowerShell**

~~~powershell
[Environment]::SetEnvironmentVariable(
  "ToraApi_API_KEY",
  "YOUR_ToraApi_API_KEY",
  "User"
)
$env:ToraApi_API_KEY = "YOUR_ToraApi_API_KEY"
~~~

**Linux**

~~~bash
echo 'export ToraApi_API_KEY="YOUR_ToraApi_API_KEY"' >> ~/.bashrc
source ~/.bashrc
~~~

**macOS**

~~~bash
echo 'export ToraApi_API_KEY="YOUR_ToraApi_API_KEY"' >> ~/.zshrc
source ~/.zshrc
~~~

## 5. Start and test

**Windows PowerShell**

~~~powershell
Set-Location $HOME
grok -p "Reply only: ToraApi connected" -m ToraApi-grok
grok
~~~

**Linux or macOS**

~~~bash
cd ~
grok -p "Reply only: ToraApi connected" -m ToraApi-grok
grok
~~~