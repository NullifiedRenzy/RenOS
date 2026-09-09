#Requires -RunAsAdministrator
#Requires -Version 5.1
$ErrorActionPreference = 'Stop'
$root = Join-Path $env:TEMP ('renos-installer-test-' + [guid]::NewGuid().ToString('N'))
$bundle = Join-Path $root 'release'
$installed = Join-Path $root 'installed'
$data = Join-Path $root 'data'
$taskName = 'RenOS-Test-' + [guid]::NewGuid().ToString('N')
New-Item -ItemType Directory -Path $bundle, $data -Force | Out-Null
try {
    Copy-Item -Path "$PSScriptRoot\*.ps1" -Destination $bundle
    Add-Type -TypeDefinition @'
using System;
using System.IO;
public class TestAgent {
    public static int Main(string[] args) {
        File.WriteAllText(Path.Combine(Environment.GetEnvironmentVariable("RENOS_DATA_DIR"), "invocation.txt"), string.Join("|", args));
        return 1;
    }
}
'@ -OutputAssembly "$bundle\renos-agent.exe" -OutputType ConsoleApplication
    & "$bundle\install.ps1" -InstallDir $installed -DataDir $data -CoreUrl 'http://127.0.0.1:18081' -TaskName $taskName -NoWizard
    $originalConfig = Get-Content -LiteralPath "$installed\config.json" -Raw
    'fixture-identity' | Set-Content -LiteralPath "$data\identity.json"
    'fixture-device-secret' | Set-Content -LiteralPath "$data\device-secret.txt"
    & "$bundle\install.ps1" -InstallDir $installed -TaskName $taskName -NoWizard
    if ((Get-Content -LiteralPath "$installed\config.json" -Raw) -ne $originalConfig) { throw 'Upgrade replaced existing configuration.' }
    if ((Get-Content -LiteralPath "$data\identity.json") -ne 'fixture-identity') { throw 'Upgrade replaced device identity.' }
    $deadline = (Get-Date).AddSeconds(40)
    while (-not (Test-Path -LiteralPath "$data\invocation.txt")) {
        if ((Get-Date) -gt $deadline) { throw 'Scheduled task did not start the agent.' }
        Start-Sleep -Milliseconds 500
    }
    if ((Get-Content -LiteralPath "$data\invocation.txt" -Raw) -ne 'run|--core|http://127.0.0.1:18081') {
        throw 'Startup did not use the supported CLI arguments.'
    }
    $task = Get-ScheduledTask -TaskName $taskName
    if ($task.Settings.RestartCount -ne 999) { throw 'Missing task restart policy.' }
    if ($task.Settings.ExecutionTimeLimit -ne 'PT0S') { throw 'Agent task has a runtime limit.' }
    & "$installed\renos-agent-service.ps1" -TaskName $taskName -Uninstall
    if (-not (Test-Path -LiteralPath "$data\device-secret.txt")) { throw 'Uninstall removed device identity.' }
    Write-Host 'PASS: install, startup, upgrade, supported CLI, and identity preservation'
} finally {
    if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
        Stop-ScheduledTask -TaskName $taskName
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    }
    $resolved = [IO.Path]::GetFullPath($root)
    $tempPrefix = [IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\'
    if (-not $resolved.StartsWith($tempPrefix, [StringComparison]::OrdinalIgnoreCase)) { throw 'Invalid test cleanup path.' }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}
