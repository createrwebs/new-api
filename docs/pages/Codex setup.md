# Codex setup

Use the installer or configure Codex manually with a ToraApi key and a Responses-compatible model.

This guide gets Codex connected to ToraApi and running its first request. Use Option A for automated setup. Use Option B to install Codex first, then create each required file yourself.

## 1. Create an API key

Sign in to the ToraApi console, open **API Keys**, and create a key that can access `gpt-5.6-sol`. Choose any available non-default group that lists the model; there is no single required group.

Keep the key ready. You can use the installer or configure the same files manually.

## 2. Choose a setup method

### Option A: run the installer

Not comfortable with the terminal yet? Start with the [interactive first-run guide](/guides/clients/codex/start). It is only a visual rehearsal and does not change your computer.

**macOS or Linux**

Run this command as your normal user:

~~~bash
curl -fsSL https://ToraApi.com/install/codex.sh | sh
~~~

When you are already logged in as `root` on a VPS or container, use:

~~~bash
curl -fsSL https://ToraApi.com/install/codex.sh | ToraApi_ALLOW_ROOT=1 sh
~~~

Do not add `sudo` on a normal personal computer. The script installs or updates Codex, backs up existing files, asks for your ToraApi API key, and writes the provider configuration under `~/.codex`.

**Windows PowerShell**

Open PowerShell as your normal Windows user and run:

~~~powershell
irm https://ToraApi.com/install/codex.ps1 | iex
~~~

The installer asks for a language and your ToraApi API key, verifies access to `gpt-5.6-sol`, and backs up existing files before changing them. When it finishes, continue to **Verify the connection**.

### Option B: create each Codex file yourself

Use this option when you do not want to run the ToraApi setup script. Follow the steps in order and use the command block for your operating system in every step.

Option B creates two files under your user-level `.codex` directory: `config.toml` and `auth.json`.

#### Step 1: install Codex CLI

**Windows PowerShell**

~~~powershell
irm https://chatgpt.com/codex/install.ps1 | iex
codex --version
~~~

**Linux**

~~~bash
curl -fsSL https://chatgpt.com/codex/install.sh | sh
codex --version
~~~

**macOS**

~~~bash
curl -fsSL https://chatgpt.com/codex/install.sh | sh
codex --version
~~~

#### Step 2: create and enter the `.codex` directory

**Windows PowerShell**

~~~powershell
New-Item -ItemType Directory -Force "$HOME\.codex" | Out-Null
Set-Location "$HOME\.codex"
~~~

**Linux**

~~~bash
mkdir -p ~/.codex
cd ~/.codex
~~~

**macOS**

~~~bash
mkdir -p ~/.codex
cd ~/.codex
~~~

#### Step 3: create `config.toml`

**Windows PowerShell**

~~~powershell
$config = @'
cli_auth_credentials_store = "file"
model_provider = "ToraApi"
model = "gpt-5.6-sol"
model_reasoning_effort = "xhigh"
disable_response_storage = true
model_supports_reasoning_summaries = true

[model_providers.ToraApi]
name = "ToraApi"
base_url = "https://ToraApi.com/v1"
wire_api = "responses"
requires_openai_auth = true
'@
$utf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
[System.IO.File]::WriteAllText(
  (Join-Path (Get-Location) "config.toml"),
  $config,
  $utf8NoBom
)
~~~

**Linux**

~~~bash
cat > config.toml << 'EOF'
cli_auth_credentials_store = "file"
model_provider = "ToraApi"
model = "gpt-5.6-sol"
model_reasoning_effort = "xhigh"
disable_response_storage = true
model_supports_reasoning_summaries = true

[model_providers.ToraApi]
name = "ToraApi"
base_url = "https://ToraApi.com/v1"
wire_api = "responses"
requires_openai_auth = true
EOF
chmod 600 config.toml
~~~

**macOS**

~~~bash
cat > config.toml << 'EOF'
cli_auth_credentials_store = "file"
model_provider = "ToraApi"
model = "gpt-5.6-sol"
model_reasoning_effort = "xhigh"
disable_response_storage = true
model_supports_reasoning_summaries = true

[model_providers.ToraApi]
name = "ToraApi"
base_url = "https://ToraApi.com/v1"
wire_api = "responses"
requires_openai_auth = true
EOF
chmod 600 config.toml
~~~

#### Step 4: create `auth.json`

Replace `YOUR_ToraApi_API_KEY` in the JSON object with the complete API key you created in the ToraApi console. This file must remain valid JSON and use the `OPENAI_API_KEY` field; do not add a provider `auth` command block. Current Codex releases require `requires_openai_auth = true` for this API-key path.

**Windows PowerShell**

~~~powershell
$authJson = @'
{
  "auth_mode": "apikey",
  "OPENAI_API_KEY": "YOUR_ToraApi_API_KEY"
}
'@
$utf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
[System.IO.File]::WriteAllText(
  (Join-Path (Get-Location) "auth.json"),
  $authJson,
  $utf8NoBom
)
~~~

**Linux**

~~~bash
cat > auth.json << 'EOF'
{
  "auth_mode": "apikey",
  "OPENAI_API_KEY": "YOUR_ToraApi_API_KEY"
}
EOF
chmod 600 auth.json
~~~

**macOS**

~~~bash
cat > auth.json << 'EOF'
{
  "auth_mode": "apikey",
  "OPENAI_API_KEY": "YOUR_ToraApi_API_KEY"
}
EOF
chmod 600 auth.json
~~~

#### Step 5: start Codex

**Windows PowerShell**

~~~powershell
Set-Location $HOME
New-Item -ItemType Directory -Force "my-codex-project" | Out-Null
Set-Location "my-codex-project"
codex --model gpt-5.6-sol
~~~

**Linux**

~~~bash
cd ~
mkdir -p my-codex-project
cd my-codex-project
codex --model gpt-5.6-sol
~~~

**macOS**

~~~bash
cd ~
mkdir -p my-codex-project
cd my-codex-project
codex --model gpt-5.6-sol
~~~

## 3. Verify the connection

If you used the installer, fully close any Codex CLI, IDE, or desktop session that was already open. Open a new terminal, enter a project directory, and run:

~~~bash
codex --model gpt-5.6-sol
~~~

After finishing setup, you can also [download the Codex App](https://developers.openai.com/codex/app).

Then ask Codex:

~~~text
Reply with one sentence confirming the active model and provider.
~~~

A normal response means setup is complete.

## Common problems

| Symptom | What to do |
| --- | --- |
| `401` | Recopy the API key and confirm that it has not been deleted or disabled. |
| `model_not_found` | Edit the API key and choose a group that can access `gpt-5.6-sol`. |
| ChatGPT login appears | Fully quit every Codex session and reopen the CLI. |
| `codex` is not found | Open a new terminal. On Windows, try `codex.cmd`. |