param(
    [string]$Version = 'latest',
    [string]$InstallDir = (Join-Path $HOME '.local\bin'),
    [string]$AssetUrl = '',
    [string]$ChecksumUrl = ''
)

$ErrorActionPreference = 'Stop'
$assetName = 'fidelius-windows-amd64.zip'
if (-not $AssetUrl) {
    if ($Version -eq 'latest') {
        $AssetUrl = "https://github.com/amxv/fidelius/releases/latest/download/$assetName"
    } else {
        if (-not $Version.StartsWith('v')) { $Version = "v$Version" }
        $AssetUrl = "https://github.com/amxv/fidelius/releases/download/$Version/$assetName"
    }
}
if (-not $ChecksumUrl) { $ChecksumUrl = "$AssetUrl.sha256" }

$temporary = Join-Path ([IO.Path]::GetTempPath()) ("fidelius-install-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $temporary | Out-Null
try {
    $archive = Join-Path $temporary $assetName
    $checksum = "$archive.sha256"
    if (Test-Path -LiteralPath $AssetUrl -PathType Leaf) {
        Copy-Item -LiteralPath $AssetUrl -Destination $archive
    } else {
        Invoke-WebRequest -Uri $AssetUrl -OutFile $archive -UseBasicParsing
    }
    if (Test-Path -LiteralPath $ChecksumUrl -PathType Leaf) {
        Copy-Item -LiteralPath $ChecksumUrl -Destination $checksum
    } else {
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile $checksum -UseBasicParsing
    }
    $expected = ((Get-Content -Path $checksum -TotalCount 1) -split '\s+')[0].ToUpperInvariant()
    $actual = (Get-FileHash -Path $archive -Algorithm SHA256).Hash.ToUpperInvariant()
    if ($expected -notmatch '^[A-F0-9]{64}$' -or $actual -ne $expected) {
        throw 'SHA-256 verification failed'
    }

    $payload = Join-Path $temporary 'payload'
    Expand-Archive -Path $archive -DestinationPath $payload
    $source = Join-Path $payload 'fidelius.exe'
    if (-not (Test-Path -LiteralPath $source -PathType Leaf)) { throw 'release is missing fidelius.exe' }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $destination = Join-Path $InstallDir 'fidelius.exe'
    $staged = Join-Path $InstallDir ('.fidelius-install-' + [Guid]::NewGuid().ToString('N') + '.exe')
    try {
        Copy-Item -LiteralPath $source -Destination $staged
        Move-Item -LiteralPath $staged -Destination $destination -Force
    } finally {
        Remove-Item -LiteralPath $staged -Force -ErrorAction SilentlyContinue
    }

    $resolvedDir = (Resolve-Path -LiteralPath $InstallDir).Path
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $entries = @($userPath -split ';' | Where-Object { $_ })
    if ($entries -notcontains $resolvedDir) {
        [Environment]::SetEnvironmentVariable('Path', (($entries + $resolvedDir) -join ';'), 'User')
    }
    if (($env:Path -split ';') -notcontains $resolvedDir) { $env:Path += ";$resolvedDir" }
    Write-Host "Installed Fidelius to $destination"
    Write-Host 'Open a new PowerShell window to use fidelius.'
} finally {
    Remove-Item -LiteralPath $temporary -Recurse -Force -ErrorAction SilentlyContinue
}
