param(
    [string]$InstallDir = "$env:ProgramFiles\RenOS\Agent",
    [string]$CoreUrl = "http://192.168.0.117"
)

$ErrorActionPreference = "Stop"

if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "Run this installer from an elevated PowerShell session."
}

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item -Path "$PSScriptRoot\renos-agent.exe" -Destination "$InstallDir\renos-agent.exe" -Force
Copy-Item -Path "$PSScriptRoot\renos-agent-service.ps1" -Destination "$InstallDir\renos-agent-service.ps1" -Force
Copy-Item -Path "$PSScriptRoot\enrollment-wizard.ps1" -Destination "$InstallDir\enrollment-wizard.ps1" -Force

$config = @{
    core_url = $CoreUrl
    data_dir = "C:\ProgramData\RenOS"
    log_level = "info"
} | ConvertTo-Json -Depth 3
$config | Set-Content -Path "$InstallDir\config.json" -Encoding UTF8

& "$InstallDir\renos-agent-service.ps1"
Write-Host "RenOS Agent installed. Use the enrollment wizard to connect this Windows device." -ForegroundColor Green
