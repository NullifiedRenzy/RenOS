<#
.SYNOPSIS
    Installs and configures the RenOS Agent as a Windows service.
.DESCRIPTION
    Creates the Windows service, sets recovery options, configures
    event logging, creates log directories, registers a scheduled task
    for auto-updates, and adds Windows Firewall rules.
.PARAMETER ServiceName
    Name of the Windows service. Defaults to "RenOS-Agent".
.PARAMETER DisplayName
    Display name of the service. Defaults to "RenOS Agent".
.PARAMETER BinaryPath
    Path to renos-agent.exe. Defaults to "$PSScriptRoot\renos-agent.exe".
.PARAMETER ConfigPath
    Path to config.json. Defaults to "$PSScriptRoot\config.json".
.PARAMETER LogDirectory
    Directory for agent logs. Defaults to "C:\ProgramData\RenOS\Logs".
.PARAMETER UpdateServer
    URL of the update server for auto-update checks. Defaults to "https://update.renos.local/v1".
.PARAMETER Uninstall
    Switch to remove the service and all related configuration.
#>

[CmdletBinding()]
param(
    [string]$ServiceName = "RenOS-Agent",
    [string]$DisplayName = "RenOS Agent",
    [string]$BinaryPath,
    [string]$ConfigPath,
    [string]$LogDirectory = "C:\ProgramData\RenOS\Logs",
    [string]$UpdateServer = "https://update.renos.local/v1",
    [switch]$Uninstall
)

#Requires -RunAsAdministrator
#Requires -Version 5.1

$ScriptRoot = Split-Path -Parent $PSCommandPath
if (-not $BinaryPath) { $BinaryPath = Join-Path $ScriptRoot "renos-agent.exe" }
if (-not $ConfigPath) { $ConfigPath  = Join-Path $ScriptRoot "config.json" }

$ServiceDescription = "RenOS device management and enrollment agent"
$EventLogSource     = "RenOS-Agent"
$ScheduledTaskName  = "RenOS-Agent-AutoUpdate"
$FirewallRuleName   = "RenOS Agent"

function Write-Log {
    param([string]$Message, [ConsoleColor]$Color = "Gray")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] $Message" -ForegroundColor $Color
}

function Test-Admin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Install-Service {
    Write-Log "Installing service: $ServiceName" -Color Cyan

    if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
        Write-Log "Service '$ServiceName' already exists. Stopping and removing..." -Color Yellow
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        & sc.exe delete $ServiceName 2>&1 | Out-Null
        Start-Sleep -Seconds 2
    }

    $binaryArgs = "-config `"$ConfigPath`" --service"
    $fullPath = "`"$BinaryPath`" $binaryArgs"

    try {
        New-Service -Name $ServiceName `
                    -DisplayName $DisplayName `
                    -Description $ServiceDescription `
                    -BinaryPathName $fullPath `
                    -StartupType Automatic `
                    -Credential $null `
                    -ErrorAction Stop | Out-Null
        Write-Log "Service created successfully." -Color Green
    } catch {
        Write-Log "Failed to create service: $_" -Color Red
        return $false
    }

    # Set recovery options
    Write-Log "Setting service recovery options..." -Color Gray
    try {
        & sc.exe failure $ServiceName reset=86400 actions=restart/30000/restart/60000/reboot/120000 2>&1 | Out-Null
        & sc.exe failureflag $ServiceName 1 2>&1 | Out-Null
        & sc.exe privs $ServiceName SeTcbPrivilege 2>&1 | Out-Null
        Write-Log "Recovery options configured: restart (30s), restart (60s), reboot (120s)." -Color Green
    } catch {
        Write-Log "Failed to set recovery options: $_" -Color Yellow
    }

    # Set service description
    try {
        & sc.exe description $ServiceName "$ServiceDescription" 2>&1 | Out-Null
    } catch {
        Write-Log "Failed to set service description." -Color Yellow
    }

    return $true
}

