param(
    [switch]$IncludeWindows = $false
)

$ErrorActionPreference = "Stop"
$Version = "Test-v3.4"
$RootDir = Resolve-Path "$PSScriptRoot\.."
$DistDir = "$RootDir\dist"
$ReleaseDir = "$DistDir\release"

Write-Host "📦 Starting CubeX Test v3.4 Complete Release Packaging (Dual-Core Embedded)..." -ForegroundColor Cyan
if (-not $IncludeWindows) {
    Write-Host "ℹ️  Windows 平台包发布已按策略暂停 (可通过 -IncludeWindows 显式启用)" -ForegroundColor DarkCyan
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

Set-Location -Path $RootDir
$env:CGO_ENABLED = "0"

if ($IncludeWindows) {
    # Windows amd64
    Write-Host "   -> Compiling Windows cubexd.exe..." -ForegroundColor Green
    go build -ldflags="-s -w" -o "$DistDir\cubexd.exe" ./cmd/cubexd
} else {
    Write-Host "   -> Skipping Windows cubexd.exe build (暂时不发布windows包)" -ForegroundColor DarkGray
}

# Linux amd64
Write-Host "   -> Cross-compiling Linux amd64 cubexd & cubex CLI..." -ForegroundColor Green
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o "$DistDir\cubexd-linux-amd64" ./cmd/cubexd
go build -ldflags="-s -w" -o "$DistDir\cubex-linux-amd64" ./cmd/cubex

# Linux arm64
Write-Host "   -> Cross-compiling Linux arm64 cubexd & cubex CLI..." -ForegroundColor Green
$env:GOARCH = "arm64"
go build -ldflags="-s -w" -o "$DistDir\cubexd-linux-arm64" ./cmd/cubexd
go build -ldflags="-s -w" -o "$DistDir\cubex-linux-arm64" ./cmd/cubex

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

    # Copy Geodata and 217heidai Dual-Format Rules (.mrs for Mihomo, .srs for sing-box)
    Copy-Item -Path "$RootDir\etc\*" -Destination "$StageDir\etc" -Recurse -Force
    Copy-Item -Path "$RootDir\data\*" -Destination "$StageDir\data" -Recurse -Force
    if (Test-Path "$StageDir\data\ros_backup") { Remove-Item -Path "$StageDir\data\ros_backup" -Recurse -Force }

    # Strict Desensitization: strip runtime caches, logs, databases, and sanitize configs
    python "$RootDir\scripts\desensitize.py" $StageDir | Out-Host

    # Copy Deployment & Script Tools
    Copy-Item -Path "$RootDir\meta.json" -Destination "$StageDir\meta.json" -Force
    Copy-Item -Path "$RootDir\install.sh" -Destination "$StageDir\install.sh" -Force
    Copy-Item -Path "$RootDir\docker-compose.yml" -Destination "$StageDir\docker-compose.yml" -Force
    Copy-Item -Path "$RootDir\README.md" -Destination "$StageDir\README.md" -Force
    Copy-Item -Path "$RootDir\LICENSE" -Destination "$StageDir\LICENSE" -Force

    return $StageDir
}

# 4. Packaging Linux amd64 & arm64 (.zip format, NO .tar.gz)
Write-Host "📦 [2/4] Packaging Linux amd64 Release (Dual-Core + Geodata + Rules -> .zip)..." -ForegroundColor Yellow
$stageAmd64 = Assemble-ReleasePackage "CubeX-$Version-linux-amd64" "$DistDir\cubexd-linux-amd64" "amd64"
Compress-Archive -Path "$stageAmd64\*" -DestinationPath "$ReleaseDir\CubeX-$Version-linux-amd64.zip" -Force

Write-Host "📦 [3/4] Packaging Linux arm64 Release (Dual-Core + Geodata + Rules -> .zip)..." -ForegroundColor Yellow
$stageArm64 = Assemble-ReleasePackage "CubeX-$Version-linux-arm64" "$DistDir\cubexd-linux-arm64" "arm64"
Compress-Archive -Path "$stageArm64\*" -DestinationPath "$ReleaseDir\CubeX-$Version-linux-arm64.zip" -Force

# 5. Packaging Windows amd64 (.zip) - Conditional
if ($IncludeWindows) {
    Write-Host "📦 [*] Packaging Windows amd64 Release (Dual-Core .exe + WebUI -> .zip)..." -ForegroundColor Yellow
    $stageWin = Assemble-ReleasePackage "CubeX-$Version-windows-amd64" "$DistDir\cubexd.exe" "windows"
    Compress-Archive -Path "$stageWin\*" -DestinationPath "$ReleaseDir\CubeX-$Version-windows-amd64.zip" -Force
} else {
    Write-Host "⏩ Skipping Windows amd64 packaging (暂时不发布 Windows 包)..." -ForegroundColor DarkGray
}

# 6. Packaging Dedicated RouterOS v7 Container Packages (Uncompressed .tar for WinBox /container/add)
Write-Host "📦 [4/4] Packaging RouterOS v7 Dedicated Container Images (Uncompressed .tar + RSC Script)..." -ForegroundColor Yellow

