param(
    [switch]$IncludeWindows = $false,
    [switch]$SkipHealthAudit = $false,
    [switch]$SkipRetention = $false
)



$ErrorActionPreference = "Stop"

$Version = "v9.993"

$RootDir = Resolve-Path "$PSScriptRoot\.."

$DistDir = "$RootDir\dist"

$ReleaseDir = "$DistDir\release"



Write-Host "📦 Starting CubeX $Version Complete Release Packaging (Dual-Core Embedded)..." -ForegroundColor Cyan

if (-not $IncludeWindows) {
    Write-Host "ℹ️  Windows 平台包发布已按策略暂缓 (可通过 -IncludeWindows 显式启用)" -ForegroundColor DarkCyan
}

# Pre-clean stale release artifacts before audit to avoid checking outdated builds
if (Test-Path $ReleaseDir) {
    Remove-Item -Path "$ReleaseDir\*" -Recurse -Force -ErrorAction SilentlyContinue
}

# 0. Mandatory Pre-Release Full-Stack Health Audit
if (-not $SkipHealthAudit) {
    Write-Host "🩺 [0/4] Executing Mandatory Full-Stack Health Audit Doctor..." -ForegroundColor Yellow

    python "$PSScriptRoot\audit_codebase_health.py"

    if ($LASTEXITCODE -ne 0) {

        Write-Error "Pre-release codebase health audit failed! Release aborted."

        exit 1

    }
} else {
    Write-Host "🛡️ [0/4] Skipping live CT107/Router probe in health audit (-SkipHealthAudit)" -ForegroundColor DarkCyan
}



# 1. Clean & Prepare Release Directory

if (Test-Path $ReleaseDir) {

    Remove-Item -Path $ReleaseDir -Recurse -Force

}

New-Item -ItemType Directory -Path $ReleaseDir -Force | Out-Null



# 2. Build Web Frontend & Multi-Arch Binaries

Write-Host "🔨 [1/4] Building Web Frontend & Multi-Arch Gateway Binaries..." -ForegroundColor Yellow

Set-Location -Path "$RootDir\web"

cmd /c npm run build
python "$RootDir\scripts\precompress_dist.py"



Set-Location -Path $RootDir

$env:CGO_ENABLED = "0"



if ($IncludeWindows) {

    # Windows amd64

    Write-Host "   -> Compiling Windows cubexd.exe..." -ForegroundColor Green

    go build -ldflags="-s -w" -o "$DistDir\cubexd.exe" ./cmd/cubexd

} else {
    Write-Host "   -> Skipping Windows cubexd.exe build (per no-Windows release policy)" -ForegroundColor DarkGray
}

# Linux amd64 & arm64 Pure Static Compilation
Write-Host "   -> Cross-compiling Linux amd64 & arm64 cubexd & cubex CLI..." -ForegroundColor Green
python "$RootDir\scripts\compile_linux.py"
if ($LASTEXITCODE -ne 0) {
    Write-Error "Linux compilation failed!"
    exit 1
}

# 3. Helper to assemble package staging directory

