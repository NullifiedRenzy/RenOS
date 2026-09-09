#Requires -Version 5.1
$ErrorActionPreference = 'Stop'
. "$PSScriptRoot\agent-common.ps1"
$config = Read-AgentConfig "$PSScriptRoot\config.json"
$env:RENOS_DATA_DIR = $config.data_dir
$env:RENOS_API_URL = $config.core_url
$env:RUST_LOG = $config.log_level
if ($config.nats_url) { $env:RENOS_NATS_URL = $config.nats_url }
if (-not (Test-Path -LiteralPath (Join-Path $config.data_dir 'device-secret.txt'))) { exit 0 }
$logDir = Join-Path $config.data_dir 'Logs'
New-Item -ItemType Directory -Path $logDir -Force | Out-Null
$log = Join-Path $logDir 'agent.log'
if ((Test-Path -LiteralPath $log) -and (Get-Item -LiteralPath $log).Length -gt 10MB) {
    Move-Item -LiteralPath $log -Destination "$log.previous" -Force
}
& "$PSScriptRoot\renos-agent.exe" run --core $config.core_url *>> $log
if ($LASTEXITCODE -eq 0) { exit 1 }
exit $LASTEXITCODE
