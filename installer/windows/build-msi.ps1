<#
.SYNOPSIS
    Builds the RenOS Agent MSI installer using WiX Toolset v4.
.DESCRIPTION
    Checks for WiX Toolset v4 (installs if missing), compiles renos.wxs,
    optionally signs with a code signing certificate, and outputs to dist/.
.PARAMETER ConfigPath
    Path to the WiX .wxs file. Defaults to renos.wxs in the script directory.
.PARAMETER OutputDir
    Directory for the built MSI. Defaults to ./dist relative to the script directory.
.PARAMETER Version
    Product version string. Defaults to "1.0.0".
.PARAMETER SourceDir
    Directory containing the files to be packaged. Defaults to the script directory.
#>

[CmdletBinding()]
param(
    [string]$ConfigPath,
    [string]$OutputDir,
    [string]$Version = "1.0.0",
    [string]$SourceDir
)

$ScriptRoot = Split-Path -Parent $PSCommandPath
if (-not $ConfigPath) { $ConfigPath = Join-Path $ScriptRoot "renos.wxs" }
if (-not $OutputDir)  { $OutputDir  = Join-Path $ScriptRoot "dist" }
if (-not $SourceDir)  { $SourceDir  = $ScriptRoot }

$WixRoot = "$env:ProgramFiles\WiX Toolset v4"
$Candle  = "$WixRoot\bin\candle.exe"
$Light   = "$WixRoot\bin\light.exe"

function Test-WixInstalled {
    return (Test-Path $Candle) -and (Test-Path $Light)
}

function Install-WiX {
    Write-Host "WiX Toolset v4 not found. Downloading..." -ForegroundColor Yellow
    $installer = "$env:TEMP\wix-installer.exe"
    $url = "https://github.com/wixtoolset/wix4/releases/latest/download/wix.exe"
    try {
        Invoke-WebRequest -Uri $url -OutFile $installer -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Host "Failed to download WiX from $url : $_" -ForegroundColor Red
        Write-Host "Please install WiX Toolset v4 manually from https://wixtoolset.org/" -ForegroundColor Yellow
        exit 1
    }

    Write-Host "Running WiX installer..." -ForegroundColor Yellow
    $proc = Start-Process -FilePath $installer -ArgumentList "/install", "/quiet", "/norestart" -Wait -PassThru
    if ($proc.ExitCode -ne 0 -and $proc.ExitCode -ne 3010) {
        Write-Host "WiX installation failed (exit code: $($proc.ExitCode))." -ForegroundColor Red
        exit 1
    }

    if (-not (Test-WixInstalled)) {
        Write-Host "WiX installed but binaries not found at expected path. Trying fallback..." -ForegroundColor Yellow
        $altPaths = @(
            "${env:ProgramFiles(x86)}\WiX Toolset v4\bin",
            "$env:ProgramFiles\WiX Toolset v4\bin",
            "$env:LOCALAPPDATA\wixtoolset\v4\bin"
        )
        foreach ($p in $altPaths) {
            $c = Join-Path $p "candle.exe"
            $l = Join-Path $p "light.exe"
            if ((Test-Path $c) -and (Test-Path $l)) {
                $script:Candle = $c
                $script:Light  = $l
                $script:WixRoot = $p
                Write-Host "Found WiX at: $p" -ForegroundColor Green
                return $true
            }
        }
        Write-Host "Cannot locate WiX binaries after installation. Please install manually." -ForegroundColor Red
        exit 1
    }
    Write-Host "WiX Toolset v4 installed successfully." -ForegroundColor Green
}

function Invoke-Sign {
    param([string]$Path)
    $cert = [Environment]::GetEnvironmentVariable("CODE_SIGN_CERT", "User")
    if (-not $cert) {
        $cert = [Environment]::GetEnvironmentVariable("CODE_SIGN_CERT", "Machine")
    }
    if (-not $cert) {
        $cert = $env:CODE_SIGN_CERT
    }
    if (-not $cert) {
        Write-Host "CODE_SIGN_CERT not set. Skipping code signing." -ForegroundColor Yellow
        return
    }

    try {
        $certObj = Get-Item -Path "Cert:\CurrentUser\My\$cert" -ErrorAction Stop
    } catch {
        try {
            $certObj = Get-Item -Path "Cert:\LocalMachine\My\$cert" -ErrorAction Stop
        } catch {
            Write-Host "Certificate with thumbprint '$cert' not found. Skipping signing." -ForegroundColor Yellow
            return
        }
    }

    $signtool = Get-Command "signtool.exe" -ErrorAction SilentlyContinue
    if (-not $signtool) {
        $signtool = Get-Command "SignTool" -ErrorAction SilentlyContinue
    }
    if (-not $signtool) {
        Write-Host "SignTool not found (install Windows SDK). Skipping signing." -ForegroundColor Yellow
        return
    }

    Write-Host "Signing: $Path" -ForegroundColor Green
    & $signtool.Source sign /fd SHA256 /sha1 $cert /tr "http://timestamp.digicert.com" /td SHA256 "$Path"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Code signing failed (exit code: $LASTEXITCODE)." -ForegroundColor Red
    } else {
        Write-Host "Signed successfully." -ForegroundColor Green
    }
}

function Build-Msi {
    Write-Host "=== RenOS Agent MSI Build ===" -ForegroundColor Cyan
    Write-Host "Source directory : $SourceDir"
    Write-Host "Config path      : $ConfigPath"
    Write-Host "Output directory : $OutputDir"
    Write-Host "Version          : $Version"
    Write-Host ""

    if (-not (Test-WixInstalled)) {
        Install-WiX
    } else {
        Write-Host "WiX Toolset v4 found at $WixRoot" -ForegroundColor Green
    }

    if (-not (Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
        Write-Host "Created output directory: $OutputDir" -ForegroundColor Gray
    }

    $wixobj = Join-Path $OutputDir "renos.wixobj"
    Write-Host "Compiling with candle.exe..." -ForegroundColor Yellow
    & $Candle -arch x64 -dVersion="$Version" -dSourceDir="$SourceDir" -out "$wixobj" "$ConfigPath"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Candle compilation failed (exit code: $LASTEXITCODE)." -ForegroundColor Red
        exit 1
    }
    Write-Host "Compilation succeeded." -ForegroundColor Green

    $msiPath = Join-Path $OutputDir "RenOS-Agent-x64.msi"
    Write-Host "Linking with light.exe..." -ForegroundColor Yellow
    & $Light -out "$msiPath" "$wixobj"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Light linking failed (exit code: $LASTEXITCODE)." -ForegroundColor Red
        exit 1
    }
    Write-Host "Linking succeeded." -ForegroundColor Green

    Invoke-Sign -Path $msiPath

    Write-Host ""
    Write-Host "=== Build Complete ===" -ForegroundColor Cyan
    Write-Host "MSI: $msiPath"
}

try {
    Build-Msi
} catch {
    Write-Host "Build failed: $_" -ForegroundColor Red
    exit 1
}