# Helper to assemble RouterOS container rootfs .tar
function Assemble-RosContainer($TargetName, $GatewayBinary, $ArchType) {
    $RosStage = "$DistDir\staging\$TargetName"
    if (Test-Path $RosStage) { Remove-Item -Path $RosStage -Recurse -Force }
    New-Item -ItemType Directory -Path "$RosStage\app\bin" -Force | Out-Null
    New-Item -ItemType Directory -Path "$RosStage\app\etc" -Force | Out-Null
    New-Item -ItemType Directory -Path "$RosStage\app\data" -Force | Out-Null

    # Binaries inside container /app/bin
    Copy-Item -Path $GatewayBinary -Destination "$RosStage\app\bin\cubexd" -Force
    if (Test-Path "$DistDir\cubex-linux-$ArchType") {
        Copy-Item -Path "$DistDir\cubex-linux-$ArchType" -Destination "$RosStage\app\bin\cubex" -Force
    }
    Copy-Item -Path "$RootDir\bin\$ArchType\mihomo" -Destination "$RosStage\app\bin\mihomo" -Force
    Copy-Item -Path "$RootDir\bin\$ArchType\sing-box" -Destination "$RosStage\app\bin\sing-box" -Force

    # Rulesets & configs inside container
    Copy-Item -Path "$RootDir\meta.json" -Destination "$RosStage\app\meta.json" -Force
    Copy-Item -Path "$RootDir\etc\*" -Destination "$RosStage\app\etc" -Recurse -Force
    Copy-Item -Path "$RootDir\data\*" -Destination "$RosStage\app\data" -Recurse -Force
    if (Test-Path "$RosStage\app\data\ros_backup") { Remove-Item -Path "$RosStage\app\data\ros_backup" -Recurse -Force }

    # Strict Desensitization: strip runtime caches, logs, databases, and sanitize configs
    python "$RootDir\scripts\desensitize.py" $RosStage | Out-Host

    # Uncompressed .tar (RouterOS strictly requires uncompressed .tar, NOT .tar.gz)
    tar -cf "$ReleaseDir\$TargetName.tar" -C "$RosStage" app
    return $RosStage
}

Assemble-RosContainer "CubeX-$Version-ros-amd64" "$DistDir\cubexd-linux-amd64" "amd64" | Out-Null
Assemble-RosContainer "CubeX-$Version-ros-arm64" "$DistDir\cubexd-linux-arm64" "arm64" | Out-Null

# Build true OCI images for RouterOS container engine
python "$RootDir\scripts\build_v21_ros_containers.py" | Out-Host

# Build fnOS / Linux Docker offline image
python "$RootDir\scripts\build_docker_tar_v30.py" | Out-Host

# Copy standalone cubexd gateway binaries for 1-click in-place hot-upgrade
Copy-Item -Path "$DistDir\cubexd-linux-amd64" -Destination "$ReleaseDir\cubexd-linux-amd64" -Force
Copy-Item -Path "$DistDir\cubexd-linux-arm64" -Destination "$ReleaseDir\cubexd-linux-arm64" -Force

# Copy RouterOS WinBox 1-Click RSC Script & OpenWrt 1-Click Shell Script
Copy-Item -Path "$RootDir\scripts\ros\setup-routeros-v7.rsc" -Destination "$ReleaseDir\setup-routeros-v7.rsc" -Force
Copy-Item -Path "$RootDir\scripts\openwrt\setup-openwrt.sh" -Destination "$ReleaseDir\setup-openwrt.sh" -Force
Copy-Item -Path "$RootDir\scripts\install_pve_lxc.sh" -Destination "$ReleaseDir\install_pve_lxc.sh" -Force

# Clean up staging
Remove-Item -Path "$DistDir\staging" -Recurse -Force

Write-Host "`n🎉 All Release Packages Successfully Generated in:" -ForegroundColor Magenta
Write-Host "   $ReleaseDir" -ForegroundColor White
Get-ChildItem -Path $ReleaseDir | Select-Object Name, Length

# 7. Strict Automated Post-Packaging Desensitization & Unpack Audit
Write-Host "`n🛡️  Executing Mandatory Post-Packaging Desensitization Audit (.tar & .zip independent unpack)..." -ForegroundColor Cyan
$auditArgs = @("$ReleaseDir")
if ($IncludeWindows) {
    $auditArgs += "--allow-windows"
}
python "$RootDir\scripts\audit_release_artifacts.py" @auditArgs
if ($LASTEXITCODE -ne 0) {
    Write-Error "🚨 Post-Packaging Desensitization Audit Failed! Release is blocked."
}



# # ═════════════════════════════════════════════════════════════════════════════
# CubeX Release Packaging Script (Bundles Dual Cores + Geodata + Rules + Config)
# Target Version: Test v2.5
# Formats: Native .zip for General Releases + Uncompressed .tar for RouterOS
# ═════════════════════════════════════════════════════════════════════════════