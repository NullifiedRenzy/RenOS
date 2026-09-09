# Windows Agent

Extract `RenOS-Windows-x64.zip` and run `install.ps1` in an administrator
PowerShell window. Windows 10/11 x64 with PowerShell 5.1 or later is required.

```powershell
.\install.ps1 -CoreUrl 'http://192.168.0.117:18081'
```

The enrollment dialog uses a one-time token from the Hub. It invokes the real
agent CLI and stores its identity under `%ProgramData%\RenOS`. Tokens are not
saved in the registry. The `RenOS-Agent` scheduled task runs as SYSTEM at startup
without a user login; failed processes retry after one minute. The console agent
is not a native Windows SCM service. No inbound firewall rules, forced reboot,
or unverified automatic updater are installed.

## Upgrade and removal

Extract each release into a separate folder and rerun `install.ps1`. Configuration
and enrollment are retained. An existing `config.json` takes precedence over
`-CoreUrl`. The task stops before replacement; old files are restored on failure.

```powershell
Get-ScheduledTask -TaskName RenOS-Agent
Start-ScheduledTask -TaskName RenOS-Agent
Stop-ScheduledTask -TaskName RenOS-Agent
& "$env:ProgramFiles\RenOS\Agent\enrollment-wizard.ps1"
& "$env:ProgramFiles\RenOS\Agent\renos-agent-service.ps1" -Uninstall
```

Uninstall removes startup registration, not identity or configuration. Logs are
under `%ProgramData%\RenOS\Logs`. NATS-based commands also require a reachable,
secured `nats_url` in config.json; enrollment/heartbeats alone do not provide it.

The experimental WiX files are not used by the published ZIP installer. These
scripts and binaries are not Authenticode-signed. Trusted publisher verification
requires a signing certificate that has not yet been configured.
