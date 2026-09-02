param(
    [string]$Version = $(if ($env:LOC_VISUALS_VERSION) { $env:LOC_VISUALS_VERSION } else { "latest" }),
    [string]$InstallDir = $(if ($env:LOC_VISUALS_INSTALL_DIR) { $env:LOC_VISUALS_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\loc-visuals\bin" })
)

$ErrorActionPreference = "Stop"
$repository = "https://github.com/EzyGang/loc-visuals"

if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
    throw "loc-visuals installer: this script supports Windows only"
}

$architecture = switch ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
    "X64" { "amd64" }
    "Arm64" { "arm64" }
    default { throw "loc-visuals installer: unsupported architecture" }
}

if ($Version -eq "latest") {
    $headers = @{ "User-Agent" = "loc-visuals-installer" }
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/EzyGang/loc-visuals/releases/latest" -Headers $headers
    $tag = $release.tag_name
    $Version = $tag -replace '^v', ''
} else {
    $Version = $Version -replace '^v', ''
    $tag = "v$Version"
}

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "loc-visuals installer: invalid release version: $Version"
}

$archive = "loc-visuals-$Version-windows-$architecture.zip"
$downloadRoot = "$repository/releases/download/$tag"
$temporary = Join-Path ([IO.Path]::GetTempPath()) "loc-visuals-$([Guid]::NewGuid())"
$expanded = Join-Path $temporary "expanded"
New-Item -ItemType Directory -Path $temporary, $expanded -Force | Out-Null

try {
    $archivePath = Join-Path $temporary $archive
    $checksumsPath = Join-Path $temporary "SHA256SUMS"
    Invoke-WebRequest -Uri "$downloadRoot/$archive" -OutFile $archivePath -UseBasicParsing
    Invoke-WebRequest -Uri "$downloadRoot/SHA256SUMS" -OutFile $checksumsPath -UseBasicParsing

    $checksumLine = Get-Content $checksumsPath | Where-Object { $_ -match "\s$([Regex]::Escape($archive))$" } | Select-Object -First 1
    if (-not $checksumLine) {
        throw "loc-visuals installer: checksum is missing for $archive"
    }
    $expected = ($checksumLine -split '\s+')[0].ToLowerInvariant()
    $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "loc-visuals installer: checksum verification failed for $archive"
    }

    Expand-Archive -Path $archivePath -DestinationPath $expanded -Force
    $binary = Join-Path $expanded "loc-visuals.exe"
    if (-not (Test-Path $binary -PathType Leaf)) {
        throw "loc-visuals installer: archive does not contain loc-visuals.exe"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path $binary -Destination (Join-Path $InstallDir "loc-visuals.exe") -Force

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = if ($userPath) { $userPath -split ';' } else { @() }
    if ($pathEntries -notcontains $InstallDir) {
        $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$env:Path;$InstallDir"
    }

    Write-Output "Installed loc-visuals $Version to $InstallDir\loc-visuals.exe"
} finally {
    Remove-Item -Path $temporary -Recurse -Force -ErrorAction SilentlyContinue
}
