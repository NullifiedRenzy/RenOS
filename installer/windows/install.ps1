#Requires -RunAsAdministrator
#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$InstallDir = "$env:ProgramFiles\RenOS\Agent",
    [string]$CoreUrl = "http://192.168.0.117:18081",
    [string]$DataDir = "$env:ProgramData\RenOS",
    [string]$TaskName = "RenOS-Agent",
    [switch]$NoWizard
)
$ErrorActionPreference = "Stop"
. "$PSScriptRoot\agent-common.ps1"
Assert-CoreUrl $CoreUrl
$files = @('renos-agent.exe', 'agent-common.ps1', 'run-agent.ps1', 'renos-agent-service.ps1', 'enrollment-wizard.ps1')
foreach ($file in $files) {
    if (-not (Test-Path -LiteralPath (Join-Path $PSScriptRoot $file) -PathType Leaf)) { throw "Missing release file: $file" }
}
if ([IO.Path]::GetFullPath($InstallDir).TrimEnd('\') -eq [IO.Path]::GetFullPath($PSScriptRoot).TrimEnd('\')) {
    throw 'Extract the release outside the installation directory before upgrading.'
}
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Protect-AgentDirectory $InstallDir
$configPath = Join-Path $InstallDir 'config.json'
if (Test-Path -LiteralPath $configPath) {
    $config = Read-AgentConfig $configPath
} else {
    $config = [pscustomobject]@{ core_url = $CoreUrl; data_dir = $DataDir; log_level = 'info' }
}
New-Item -ItemType Directory -Path $config.data_dir -Force | Out-Null
Protect-AgentDirectory $config.data_dir
$oldTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($oldTask) {
    Stop-ScheduledTask -TaskName $TaskName
    $deadline = (Get-Date).AddSeconds(30)
    while ((Get-ScheduledTask -TaskName $TaskName).State -eq 'Running') {
        if ((Get-Date) -gt $deadline) { throw 'The agent did not stop; no files were replaced.' }
        Start-Sleep -Milliseconds 250
    }
}
$backup = Join-Path $InstallDir ('.upgrade-' + [guid]::NewGuid().ToString('N'))
$installed = $false
New-Item -ItemType Directory -Path $backup | Out-Null
try {
    foreach ($file in $files) {
        $target = Join-Path $InstallDir $file
        if (Test-Path -LiteralPath $target) { Copy-Item -LiteralPath $target -Destination $backup }
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot $file) -Destination $target -Force
    }
    if (-not (Test-Path -LiteralPath $configPath)) {
        $config | ConvertTo-Json | Set-Content -LiteralPath $configPath -Encoding UTF8
    }
    & "$InstallDir\renos-agent-service.ps1" -TaskName $TaskName
    $installed = $true
} catch {
    foreach ($file in $files) {
        $previous = Join-Path $backup $file
        if (Test-Path -LiteralPath $previous) { Copy-Item -LiteralPath $previous -Destination (Join-Path $InstallDir $file) -Force }
    }
    if ($oldTask) { Start-ScheduledTask -TaskName $TaskName }
    throw
} finally {
    $resolvedBackup = [IO.Path]::GetFullPath($backup)
    $installPrefix = [IO.Path]::GetFullPath($InstallDir).TrimEnd('\') + '\'
    if ($installed -and $resolvedBackup.StartsWith($installPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        Remove-Item -LiteralPath $resolvedBackup -Recurse -Force
    }
}
Write-Host 'RenOS Agent installed. Existing configuration and identity were preserved.'
if (-not $NoWizard -and -not (Test-Path -LiteralPath (Join-Path $config.data_dir 'device-secret.txt'))) {
    & "$InstallDir\enrollment-wizard.ps1" -TaskName $TaskName
}