function Assemble-ReleasePackage($TargetName, $GatewayBinary, $ArchType) {

    $StageDir = "$DistDir\staging\$TargetName"

    if (Test-Path $StageDir) { Remove-Item -Path $StageDir -Recurse -Force }

    New-Item -ItemType Directory -Path $StageDir -Force | Out-Null

    New-Item -ItemType Directory -Path "$StageDir\bin" -Force | Out-Null

    New-Item -ItemType Directory -Path "$StageDir\etc" -Force | Out-Null

    New-Item -ItemType Directory -Path "$StageDir\data" -Force | Out-Null



    # Copy Gateway binary

    if ($ArchType -eq "windows") {

        Copy-Item -Path $GatewayBinary -Destination "$StageDir\cubexd.exe" -Force

        Copy-Item -Path "$RootDir\bin\windows\mihomo.exe" -Destination "$StageDir\bin\mihomo.exe" -Force

        Copy-Item -Path "$RootDir\bin\windows\sing-box.exe" -Destination "$StageDir\bin\sing-box.exe" -Force

    } else {

        Copy-Item -Path $GatewayBinary -Destination "$StageDir\bin\cubexd" -Force

        if (Test-Path "$DistDir\cubex-linux-$ArchType") {

            Copy-Item -Path "$DistDir\cubex-linux-$ArchType" -Destination "$StageDir\bin\cubex" -Force

        }

        Copy-Item -Path "$RootDir\bin\$ArchType\mihomo" -Destination "$StageDir\bin\mihomo" -Force

        Copy-Item -Path "$RootDir\bin\$ArchType\sing-box" -Destination "$StageDir\bin\sing-box" -Force

    }



    # Copy Clean-Room Default IR, Geodata (GeoIP.dat + geosite.dat) and 101 MRS/SRS Rules
    $RulesDst = "$StageDir\etc\rules"
    New-Item -ItemType Directory -Path $RulesDst -Force | Out-Null
    Copy-Item -Path "$RootDir\etc\rules\*" -Destination $RulesDst -Recurse -Force
    foreach ($geo in @("GeoIP.dat", "geosite.dat")) {
        if (Test-Path "$RootDir\etc\$geo") {
            Copy-Item -Path "$RootDir\etc\$geo" -Destination "$StageDir\etc\$geo" -Force
        }
    }
    Copy-Item -Path "$RootDir\packaging\defaults\data\*" -Destination "$StageDir\data" -Recurse -Force

    # Strict Desensitization: secondary safety check
    python "$RootDir\scripts\desensitize.py" $StageDir | Out-Host



    # Copy Deployment & Script Tools

    Copy-Item -Path "$RootDir\meta.json" -Destination "$StageDir\meta.json" -Force

    Copy-Item -Path "$RootDir\install.sh" -Destination "$StageDir\install.sh" -Force

    Copy-Item -Path "$RootDir\docker-compose.yml" -Destination "$StageDir\docker-compose.yml" -Force

    Copy-Item -Path "$RootDir\README.md" -Destination "$StageDir\README.md" -Force

    Copy-Item -Path "$RootDir\LICENSE" -Destination "$StageDir\LICENSE" -Force



    return $StageDir

}



