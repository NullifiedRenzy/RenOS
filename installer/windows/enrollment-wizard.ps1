<#
.SYNOPSIS
    RenOS Enrollment Wizard – Windows Forms GUI for device enrollment.
.DESCRIPTION
    Collects server URL, enrollment token, device name, and device group,
    then POSTs to the RenOS enrollment API. Saves configuration to the
    registry and optionally installs the agent service on success.
#>

#Requires -RunAsAdministrator
#Requires -Version 5.1
#Requires -Assembly System.Windows.Forms
#Requires -Assembly System.Drawing

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

# --- Registry paths ---
$RegPath      = "HKLM:\Software\RenOS"
$ConfigKey    = "HKLM:\Software\RenOS\Config"

# --- API endpoint ---
$EnrollEndpoint = "/api/v1/devices/enroll"

# --- Form dimensions ---
$FormWidth  = 520
$FormHeight = 460

function Test-Admin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Load-SavedConfig {
    $config = @{
        ServerUrl     = ""
        DeviceName    = $env:COMPUTERNAME
        DeviceGroup   = "default"
        EnrollmentToken = ""
    }

    try {
        if (Test-Path $RegPath) {
            $installPath = (Get-ItemProperty -Path $RegPath -Name "InstallPath" -ErrorAction SilentlyContinue).InstallPath
            if ($installPath) { $config.InstallPath = $installPath }
        }
        if (Test-Path $ConfigKey) {
            $props = Get-ItemProperty -Path $ConfigKey -ErrorAction SilentlyContinue
            if ($props.ServerUrl)     { $config.ServerUrl     = $props.ServerUrl }
            if ($props.DeviceName)    { $config.DeviceName    = $props.DeviceName }
            if ($props.DeviceGroup)   { $config.DeviceGroup   = $props.DeviceGroup }
            if ($props.EnrollmentToken) { $config.EnrollmentToken = $props.EnrollmentToken }
        }
    } catch {
        # Silently fall back to defaults
    }
    return $config
}

function Save-Config {
    param($Config)
    try {
        if (-not (Test-Path $RegPath)) {
            New-Item -Path $RegPath -Force -ErrorAction Stop | Out-Null
        }
        if (-not (Test-Path $ConfigKey)) {
            New-Item -Path $ConfigKey -Force -ErrorAction Stop | Out-Null
        }

        Set-ItemProperty -Path $ConfigKey -Name "ServerUrl"       -Value $Config.ServerUrl -ErrorAction Stop
        Set-ItemProperty -Path $ConfigKey -Name "DeviceName"      -Value $Config.DeviceName -ErrorAction Stop
        Set-ItemProperty -Path $ConfigKey -Name "DeviceGroup"     -Value $Config.DeviceGroup -ErrorAction Stop
        Set-ItemProperty -Path $ConfigKey -Name "EnrollmentToken" -Value $Config.EnrollmentToken -ErrorAction Stop
        Set-ItemProperty -Path $RegPath  -Name "Enrolled"         -Value 1 -ErrorAction Stop
        Set-ItemProperty -Path $RegPath  -Name "EnrollmentTime"   -Value (Get-Date -Format "o") -ErrorAction Stop

        return $true
    } catch {
        return $false
    }
}

function Test-Inputs {
    param($ServerUrl, $Token, $DeviceName)
    $errors = @()

    if ([string]::IsNullOrWhiteSpace($ServerUrl)) {
        $errors += "Server URL is required."
    } elseif ($ServerUrl -notmatch '^https?://[a-zA-Z0-9]([a-zA-Z0-9\-\.]+)?(:[0-9]+)?(/.*)?$') {
        $errors += "Server URL is not a valid URL (e.g., https://renos.example.com:8443)."
    }

    if ([string]::IsNullOrWhiteSpace($Token)) {
        $errors += "Enrollment token is required."
    } elseif ($Token -match '[\s]') {
        $errors += "Enrollment token must not contain whitespace."
    }

    if ([string]::IsNullOrWhiteSpace($DeviceName)) {
        $errors += "Device name is required."
    } elseif ($DeviceName.Length -gt 64) {
        $errors += "Device name must be 64 characters or fewer."
    } elseif ($DeviceName -match '[/\\:*?"<>|]') {
        $errors += "Device name contains invalid characters."
    }

    return $errors
}

