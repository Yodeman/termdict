# Runtime smoke test for install.ps1 — mirrors the documented one-liner
# invocation exactly (plan v2 Phase 1 M4 / requirement R3).
#
# Runs entirely offline: builds a local zip fixture and installs from it
# via -FromDir. Asserts:
#   1. real code paths execute (install + --version + uninstall)
#   2. the session SURVIVES the invocation (install.ps1 must not "exit",
#      which under "irm | iex" would close the user's shell)
#   3. no scope pollution (script variables stay inside the scriptblock)
#
# Invoked by .github/workflows/ci.yml (windows-install-smoke job).
param(
    [string]$InstallScript = "install.ps1",
    [string]$ExeSource = "termdict.exe"
)

$ErrorActionPreference = "Stop"
$failures = 0
function Check([string]$name, [bool]$ok) {
    if ($ok) { Write-Host "ok   $name" }
    else { Write-Host "FAIL $name"; $script:failures++ }
}

# --- fixture: zip the freshly built exe under the release naming scheme ---
$fixture = Join-Path $env:TEMP ("termdict-smoke-" + [System.Guid]::NewGuid().ToString("N"))
$extract = Join-Path $fixture "payload"
New-Item -ItemType Directory -Path $extract -Force | Out-Null
Copy-Item $ExeSource (Join-Path $extract "termdict.exe")
$zip = Join-Path $fixture "termdict_v0.2.1-test_windows_amd64.zip"
Compress-Archive -Path (Join-Path $extract "*") -DestinationPath $zip -Force

# Pre-existing PATH entry guard: remember whether BinDir was already present.
$binDir = Join-Path $env:LOCALAPPDATA "Programs\termdict\bin"
$hadPath = ([Environment]::GetEnvironmentVariable("Path", "User")) -and `
    (([Environment]::GetEnvironmentVariable("Path", "User")).Split(";") -contains $binDir)

try {
    # --- invoke exactly like the documented one-liner, from local files ---
    & ([scriptblock]::Create((Get-Content $InstallScript -Raw))) -FromDir $fixture -Version 0.2.1-test

    # If install.ps1 had used "exit", execution would never reach this line.
    Check "session survived invocation (no exit inside script)" $true

    $exePath = Join-Path $binDir "termdict.exe"
    Check "binary installed" (Test-Path $exePath)

    $versionOutput = & $exePath --version
    Check "--version runs ($versionOutput)" ($LASTEXITCODE -eq 0)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathAdded = ($userPath -and ($userPath.Split(";") -contains $binDir))
    Check "user PATH contains install dir (or was already present)" ($pathAdded -or $hadPath)

    # --- scope pollution: wrapper must keep script variables inside ---
    Check "no scope pollution (Repo)"     ($null -eq (Get-Variable Repo -ErrorAction SilentlyContinue))
    Check "no scope pollution (BinDir)"   ($null -eq (Get-Variable BinDir -ErrorAction SilentlyContinue))
    Check "no scope pollution (ExePath)"  ($null -eq (Get-Variable ExePath -ErrorAction SilentlyContinue))

    # --- uninstall path also runs in-session ---
    & ([scriptblock]::Create((Get-Content $InstallScript -Raw))) -Uninstall
    Check "session survived uninstall (no exit inside script)" $true
    Check "uninstall removed the binary" (-not (Test-Path $exePath))
} finally {
    Remove-Item $fixture -Recurse -Force -ErrorAction SilentlyContinue
    # Leave the runner as we found it.
    if (-not $hadPath) {
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -and ($userPath.Split(";") -contains $binDir)) {
            $new = ($userPath.Split(";") | Where-Object { $_ -ne $binDir }) -join ";"
            [Environment]::SetEnvironmentVariable("Path", $new, "User")
        }
    }
}

Write-Host ""
if ($failures -gt 0) {
    Write-Host "INSTALL SMOKE FAILED: $failures check(s)"
    exit 1
}
Write-Host "INSTALL SMOKE PASSED"