function Create-PosixZip {
    param(
        [string]$SourceDir,
        [string]$DestinationZip
    )
    $pyScript = @"
import os, sys, zipfile
src = sys.argv[1]
dst = sys.argv[2]
if os.path.exists(dst):
    try:
        os.remove(dst)
    except Exception:
        pass
with zipfile.ZipFile(dst, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as z:
    for root, dirs, files in os.walk(src):
        for f in files:
            full = os.path.join(root, f)
            rel = os.path.relpath(full, src).replace('\\', '/')
            is_bin = rel.startswith('bin/') or rel.endswith('.sh')
            attr = (0o755 << 16) if is_bin else (0o644 << 16)
            zinfo = zipfile.ZipInfo(rel)
            zinfo.external_attr = attr
            with open(full, 'rb') as fp:
                z.writestr(zinfo, fp.read(), compress_type=zipfile.ZIP_DEFLATED)
"@
    python -c $pyScript "$SourceDir" "$DestinationZip"
    if (-not (Test-Path "$DestinationZip")) {
        Compress-Archive -Path "$SourceDir\*" -DestinationPath "$DestinationZip" -Force
    }
}

# 4. Packaging Linux amd64 & arm64 (.zip format, NO .tar.gz)

Write-Host "📦 [2/4] Packaging Linux amd64 Release (Dual-Core + Geodata + Rules -> .zip)..." -ForegroundColor Yellow

$stageAmd64 = Assemble-ReleasePackage "CubeX-$Version-linux-amd64" "$DistDir\cubexd-linux-amd64" "amd64"

Create-PosixZip "$stageAmd64" "$ReleaseDir\CubeX-$Version-linux-amd64.zip"



Write-Host "📦 [3/4] Packaging Linux arm64 Release (Dual-Core + Geodata + Rules -> .zip)..." -ForegroundColor Yellow

$stageArm64 = Assemble-ReleasePackage "CubeX-$Version-linux-arm64" "$DistDir\cubexd-linux-arm64" "arm64"

Create-PosixZip "$stageArm64" "$ReleaseDir\CubeX-$Version-linux-arm64.zip"



# 5. Packaging Windows amd64 (.zip) - Conditional

if ($IncludeWindows) {

    Write-Host "📦 [*] Packaging Windows amd64 Release (Dual-Core .exe + WebUI -> .zip)..." -ForegroundColor Yellow

    $stageWin = Assemble-ReleasePackage "CubeX-$Version-windows-amd64" "$DistDir\cubexd.exe" "windows"

    Compress-Archive -Path "$stageWin\*" -DestinationPath "$ReleaseDir\CubeX-$Version-windows-amd64.zip" -Force

} else {
    Write-Host "   -> Skipping Windows amd64 packaging (per no-Windows release policy)..." -ForegroundColor DarkGray
}

# 6. Packaging Unified Clean-Room RouterOS OCI & Docker Offline Archives
Write-Host "📦 [4/4] Packaging Clean-Room RouterOS (.tar) & Docker (.tar.gz) Container Images..." -ForegroundColor Yellow
python "$RootDir\scripts\build_oci.py" --arch all --format all | Out-Host
if ($LASTEXITCODE -ne 0) {
    Write-Error "Unified OCI/Docker packaging failed!"
    exit 1
}

# Build fnOS Native .fpk Packages
python "$RootDir\scripts\build_fnos_fpk.py" | Out-Host
if (Test-Path "$RootDir\dist-fpk") {
    Copy-Item -Path "$RootDir\dist-fpk\*.fpk" -Destination $ReleaseDir -Force
}



# Copy standalone cubexd gateway binaries for 1-click in-place hot-upgrade

Copy-Item -Path "$DistDir\cubexd-linux-amd64" -Destination "$ReleaseDir\cubexd-linux-amd64" -Force

Copy-Item -Path "$DistDir\cubexd-linux-arm64" -Destination "$ReleaseDir\cubexd-linux-arm64" -Force



# Copy standalone CLI binaries

if (Test-Path "$DistDir\cubex-linux-amd64") {

    Copy-Item -Path "$DistDir\cubex-linux-amd64" -Destination "$ReleaseDir\cubex-linux-amd64" -Force

}

if (Test-Path "$DistDir\cubex-linux-arm64") {

    Copy-Item -Path "$DistDir\cubex-linux-arm64" -Destination "$ReleaseDir\cubex-linux-arm64" -Force

}



# Protect & Encrypt deployment scripts (Zero Source Code Exposure)

python "$RootDir\scripts\protect_script.py" | Out-Host



# Clean up staging
Remove-Item -Path "$DistDir\staging" -Recurse -Force

# Generate SHA256SUMS for all release assets
Write-Host "🔒 Generating SHA256SUMS for release assets..." -ForegroundColor Cyan
$shaLines = @()
Get-ChildItem -Path $ReleaseDir -File | Where-Object { $_.Name -ne "SHA256SUMS" } | ForEach-Object {
    $hash = (Get-FileHash -Path $_.FullName -Algorithm SHA256).Hash.ToLower()
    $shaLines += "$hash  $($_.Name)"
}
$shaLines | Out-File -FilePath "$ReleaseDir\SHA256SUMS" -Encoding ascii -Force
Write-Host "✓ SHA256SUMS generated successfully" -ForegroundColor Green

Write-Host "`n🎉 All Release Packages Successfully Generated in:" -ForegroundColor Magenta
Write-Host "   $ReleaseDir" -ForegroundColor White
Get-ChildItem -Path $ReleaseDir | Select-Object Name, Length



# 7. Strict Automated Post-Packaging Desensitization, Size-Gate & Unpack Audit

Write-Host "`n🛡️ Executing Mandatory Post-Packaging Desensitization & Size-Gate Audit (.tar & .zip independent unpack)..." -ForegroundColor Cyan

$auditArgs = @("$ReleaseDir", "--expected-version", "$Version", "--size-gate")

if ($IncludeWindows) {
    $auditArgs += "--allow-windows"
}

python "$RootDir\scripts\audit_release_artifacts.py" @auditArgs

if ($LASTEXITCODE -ne 0) {
    Write-Error "🚨 Post-Packaging Desensitization / Size-Gate Audit Failed! Release is blocked."
    exit 1
}

if (-not $SkipRetention) {
    # Enforce Strict 10-Version Retention Policy Across All Platforms
    python "$RootDir\scripts\enforce_10_version_retention.py"
} else {
    Write-Host "ℹ️  Skipping remote 10-version retention cleanup (-SkipRetention)" -ForegroundColor DarkCyan
}