function Invoke-Enrollment {
    param($ServerUrl, $Token, $DeviceName, $DeviceGroup)

    $url = $ServerUrl.TrimEnd('/') + $EnrollEndpoint
    $body = @{
        device_name  = $DeviceName
        device_group = $DeviceGroup
        token        = $Token
        hostname     = $env:COMPUTERNAME
        os           = "Windows"
        arch         = "x64"
    } | ConvertTo-Json

    try {
        $response = Invoke-RestMethod -Uri $url `
            -Method Post `
            -Body $body `
            -ContentType "application/json" `
            -TimeoutSec 30 `
            -UseBasicParsing `
            -ErrorAction Stop

        return @{
            Success = $true
            Data    = $response
        }
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__

        # Try to read response body for details
        $responseBody = ""
        try {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $responseBody = $reader.ReadToEnd() | Out-String
            $reader.Close()
        } catch {}

        return @{
            Success    = $false
            StatusCode = $statusCode
            Message    = $_.Exception.Message
            Body       = $responseBody
        }
    }
}

function Build-Form {
    $saved = Load-SavedConfig

    # --- Main form ---
    $form = New-Object System.Windows.Forms.Form
    $form.Text             = "RenOS Agent – Device Enrollment"
    $form.Size             = New-Object Drawing.Size($FormWidth, $FormHeight)
    $form.StartPosition    = "CenterScreen"
    $form.FormBorderStyle  = "FixedDialog"
    $form.MaximizeBox      = $false
    $form.MinimizeBox      = $false
    $form.Icon             = [System.Drawing.Icon]::ExtractAssociatedIcon("powershell.exe")

    # --- Banner ---
    $banner = New-Object System.Windows.Forms.Label
    $banner.Text   = "RenOS Agent Enrollment"
    $banner.Font   = New-Object Drawing.Font("Segoe UI", 18, [Drawing.FontStyle]::Bold)
    $banner.Size   = New-Object Drawing.Size($FormWidth - 40, 40)
    $banner.Location = New-Object Drawing.Point(20, 15)
    $banner.ForeColor = [Drawing.Color]::FromArgb(0, 102, 204)
    $form.Controls.Add($banner)

    $subtitle = New-Object System.Windows.Forms.Label
    $subtitle.Text   = "Register this device with your RenOS management server"
    $subtitle.Font   = New-Object Drawing.Font("Segoe UI", 10)
    $subtitle.Size   = New-Object Drawing.Size($FormWidth - 40, 20)
    $subtitle.Location = New-Object Drawing.Point(20, 55)
    $subtitle.ForeColor = [Drawing.Color]::Gray
    $form.Controls.Add($subtitle)

    # --- Server URL ---
    $lblServer = New-Object System.Windows.Forms.Label
    $lblServer.Text     = "Server URL:"
    $lblServer.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $lblServer.Location = New-Object Drawing.Point(20, 95)
    $lblServer.Size     = New-Object Drawing.Size(120, 25)
    $form.Controls.Add($lblServer)

    $txtServer = New-Object System.Windows.Forms.TextBox
    $txtServer.Text     = $saved.ServerUrl
    $txtServer.Location = New-Object Drawing.Point(150, 92)
    $txtServer.Size     = New-Object Drawing.Size(330, 25)
    $txtServer.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $txtServer.ToolTip  = "e.g., https://renos.example.com:8443"
    $form.Controls.Add($txtServer)

    # --- Enrollment Token ---
    $lblToken = New-Object System.Windows.Forms.Label
    $lblToken.Text     = "Enrollment Token:"
    $lblToken.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $lblToken.Location = New-Object Drawing.Point(20, 130)
    $lblToken.Size     = New-Object Drawing.Size(120, 25)
    $form.Controls.Add($lblToken)

    $txtToken = New-Object System.Windows.Forms.TextBox
    $txtToken.Text        = $saved.EnrollmentToken
    $txtToken.Location    = New-Object Drawing.Point(150, 127)
    $txtToken.Size        = New-Object Drawing.Size(330, 25)
    $txtToken.Font        = New-Object Drawing.Font("Segoe UI", 10)
    $txtToken.UseSystemPasswordChar = $true
    $form.Controls.Add($txtToken)

    # Show/Hide token checkbox
    $chkShowToken = New-Object System.Windows.Forms.CheckBox
    $chkShowToken.Text     = "Show token"
    $chkShowToken.Location = New-Object Drawing.Point(490, 129)
    $chkShowToken.Size     = New-Object Drawing.Size(80, 20)
    $chkShowToken.Font     = New-Object Drawing.Font("Segoe UI", 8)
    $chkShowToken.Add_CheckedChanged({
        $txtToken.UseSystemPasswordChar = (-not $chkShowToken.Checked)
    })
    $form.Controls.Add($chkShowToken)

    # --- Device Name ---
    $lblDevice = New-Object System.Windows.Forms.Label
    $lblDevice.Text     = "Device Name:"
    $lblDevice.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $lblDevice.Location = New-Object Drawing.Point(20, 165)
    $lblDevice.Size     = New-Object Drawing.Size(120, 25)
    $form.Controls.Add($lblDevice)

    $txtDevice = New-Object System.Windows.Forms.TextBox
    $txtDevice.Text     = $saved.DeviceName
    $txtDevice.Location = New-Object Drawing.Point(150, 162)
    $txtDevice.Size     = New-Object Drawing.Size(330, 25)
    $txtDevice.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $form.Controls.Add($txtDevice)

    # --- Device Group ---
    $lblGroup = New-Object System.Windows.Forms.Label
    $lblGroup.Text     = "Device Group:"
    $lblGroup.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $lblGroup.Location = New-Object Drawing.Point(20, 200)
    $lblGroup.Size     = New-Object Drawing.Size(120, 25)
    $form.Controls.Add($lblGroup)

    $txtGroup = New-Object System.Windows.Forms.TextBox
    $txtGroup.Text     = $saved.DeviceGroup
    $txtGroup.Location = New-Object Drawing.Point(150, 197)
    $txtGroup.Size     = New-Object Drawing.Size(330, 25)
    $txtGroup.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $form.Controls.Add($txtGroup)

    # --- Status label ---
    $lblStatus = New-Object System.Windows.Forms.Label
    $lblStatus.Text   = ""
    $lblStatus.Font   = New-Object Drawing.Font("Segoe UI", 9)
    $lblStatus.Size   = New-Object Drawing.Size($FormWidth - 40, 40)
    $lblStatus.Location = New-Object Drawing.Point(20, 235)
    $lblStatus.ForeColor = [Drawing.Color]::DarkRed
    $form.Controls.Add($lblStatus)

    # --- Progress bar ---
    $progressBar = New-Object System.Windows.Forms.ProgressBar
    $progressBar.Location = New-Object Drawing.Point(20, 275)
    $progressBar.Size     = New-Object Drawing.Size(460, 20)
    $progressBar.Style    = "Marquee"
    $progressBar.Visible  = $false
    $form.Controls.Add($progressBar)

    # --- Register button ---
    $btnRegister = New-Object System.Windows.Forms.Button
    $btnRegister.Text     = "Register"
    $btnRegister.Font     = New-Object Drawing.Font("Segoe UI", 10, [Drawing.FontStyle]::Bold)
    $btnRegister.Size     = New-Object Drawing.Size(140, 36)
    $btnRegister.Location = New-Object Drawing.Point(20, 310)
    $btnRegister.BackColor = [Drawing.Color]::FromArgb(0, 102, 204)
    $btnRegister.ForeColor = [Drawing.Color]::White
    $btnRegister.FlatStyle = "Flat"
    $btnRegister.Cursor   = "Hand"
    $btnRegister.Add_Click({
        $btnRegister.Enabled      = $false
        $btnSkip.Enabled          = $false
        $progressBar.Visible      = $true
        $lblStatus.Text           = "Enrolling..."
        $lblStatus.ForeColor      = [Drawing.Color]::DarkBlue
        $form.Refresh()

        $serverUrl  = $txtServer.Text.Trim()
        $token      = $txtToken.Text.Trim()
        $deviceName = $txtDevice.Text.Trim()
        $deviceGroup = $txtGroup.Text.Trim()

        $errors = Test-Inputs -ServerUrl $serverUrl -Token $token -DeviceName $deviceName
        if ($errors.Count -gt 0) {
            $lblStatus.Text = $errors -join "`n"
            $lblStatus.ForeColor = [Drawing.Color]::DarkRed
            $progressBar.Visible = $false
            $btnRegister.Enabled = $true
            $btnSkip.Enabled     = $true
            return
        }

        $result = Invoke-Enrollment -ServerUrl $serverUrl -Token $token -DeviceName $deviceName -DeviceGroup $deviceGroup

        if ($result.Success) {
            $config = @{
                ServerUrl       = $serverUrl
                EnrollmentToken = $token
                DeviceName      = $deviceName
                DeviceGroup     = $deviceGroup
            }
            Save-Config -Config $config

            $lblStatus.Text = "Enrollment successful! Device registered."
            $lblStatus.ForeColor = [Drawing.Color]::DarkGreen
            $progressBar.Visible = $false

            # Launch service installation
            $scriptPath = Join-Path (Split-Path -Parent $PSCommandPath) "renos-agent-service.ps1"
            if (Test-Path $scriptPath) {
                Start-Process powershell.exe -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$scriptPath`"" -Verb RunAs
            }

            [System.Windows.Forms.MessageBox]::Show(
                "Device enrolled successfully.`n`nServer: $serverUrl`nDevice: $deviceName`nGroup: $deviceGroup",
                "RenOS Enrollment",
                [System.Windows.Forms.MessageBoxButtons]::OK,
                [System.Windows.Forms.MessageBoxIcon]::Information
            )

            $form.Close()
        } else {
            $statusMsg = if ($result.StatusCode) { "HTTP $($result.StatusCode)" } else { "Connection error" }
            $detail = if ($result.Body) { "`n$($result.Body)" } else { "" }
            $lblStatus.Text = "Enrollment failed: $statusMsg$detail"
            $lblStatus.ForeColor = [Drawing.Color]::DarkRed
            $progressBar.Visible = $false
            $btnRegister.Enabled = $true
            $btnSkip.Enabled     = $true
        }
    })
    $form.Controls.Add($btnRegister)

    # --- Skip button ---
    $btnSkip = New-Object System.Windows.Forms.Button
    $btnSkip.Text     = "Skip"
    $btnSkip.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $btnSkip.Size     = New-Object Drawing.Size(100, 36)
    $btnSkip.Location = New-Object Drawing.Point(180, 310)
    $btnSkip.Cursor   = "Hand"
    $btnSkip.Add_Click({
        $config = @{
            ServerUrl       = $txtServer.Text.Trim()
            EnrollmentToken = $txtToken.Text.Trim()
            DeviceName      = $txtDevice.Text.Trim()
            DeviceGroup     = $txtGroup.Text.Trim()
        }
        Save-Config -Config $config

        $result = [System.Windows.Forms.MessageBox]::Show(
            "Enrollment deferred. You can re-run this wizard from the Start Menu later.`n`nWould you like to install the agent service now?",
            "RenOS Enrollment",
            [System.Windows.Forms.MessageBoxButtons]::YesNo,
            [System.Windows.Forms.MessageBoxIcon]::Question
        )

        if ($result -eq "Yes") {
            $scriptPath = Join-Path (Split-Path -Parent $PSCommandPath) "renos-agent-service.ps1"
            if (Test-Path $scriptPath) {
                Start-Process powershell.exe -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$scriptPath`"" -Verb RunAs
            }
        }

        $form.Close()
    })
    $form.Controls.Add($btnSkip)

    # --- Save & Close button ---
    $btnSaveClose = New-Object System.Windows.Forms.Button
    $btnSaveClose.Text     = "Save && Close"
    $btnSaveClose.Font     = New-Object Drawing.Font("Segoe UI", 10)
    $btnSaveClose.Size     = New-Object Drawing.Size(120, 36)
    $btnSaveClose.Location = New-Object Drawing.Point(300, 310)
    $btnSaveClose.Cursor   = "Hand"
    $btnSaveClose.Add_Click({
        $config = @{
            ServerUrl       = $txtServer.Text.Trim()
            EnrollmentToken = $txtToken.Text.Trim()
            DeviceName      = $txtDevice.Text.Trim()
            DeviceGroup     = $txtGroup.Text.Trim()
        }
        Save-Config -Config $config
        $form.Close()
    })
    $form.Controls.Add($btnSaveClose)

    # --- Footer ---
    $footer = New-Object System.Windows.Forms.Label
    $footer.Text   = "RenOS Agent v1.0.0  |  Configuration saved to HKLM\Software\RenOS"
    $footer.Font   = New-Object Drawing.Font("Segoe UI", 8)
    $footer.Size   = New-Object Drawing.Size($FormWidth - 40, 20)
    $footer.Location = New-Object Drawing.Point(20, 370)
    $footer.ForeColor = [Drawing.Color]::Gray
    $form.Controls.Add($footer)

    return $form
}

# --- Main ---
if (-not (Test-Admin)) {
    $result = [System.Windows.Forms.MessageBox]::Show(
        "RenOS Enrollment Wizard requires Administrator privileges. Restart with elevation?",
        "RenOS Enrollment",
        [System.Windows.Forms.MessageBoxButtons]::YesNo,
        [System.Windows.Forms.MessageBoxIcon]::Warning
    )
    if ($result -eq "Yes") {
        Start-Process powershell.exe -Verb RunAs -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`""
    }
    exit
}

$form = Build-Form
[void]$form.ShowDialog()
