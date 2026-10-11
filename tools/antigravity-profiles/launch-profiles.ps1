<#
.SYNOPSIS
    launch-profiles.ps1 - Antigravity Multi-Profile Launcher for Windows / PowerShell

.DESCRIPTION
    รัน Antigravity CLI (agy.exe) แยก Profile อิสระจากกัน
    แยก Config (.gemini), gcloud, workspace และ credentials สำหรับแต่ละบัญชี
#>

[CmdletBinding()]
param(
    [string]$BaseConfigDir = "$HOME\.antigravity-profiles",
    [string]$Workspace1 = "$HOME\projects\workspace-1",
    [string]$Workspace2 = "$HOME\projects\workspace-2"
)

# ค้นหา executable ของ agy
$AgyCmd = Get-Command agy, agy.exe -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -First 1
if (-not $AgyCmd) {
    $fallbackPath = "$env:LOCALAPPDATA\agy\bin\agy.exe"
    if (Test-Path $fallbackPath) {
        $AgyCmd = $fallbackPath
    } else {
        Write-Error "ไม่พบ agy.exe ใน PATH หรือ $fallbackPath กรุณาตรวจสอบการติดตั้ง"
        exit 1
    }
}

# ฟังก์ชันสร้างโฟลเดอร์สำหรับ Profile
function Initialize-Profile {
    param([string]$ProfileName)
    $profileDir = Join-Path $BaseConfigDir $ProfileName
    New-Item -ItemType Directory -Force -Path (Join-Path $profileDir "gcloud") | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $profileDir "home\.gemini") | Out-Null
    return $profileDir
}

$dir1 = Initialize-Profile -ProfileName "acc1"
$dir2 = Initialize-Profile -ProfileName "acc2"

# สร้าง command สำหรับแต่ละ profile
function Get-ProfileScriptBlock {
    param(
        [string]$ProfileName,
        [string]$ProfileDir,
        [string]$Workspace
    )
    New-Item -ItemType Directory -Force -Path $Workspace | Out-Null

    $homeDir = Join-Path $ProfileDir "home"
    $gcloudDir = Join-Path $ProfileDir "gcloud"
    $dataDir = Join-Path $homeDir ".gemini\antigravity"

    return @"
`$env:CLOUDSDK_CONFIG = '$gcloudDir'
`$env:USERPROFILE = '$homeDir'
`$env:HOME = '$homeDir'
`$env:ANTIGRAVITY_EXECUTABLE_DATA_DIR = '$dataDir'

Write-Host "==================================================" -ForegroundColor Cyan
Write-Host " Antigravity Instance: $ProfileName" -ForegroundColor Green
Write-Host " Config Home:          `$env:USERPROFILE"
Write-Host " Workspace:            $Workspace"
Write-Host "==================================================" -ForegroundColor Cyan

Set-Location '$Workspace'
& '$AgyCmd'
"@
}

$script1 = Get-ProfileScriptBlock -ProfileName "acc1" -ProfileDir $dir1 -Workspace $Workspace1
$script2 = Get-ProfileScriptBlock -ProfileName "acc2" -ProfileDir $dir2 -Workspace $Workspace2

$tempScript1 = Join-Path $dir1 "run.ps1"
$tempScript2 = Join-Path $dir2 "run.ps1"
Set-Content -Path $tempScript1 -Value $script1 -Encoding UTF8
Set-Content -Path $tempScript2 -Value $script2 -Encoding UTF8

Write-Host "กำลังเปิด Windows Terminal แยก 2 หน้าต่าง..." -ForegroundColor Yellow

$wt = Get-Command wt.exe -ErrorAction SilentlyContinue
if ($wt) {
    # เปิด 2 Panes ด้วย Windows Terminal
    Start-Process wt.exe -ArgumentList "-w 0 split-pane -d `"$Workspace1`" powershell.exe -NoExit -File `"$tempScript1`" `; split-pane -d `"$Workspace2`" powershell.exe -NoExit -File `"$tempScript2`""
    Write-Host "เปิด 2 Instances ใน Windows Terminal สำเร็จ!" -ForegroundColor Green
} else {
    # Fallback เปิด PowerShell แยก 2 หน้าต่างธรรมดา
    Start-Process powershell.exe -ArgumentList "-NoExit", "-File", "`"$tempScript1`""
    Start-Process powershell.exe -ArgumentList "-NoExit", "-File", "`"$tempScript2`""
    Write-Host "เปิด 2 หน้าต่าง PowerShell สำเร็จ!" -ForegroundColor Green
}
