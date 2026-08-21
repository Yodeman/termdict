<#
.SYNOPSIS
    TermDict installer for Windows.

.DESCRIPTION
    Installs the termdict binary to %LOCALAPPDATA%\Programs\termdict\bin and
    adds that directory to the user PATH. The script is not code-signed;
    review it before running, or use the zip from the releases page manually.

.EXAMPLE
    .\install.ps1                          # latest release
    .\install.ps1 -Version v0.2.0
    .\install.ps1 -FromDir .\dist          # dev: local goreleaser output
    .\install.ps1 -Uninstall
#>
param(
    [string]$Version = "latest",
    [string]$FromDir = "",
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"
$Repo = "yodeman/termdict"
$BinDir = Join-Path $env:LOCALAPPDATA "Programs\termdict\bin"
$ExePath = Join-Path $BinDir "termdict.exe"

function Remove-FromUserPath([string]$dir) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -and ($userPath.Split(";") -contains $dir)) {
        $new = ($userPath.Split(";") | Where-Object { $_ -ne $dir }) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $new, "User")
        Write-Host "Removed '$dir' from your user PATH."
    }
}

if ($Uninstall) {
    if (Test-Path $ExePath) {
        Remove-Item $ExePath -Force
        Write-Host "Removed $ExePath."
        Remove-FromUserPath $BinDir
        Write-Host "Your dictionary data was kept (%LOCALAPPDATA%\termdict)."
    } else {
        Write-Host "No termdict.exe found at $ExePath."
    }
    exit 0
}

switch -Regex ($env:PROCESSOR_ARCHITECTURE) {
    "^ARM64" { $arch = "arm64" }
    default  { $arch = "amd64" }
}

function Resolve-Tag([string]$requested) {
    if ($requested -ne "latest") {
        if (-not $requested.StartsWith("v")) { return "v$requested" }
        return $requested
    }
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    return $release.tag_name
}

$tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([System.Guid]::NewGuid().ToString()))
try {
    if ($FromDir -ne "") {
        $candidates = @(
            (Join-Path $FromDir "termdict_v${Version}_windows_${arch}.zip"),
            (Join-Path $FromDir "termdict_windows_${arch}.zip")
        ) | Where-Object { Test-Path $_ }
        if (-not $candidates) {
            throw "no windows/$arch archive found in $FromDir"
        }
        $zipPath = $candidates[0]
        Write-Host "Installing termdict ${Version} (windows/${arch}) from $FromDir..."
    } else {
        $tag = Resolve-Tag $Version
        Write-Host "Installing termdict $tag (windows/$arch)..."
        $base = "https://github.com/$Repo/releases/download/$tag"
        $zipPath = Join-Path $tmp "termdict.zip"
        $sumsPath = Join-Path $tmp "checksums.txt"

        Invoke-WebRequest -Uri "$base/termdict_${tag}_windows_${arch}.zip" -OutFile $zipPath

        $verified = $false
        try {
            Invoke-WebRequest -Uri "$base/termdict_checksums.txt" -OutFile $sumsPath
            $archiveName = Split-Path $zipPath -Leaf
            $want = (Get-Content $sumsPath | Where-Object { $_ -like "*$archiveName*" }) -split "\s+" | Select-Object -First 1
            if ($want) {
                $got = (Get-FileHash -Algorithm SHA256 $zipPath).Hash.ToLower()
                if ($got -ne $want.ToLower()) { throw "checksum mismatch for downloaded archive" }
                Write-Host "Checksum verified."
                $verified = $true
            }
        } catch {
            Write-Warning "checksums not available for this release; skipping verification"
        }
    }

    $extractDir = Join-Path $tmp "extracted"
    Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force
    $exe = Get-ChildItem -Path $extractDir -Filter "termdict.exe" -Recurse | Select-Object -First 1
    if (-not $exe) { throw "archive did not contain a termdict.exe binary" }

    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    Copy-Item $exe.FullName $ExePath -Force

    & $ExePath --version
    Write-Host "Installed to $ExePath."

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $userPath -or -not ($userPath.Split(";") -contains $BinDir)) {
        $newPath = if ($userPath) { "$userPath;$BinDir" } else { $BinDir }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host ""
        Write-Host "Added '$BinDir' to your user PATH."
        Write-Host "Restart your terminal for 'termdict' to be found on PATH."
    }
} finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