function Install-EventLog {
    Write-Log "Configuring EventLog source: $EventLogSource" -Color Gray
    try {
        if (-not [System.Diagnostics.EventLog]::SourceExists($EventLogSource)) {
            [System.Diagnostics.EventLog]::CreateEventSource($EventLogSource, "Application")
            Write-Log "EventLog source '$EventLogSource' created." -Color Green
        } else {
            Write-Log "EventLog source '$EventLogSource' already exists." -Color Gray
        }
    } catch {
        Write-Log "Failed to create EventLog source: $_" -Color Yellow
    }
}

function Install-LogDirectory {
    Write-Log "Creating log directory: $LogDirectory" -Color Gray
    try {
        if (-not (Test-Path $LogDirectory)) {
            New-Item -ItemType Directory -Path $LogDirectory -Force -ErrorAction Stop | Out-Null
            Write-Log "Log directory created." -Color Green
        } else {
            Write-Log "Log directory already exists." -Color Gray
        }

        # Set ACL: allow LocalSystem and Administrators full control
        $acl = Get-Acl -Path $LogDirectory -ErrorAction Stop
        $system = New-Object System.Security.Principal.NTAccount("NT AUTHORITY\SYSTEM")
        $admins = New-Object System.Security.Principal.NTAccount("BUILTIN\Administrators")
        $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
            $system, "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow"
        )
        $accessRuleAdmins = New-Object System.Security.AccessControl.FileSystemAccessRule(
            $admins, "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow"
        )
        $acl.SetAccessRule($accessRule)
        $acl.SetAccessRule($accessRuleAdmins)
        Set-Acl -Path $LogDirectory -AclObject $acl -ErrorAction Stop
        Write-Log "Log directory ACL configured." -Color Green
    } catch {
        Write-Log "Failed to configure log directory: $_" -Color Yellow
    }
}

function Install-ScheduledTask {
    Write-Log "Registering scheduled task: $ScheduledTaskName" -Color Gray

    $updateScript = @"
`$updateServer = "$UpdateServer"
`$currentVersion = "1.0.0"
`$binaryPath = "$BinaryPath"

try {
    `$response = Invoke-RestMethod -Uri "`$updateServer/check?version=`$currentVersion" -Method Get -TimeoutSec 30 -UseBasicParsing
    if (`$response.update_available -eq `$true) {
        `$tempDir = Join-Path `$env:TEMP "RenOS-Update"
        New-Item -ItemType Directory -Path `$tempDir -Force | Out-Null
        `$msiPath = Join-Path `$tempDir "RenOS-Agent-x64.msi"
        Invoke-WebRequest -Uri "`$updateServer/download/latest" -OutFile `$msiPath -UseBasicParsing
        Start-Process -FilePath msiexec.exe -ArgumentList "/i `"`$msiPath`" /quiet /norestart" -Wait
    }
} catch {
    [System.Diagnostics.EventLog]::WriteEntry("RenOS-Agent", "Update check failed: $_", "Warning", 1001)
}
"@

    try {
        $action = New-ScheduledTaskAction -Execute "PowerShell.exe" `
            -Argument "-NoProfile -ExecutionPolicy Bypass -Command `"$updateScript`"" `
            -ErrorAction Stop

        $trigger = New-ScheduledTaskTrigger -Daily -At 3am -ErrorAction Stop

        $principal = New-ScheduledTaskPrincipal -UserId "NT AUTHORITY\SYSTEM" `
            -LogonType ServiceAccount `
            -RunLevel Highest `
            -ErrorAction Stop

        $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries `
            -DontStopIfGoingOnBatteries `
            -StartWhenAvailable `
            -ExecutionTimeLimit (New-TimeSpan -Hours 1) `
            -ErrorAction Stop

        Register-ScheduledTask -TaskName $ScheduledTaskName `
            -Action $action `
            -Trigger $trigger `
            -Principal $principal `
            -Settings $settings `
            -Force `
            -ErrorAction Stop | Out-Null

        Write-Log "Scheduled task '$ScheduledTaskName' registered (daily at 03:00)." -Color Green
    } catch {
        Write-Log "Failed to register scheduled task: $_" -Color Yellow
    }
}

function Install-FirewallRules {
    Write-Log "Adding Windows Firewall rules..." -Color Gray

    try {
        $existing = Get-NetFirewallRule -DisplayName "$FirewallRuleName (TCP-In)" -ErrorAction SilentlyContinue
        if (-not $existing) {
            New-NetFirewallRule -DisplayName "$FirewallRuleName (TCP-In)" `
                -Direction Inbound `
                -Protocol TCP `
                -LocalPort 80,443,8443 `
                -Action Allow `
                -Program "$BinaryPath" `
                -Profile Any `
                -Description "Allow RenOS Agent inbound traffic on ports 80, 443, 8443" `
                -ErrorAction Stop | Out-Null
            Write-Log "Inbound firewall rule created for ports 80, 443, 8443." -Color Green
        } else {
            Write-Log "Inbound firewall rule already exists." -Color Gray
        }

        $existingOut = Get-NetFirewallRule -DisplayName "$FirewallRuleName (TCP-Out)" -ErrorAction SilentlyContinue
        if (-not $existingOut) {
            New-NetFirewallRule -DisplayName "$FirewallRuleName (TCP-Out)" `
                -Direction Outbound `
                -Protocol TCP `
                -Action Allow `
                -Program "$BinaryPath" `
                -Profile Any `
                -Description "Allow RenOS Agent outbound traffic" `
                -ErrorAction Stop | Out-Null
            Write-Log "Outbound firewall rule created." -Color Green
        } else {
            Write-Log "Outbound firewall rule already exists." -Color Gray
        }
    } catch {
        Write-Log "Failed to add firewall rules: $_" -Color Yellow
    }
}

