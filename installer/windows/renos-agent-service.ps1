#Requires -RunAsAdministrator
#Requires -Version 5.1
[CmdletBinding()]
param([string]$TaskName = 'RenOS-Agent', [switch]$Uninstall)
$ErrorActionPreference = 'Stop'
if ($Uninstall) {
    if (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue) {
        Stop-ScheduledTask -TaskName $TaskName
        Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    }
    Write-Host 'Startup task removed. Configuration and enrollment data were retained.'
    return
}
. "$PSScriptRoot\agent-common.ps1"
$config = Read-AgentConfig "$PSScriptRoot\config.json"
if ($TaskName -eq 'RenOS-Agent') {
    $oldUpdater = Get-ScheduledTask -TaskName 'RenOS-Agent-AutoUpdate' -ErrorAction SilentlyContinue
    if ($oldUpdater -and ($oldUpdater.Actions.Arguments -join ' ') -match 'update\.renos\.local') {
        Stop-ScheduledTask -TaskName 'RenOS-Agent-AutoUpdate'
        Unregister-ScheduledTask -TaskName 'RenOS-Agent-AutoUpdate' -Confirm:$false
    }
}
# Older releases registered a console executable as an unsupported SCM service.
$legacy = Get-CimInstance Win32_Service -Filter "Name='RenOS-Agent'"
if ($legacy -and $TaskName -eq 'RenOS-Agent') {
    $expected = [regex]::Escape((Join-Path $PSScriptRoot 'renos-agent.exe'))
    if ($legacy.PathName -notmatch ('^"?' + $expected + '("|\s|$)')) {
        throw 'An unrelated RenOS-Agent service exists; it was not modified.'
    }
    Stop-Service -Name 'RenOS-Agent' -Force -ErrorAction Stop
    & sc.exe delete RenOS-Agent | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not remove the legacy agent service.' }
}
$shell = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
$action = New-ScheduledTaskAction -Execute $shell -WorkingDirectory $PSScriptRoot `
    -Argument ('-NoProfile -NonInteractive -WindowStyle Hidden -ExecutionPolicy Bypass -File "' + (Join-Path $PSScriptRoot 'run-agent.ps1') + '"')
$trigger = New-ScheduledTaskTrigger -AtStartup
$principal = New-ScheduledTaskPrincipal -UserId 'S-1-5-18' -LogonType ServiceAccount -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
    -StartWhenAvailable -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew `
    -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger `
    -Principal $principal -Settings $settings -Force | Out-Null
if (Test-Path -LiteralPath (Join-Path $config.data_dir 'device-secret.txt')) { Start-ScheduledTask -TaskName $TaskName }
