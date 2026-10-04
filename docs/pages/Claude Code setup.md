# Claude Code setup

Connect Claude Code through Anthropic Messages with an explicit model alias for background tasks.

This guide connects Claude Code to ToraApi and sends the first successful request. Use Option A for automated setup. Use Option B to install Claude Code first, then create the user settings yourself.

## 1. Create an API key

Create a ToraApi API key whose non-default group can access `claude-opus-4-8` and `claude-sonnet-5`. Keep the complete key ready; the installer asks for it once.

## 2. Choose a setup method

### Option A: run the installer

Not comfortable with the terminal yet? Start with the [interactive first-run guide](/guides/clients/claude-code/start). It is only a visual rehearsal and does not change your computer.

**macOS or Linux**

Run this command as your normal user:

~~~bash
curl -fsSL https://toraapi.com/install/claude.sh | sh
~~~

When you are already logged in as `root` on a VPS or container, use:

~~~bash
curl -fsSL https://toraapi.com/install/claude.sh | ToraApi_ALLOW_ROOT=1 sh
~~~

Do not add `sudo` on a personal computer.

**Windows PowerShell**

Open PowerShell as your normal Windows user and run:

~~~powershell
irm https://toraapi.com/install/claude.ps1 | iex
~~~

The installer asks you to choose Simplified Chinese, Japanese, or English, installs the official Claude Code CLI when needed, asks for the ToraApi API key once, backs up and merges the existing user settings, and verifies both configured models. It does not start Claude Code automatically.

### Option B: configure Claude Code manually

Follow each step in order and run only the command for your operating system.

#### Step 1: install Claude Code

**Windows PowerShell**

~~~powershell
irm https://claude.ai/install.ps1 | iex
claude --version
~~~

**Linux**

~~~bash
curl -fsSL https://claude.ai/install.sh | bash
claude --version
~~~

**macOS**

~~~bash
curl -fsSL https://claude.ai/install.sh | bash
claude --version
~~~

#### Step 2: create and enter the user settings directory

**Windows PowerShell**

~~~powershell
New-Item -ItemType Directory -Force "$HOME\.claude" | Out-Null
Set-Location "$HOME\.claude"
~~~

**Linux**

~~~bash
mkdir -p ~/.claude
cd ~/.claude
~~~

**macOS**

~~~bash
mkdir -p ~/.claude
cd ~/.claude
~~~

#### Step 3: create `settings.json`

Before running the command for your system, replace `YOUR_ToraApi_API_KEY` with your complete ToraApi API key.

**Windows PowerShell**

~~~powershell
$settings = @'
{
  "env": {
    "ANTHROPIC_BASE_URL": "https://toraapi.com",
    "ANTHROPIC_AUTH_TOKEN": "YOUR_ToraApi_API_KEY",
    "ANTHROPIC_MODEL": "claude-opus-4-8",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-opus-4-8",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-sonnet-5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-sonnet-5"
  }
}
'@
$utf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
[System.IO.File]::WriteAllText(
  (Join-Path (Get-Location) "settings.json"),
  $settings,
  $utf8NoBom
)
~~~

**Linux**

~~~bash
cat > settings.json << 'EOF'
{
  "env": {
    "ANTHROPIC_BASE_URL": "https://toraapi.com",
    "ANTHROPIC_AUTH_TOKEN": "YOUR_ToraApi_API_KEY",
    "ANTHROPIC_MODEL": "claude-opus-4-8",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-opus-4-8",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-sonnet-5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-sonnet-5"
  }
}
EOF
chmod 600 settings.json
~~~

**macOS**

~~~bash
cat > settings.json << 'EOF'
{
  "env": {
    "ANTHROPIC_BASE_URL": "https://toraapi.com",
    "ANTHROPIC_AUTH_TOKEN": "YOUR_ToraApi_API_KEY",
    "ANTHROPIC_MODEL": "claude-opus-4-8",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-opus-4-8",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-sonnet-5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-sonnet-5"
  }
}
EOF
chmod 600 settings.json
~~~

#### Step 4: start Claude Code

**Windows PowerShell**

~~~powershell
Set-Location $HOME
New-Item -ItemType Directory -Force "my-claude-project" | Out-Null
Set-Location "my-claude-project"
claude
~~~

**Linux**

~~~bash
cd ~
mkdir -p my-claude-project
cd my-claude-project
claude
~~~

**macOS**

~~~bash
cd ~
mkdir -p my-claude-project
cd my-claude-project
claude
~~~

## 3. Send the first request

After Option A finishes, enter a project directory you trust and start Claude Code yourself:

~~~bash
claude
~~~

With either method, send:

~~~text
Reply with one sentence confirming the active model.
~~~

A normal reply means setup is complete.