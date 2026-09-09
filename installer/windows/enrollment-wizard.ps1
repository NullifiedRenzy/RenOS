#Requires -RunAsAdministrator
#Requires -Version 5.1
[CmdletBinding()]
param([string]$TaskName = 'RenOS-Agent')
$ErrorActionPreference = 'Stop'
. "$PSScriptRoot\agent-common.ps1"
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$config = Read-AgentConfig "$PSScriptRoot\config.json"
$form = New-Object Windows.Forms.Form
$form.Text = 'RenOS Device Enrollment'
$form.ClientSize = New-Object Drawing.Size(500, 245)
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedDialog'
$form.MaximizeBox = $false
$form.AutoScaleMode = 'Dpi'
$form.Font = New-Object Drawing.Font('Segoe UI', 10)
foreach ($spec in @(@('Server URL', 25), @('Enrollment token', 85))) {
    $label = New-Object Windows.Forms.Label
    $label.Text = $spec[0]
    $label.SetBounds(20, $spec[1], 145, 25)
    $form.Controls.Add($label)
}
$server = New-Object Windows.Forms.TextBox
$server.Text = $config.core_url
$server.SetBounds(170, 22, 310, 28)
$form.Controls.Add($server)
$token = New-Object Windows.Forms.TextBox
$token.UseSystemPasswordChar = $true
$token.SetBounds(170, 82, 310, 28)
$form.Controls.Add($token)
$status = New-Object Windows.Forms.Label
$status.SetBounds(20, 125, 460, 55)
$form.Controls.Add($status)
$enroll = New-Object Windows.Forms.Button
$enroll.Text = 'Enroll device'
$enroll.SetBounds(340, 190, 140, 35)
$form.Controls.Add($enroll)
$form.AcceptButton = $enroll
$enroll.Add_Click({
    $enroll.Enabled = $false
    try {
        Assert-CoreUrl $server.Text.Trim()
        if ([string]::IsNullOrWhiteSpace($token.Text)) { throw 'Enter an enrollment token.' }
        $status.Text = 'Enrolling...'
        $form.Refresh()
        $env:RENOS_DATA_DIR = $config.data_dir
        & "$PSScriptRoot\renos-agent.exe" enroll --core $server.Text.Trim() --token $token.Text.Trim() | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'Enrollment failed. Check the server URL and token.' }
        if (-not (Test-Path -LiteralPath (Join-Path $config.data_dir 'device-secret.txt'))) {
            throw 'Enrollment did not create the device identity.'
        }
        $config.core_url = $server.Text.Trim()
        $config | ConvertTo-Json | Set-Content -LiteralPath "$PSScriptRoot\config.json" -Encoding UTF8
        $token.Clear()
        & "$PSScriptRoot\renos-agent-service.ps1" -TaskName $TaskName
        $form.Close()
    } catch { $status.Text = $_.Exception.Message }
    finally { $enroll.Enabled = $true }
})
[void]$form.ShowDialog()
$form.Dispose()
