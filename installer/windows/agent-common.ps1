function Assert-CoreUrl([string]$Url) {
    $uri = $null
    if (-not [Uri]::TryCreate($Url, [UriKind]::Absolute, [ref]$uri) -or
        $uri.Scheme -notin @('http', 'https') -or -not $uri.Host -or $uri.UserInfo) {
        throw 'Server URL must be an absolute HTTP or HTTPS URL without embedded credentials.'
    }
}

function Read-AgentConfig([string]$Path) {
    $config = Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
    Assert-CoreUrl $config.core_url
    if (-not $config.data_dir -or -not [IO.Path]::IsPathRooted($config.data_dir)) {
        throw 'Agent data_dir must be an absolute path.'
    }
    return $config
}

function Protect-AgentDirectory([string]$Path) {
    if (-not [IO.Path]::IsPathRooted($Path) -or
        [IO.Path]::GetFullPath($Path).TrimEnd('\') -eq [IO.Path]::GetPathRoot($Path).TrimEnd('\')) {
        throw 'Agent installation and data directories must not be a drive root or relative path.'
    }
    $acl = New-Object Security.AccessControl.DirectorySecurity
    $acl.SetAccessRuleProtection($true, $false)
    foreach ($sid in @('S-1-5-18', 'S-1-5-32-544')) {
        $identity = New-Object Security.Principal.SecurityIdentifier($sid)
        $rule = New-Object Security.AccessControl.FileSystemAccessRule(
            $identity, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
        $acl.AddAccessRule($rule)
    }
    Set-Acl -LiteralPath $Path -AclObject $acl
}