function Uninstall-Service {
    Write-Log "Removing RenOS Agent service configuration..." -Color Cyan

    try {
        if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            & sc.exe delete $ServiceName 2>&1 | Out-Null
            Write-Log "Service '$ServiceName' removed." -Color Green
        }
    } catch {
        Write-Log "Failed to remove service: $_" -Color Yellow
    }

    try {
        $task = Get-ScheduledTask -TaskName $ScheduledTaskName -ErrorAction SilentlyContinue
        if ($task) {
            Unregister-ScheduledTask -TaskName $ScheduledTaskName -Confirm:$false -ErrorAction Stop
            Write-Log "Scheduled task '$ScheduledTaskName' removed." -Color Green
        }
    } catch {
        Write-Log "Failed to remove scheduled task: $_" -Color Yellow
    }

    try {
        Get-NetFirewallRule -DisplayName "$FirewallRuleName (TCP-In)" -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction Stop
        Get-NetFirewallRule -DisplayName "$FirewallRuleName (TCP-Out)" -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction Stop
        Write-Log "Firewall rules removed." -Color Green
    } catch {
        Write-Log "Failed to remove firewall rules: $_" -Color Yellow
    }

    Write-Log "Uninstall complete." -Color Green
}

# --- Main ---
if (-not (Test-Admin)) {
    Write-Host "ERROR: This script requires Administrator privileges. Restarting with elevation..." -ForegroundColor Red
    $args = "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`" $($MyInvocation.BoundParameters | ConvertTo-Json -Compress)"
    Start-Process powershell.exe -Verb RunAs -ArgumentList $args
    exit
}

if ($Uninstall) {
    Uninstall-Service
    exit 0
}

Write-Log "=== RenOS Agent Service Installation ===" -Color Cyan
Write-Log "Service Name : $ServiceName"
Write-Log "Binary Path  : $BinaryPath"
Write-Log "Config Path  : $ConfigPath"
Write-Log "Log Dir      : $LogDirectory"
Write-Log ""

Install-LogDirectory
Install-EventLog
$serviceResult = Install-Service
if ($serviceResult) {
    Install-ScheduledTask
    Install-FirewallRules
    Write-Log ""
    Write-Log "=== Installation Complete ===" -Color Cyan
    Write-Log "Start the service with: Start-Service -Name $ServiceName"
} else {
    Write-Log "Service installation failed. See errors above." -Color Red
    exit 1
}
