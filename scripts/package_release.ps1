# ═════════════════════════════════════════════════════════════════════════════
# CubeX Release Packaging Script (Bundles Dual Cores + Geodata + Rules + Config)
# Target Version: Test v0.999 (Next: v1.0)
# Formats: Native .zip for General Releases + Uncompressed .tar for RouterOS
# ═════════════════════════════════════════════════════════════════════════════
$ErrorActionPreference = "Stop"
$Version = "Test-v0.999"
$RootDir = Resolve-Path "$PSScriptRoot\.."
$DistDir = "$RootDir\dist"
$ReleaseDir = "$DistDir\release"

Write-Host "📦 Starting CubeX Test v0.999 Complete Release Packaging (Dual-Core Embedded)..." -ForegroundColor Cyan

# 1. Clean & Prepare Release Directory
if (Test-Path $ReleaseDir) {
    Remove-Item -Path $ReleaseDir -Recurse -Force
}
New-Item -ItemType Directory -Path $ReleaseDir -Force | Out-Null

# 2. Build Web Frontend & Multi-Arch Binaries
Write-Host "🔨 [1/5] Building Web Frontend & Multi-Arch Gateway Binaries..." -ForegroundColor Yellow
Set-Location -Path "$RootDir\web"
cmd /c npm run build

Set-Location -Path $RootDir
$env:CGO_ENABLED = "0"

# Windows amd64
Write-Host "   -> Compiling Windows cubexd.exe..." -ForegroundColor Green
go build -ldflags="-s -w" -o "$DistDir\cubexd.exe" ./cmd/cubexd

# Linux amd64
Write-Host "   -> Cross-compiling Linux amd64 cubexd..." -ForegroundColor Green
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o "$DistDir\cubexd-linux-amd64" ./cmd/cubexd

# Linux arm64
Write-Host "   -> Cross-compiling Linux arm64 cubexd..." -ForegroundColor Green
$env:GOARCH = "arm64"
go build -ldflags="-s -w" -o "$DistDir\cubexd-linux-arm64" ./cmd/cubexd

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
        Copy-Item -Path "$RootDir\bin\$ArchType\mihomo" -Destination "$StageDir\bin\mihomo" -Force
        Copy-Item -Path "$RootDir\bin\$ArchType\sing-box" -Destination "$StageDir\bin\sing-box" -Force
    }

    # Copy Geodata and 217heidai Dual-Format Rules (.mrs for Mihomo, .srs for sing-box)
    Copy-Item -Path "$RootDir\etc\*" -Destination "$StageDir\etc" -Recurse -Force
    Copy-Item -Path "$RootDir\data\*" -Destination "$StageDir\data" -Recurse -Force

    # Strict Desensitization: strip runtime caches, logs, databases, and temporary state
    Remove-Item -Path "$StageDir\etc\cache.db" -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "$StageDir\data\*.log" -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "$StageDir\data\*.sqlite" -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "$StageDir\data\*.pid" -Force -ErrorAction SilentlyContinue

    # Copy Deployment & Script Tools
    Copy-Item -Path "$RootDir\install.sh" -Destination "$StageDir\install.sh" -Force
    Copy-Item -Path "$RootDir\docker-compose.yml" -Destination "$StageDir\docker-compose.yml" -Force
    Copy-Item -Path "$RootDir\README.md" -Destination "$StageDir\README.md" -Force
    Copy-Item -Path "$RootDir\LICENSE" -Destination "$StageDir\LICENSE" -Force

    return $StageDir
}

# 4. Packaging Linux amd64 & arm64 (.zip format, NO .tar.gz)
Write-Host "📦 [2/5] Packaging Linux amd64 Release (Dual-Core + Geodata + Rules -> .zip)..." -ForegroundColor Yellow
$stageAmd64 = Assemble-ReleasePackage "CubeX-$Version-linux-amd64" "$DistDir\cubexd-linux-amd64" "amd64"
Compress-Archive -Path "$stageAmd64\*" -DestinationPath "$ReleaseDir\CubeX-$Version-linux-amd64.zip" -Force

