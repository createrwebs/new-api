# OpenCode setup

Configure ToraApi as an OpenAI-compatible provider in OpenCode.

## 1. Create an API key

Create a ToraApi API key whose non-default group can access `gpt-5.6-sol`.

## 2. Install OpenCode

**Windows PowerShell**

~~~powershell
choco install opencode
opencode --version
~~~

**Linux or macOS**

~~~bash
curl -fsSL https://opencode.ai/install | bash
opencode --version
~~~

## 3. Create `opencode.json`

Enter your project directory, then run the command for your system.

**Windows PowerShell**

~~~powershell
$config = @'
{
  "$schema": "https://opencode.ai/config.json",
  "model": "ToraApi/gpt-5.6-sol",
  "provider": {
    "ToraApi": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "ToraApi",
      "options": { "baseURL": "https://toraapi.com/v1" },
      "models": {
        "gpt-5.6-sol": { "name": "GPT 5.6 Sol" }
      }
    }
  }
}
'@
$utf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
[System.IO.File]::WriteAllText(
  (Join-Path (Get-Location) "opencode.json"),
  $config,
  $utf8NoBom
)
~~~

**Linux or macOS**

~~~bash
cat > opencode.json << 'EOF'
{
  "$schema": "https://opencode.ai/config.json",
  "model": "ToraApi/gpt-5.6-sol",
  "provider": {
    "ToraApi": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "ToraApi",
      "options": { "baseURL": "https://toraapi.com/v1" },
      "models": {
        "gpt-5.6-sol": { "name": "GPT 5.6 Sol" }
      }
    }
  }
}
EOF
~~~

## 4. Connect your API key

Start OpenCode in the same project directory:

~~~bash
opencode
~~~

Enter `/connect`, choose **Other**, set the provider id to `ToraApi`, and paste your ToraApi API key.

## 5. Select the model and test

Enter `/models`, select `ToraApi/gpt-5.6-sol`, and send:

~~~text
Reply with one short greeting.
~~~

A normal reply means setup is complete.