Write-Host "📦 [3/5] Packaging Linux arm64 Release (Dual-Core + Geodata + Rules -> .zip)..." -ForegroundColor Yellow
$stageArm64 = Assemble-ReleasePackage "CubeX-$Version-linux-arm64" "$DistDir\cubexd-linux-arm64" "arm64"
Compress-Archive -Path "$stageArm64\*" -DestinationPath "$ReleaseDir\CubeX-$Version-linux-arm64.zip" -Force

# 5. Packaging Windows amd64 (.zip)
Write-Host "📦 [4/5] Packaging Windows amd64 Release (Dual-Core .exe + WebUI -> .zip)..." -ForegroundColor Yellow
$stageWin = Assemble-ReleasePackage "CubeX-$Version-windows-amd64" "$DistDir\cubexd.exe" "windows"
Compress-Archive -Path "$stageWin\*" -DestinationPath "$ReleaseDir\CubeX-$Version-windows-amd64.zip" -Force

# 6. Packaging Dedicated RouterOS v7 Container Packages (Uncompressed .tar for WinBox /container/add)
Write-Host "📦 [5/5] Packaging RouterOS v7 Dedicated Container Images (Uncompressed .tar + RSC Script)..." -ForegroundColor Yellow

# Helper to assemble RouterOS container rootfs .tar
function Assemble-RosContainer($TargetName, $GatewayBinary, $ArchType) {
    $RosStage = "$DistDir\staging\$TargetName"
    if (Test-Path $RosStage) { Remove-Item -Path $RosStage -Recurse -Force }
    New-Item -ItemType Directory -Path "$RosStage\app\bin" -Force | Out-Null
    New-Item -ItemType Directory -Path "$RosStage\app\etc" -Force | Out-Null
    New-Item -ItemType Directory -Path "$RosStage\app\data" -Force | Out-Null

    # Binaries inside container /app/bin
    Copy-Item -Path $GatewayBinary -Destination "$RosStage\app\bin\cubexd" -Force
    Copy-Item -Path "$RootDir\bin\$ArchType\mihomo" -Destination "$RosStage\app\bin\mihomo" -Force
    Copy-Item -Path "$RootDir\bin\$ArchType\sing-box" -Destination "$RosStage\app\bin\sing-box" -Force

    # Rulesets & configs inside container
    Copy-Item -Path "$RootDir\etc\*" -Destination "$RosStage\app\etc" -Recurse -Force
    Copy-Item -Path "$RootDir\data\*" -Destination "$RosStage\app\data" -Recurse -Force

    # Strict Desensitization: strip runtime caches, logs, databases, and temporary state
    Remove-Item -Path "$RosStage\app\etc\cache.db" -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "$RosStage\app\data\*.log" -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "$RosStage\app\data\*.sqlite" -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "$RosStage\app\data\*.pid" -Force -ErrorAction SilentlyContinue

    # Uncompressed .tar (RouterOS strictly requires uncompressed .tar, NOT .tar.gz)
    tar -cf "$ReleaseDir\$TargetName.tar" -C "$RosStage" app
    return $RosStage
}

Assemble-RosContainer "CubeX-$Version-ros-amd64" "$DistDir\cubexd-linux-amd64" "amd64" | Out-Null
Assemble-RosContainer "CubeX-$Version-ros-arm64" "$DistDir\cubexd-linux-arm64" "arm64" | Out-Null

# Copy RouterOS WinBox 1-Click RSC Script
Copy-Item -Path "$RootDir\scripts\ros\setup-routeros-v7.rsc" -Destination "$ReleaseDir\setup-routeros-v7.rsc" -Force

# Clean up staging
Remove-Item -Path "$DistDir\staging" -Recurse -Force

Write-Host "`n🎉 All Dual-Core Release Packages (ZIP & RouterOS TAR) Successfully Generated in:" -ForegroundColor Magenta
Write-Host "   $ReleaseDir" -ForegroundColor White
Get-ChildItem -Path $ReleaseDir | Select-Object Name, Length
