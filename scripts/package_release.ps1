# ═════════════════════════════════════════════════════════════════════════════
# CubeX Release Packaging Script (Bundles Binaries + Geodata + Rules + Config)
# Target Version: Test v0.9 (Next: Test v0.99 -> Test v0.999 -> v1.0)
# ═════════════════════════════════════════════════════════════════════════════
$ErrorActionPreference = "Stop"
$Version = "Test-v0.9"
$RootDir = Resolve-Path "$PSScriptRoot\.."
$DistDir = "$RootDir\dist"
$ReleaseDir = "$DistDir\release"

Write-Host "📦 Starting CubeX Release Packaging for version: $Version..." -ForegroundColor Cyan

# 1. Ensure Release Output Directory
if (Test-Path $ReleaseDir) {
    Remove-Item -Path $ReleaseDir -Recurse -Force
}
New-Item -ItemType Directory -Path $ReleaseDir -Force | Out-Null

# 2. Build Web Frontend & Binaries
Write-Host "🔨 [1/4] Building Web Frontend & Multi-Arch Binaries..." -ForegroundColor Yellow
Set-Location -Path "$RootDir\web"
cmd /c npm run build

Set-Location -Path $RootDir
$env:CGO_ENABLED = "0"

# Windows amd64
Write-Host "   -> Compiling Windows amd64..." -ForegroundColor Green
go build -ldflags="-s -w" -o "$DistDir\cubexd.exe" ./cmd/cubexd

# Linux amd64
Write-Host "   -> Cross-compiling Linux amd64..." -ForegroundColor Green
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o "$DistDir\cubexd-linux-amd64" ./cmd/cubexd

# Linux arm64
Write-Host "   -> Cross-compiling Linux arm64..." -ForegroundColor Green
$env:GOARCH = "arm64"
go build -ldflags="-s -w" -o "$DistDir\cubexd-linux-arm64" ./cmd/cubexd

# 3. Helper to assemble bundle staging directory
function Assemble-Bundle($TargetName, $BinaryPath, $BinaryName) {
    $StageDir = "$DistDir\staging\$TargetName"
    if (Test-Path $StageDir) { Remove-Item -Path $StageDir -Recurse -Force }
    New-Item -ItemType Directory -Path $StageDir -Force | Out-Null
    New-Item -ItemType Directory -Path "$StageDir\bin" -Force | Out-Null
    New-Item -ItemType Directory -Path "$StageDir\etc" -Force | Out-Null
    New-Item -ItemType Directory -Path "$StageDir\data" -Force | Out-Null

    # Copy binary
    Copy-Item -Path $BinaryPath -Destination "$StageDir\bin\$BinaryName" -Force

    # Copy data packages (GeoIP, GeoSite, 217heidai MRS rules)
    Write-Host "   -> Packing Geodata & 217heidai rules for $TargetName..." -ForegroundColor Gray
    Copy-Item -Path "$RootDir\etc\*" -Destination "$StageDir\etc" -Recurse -Force
    Copy-Item -Path "$RootDir\data\*" -Destination "$StageDir\data" -Recurse -Force

    # Copy scripts & configs
    Copy-Item -Path "$RootDir\install.sh" -Destination "$StageDir\install.sh" -Force
    Copy-Item -Path "$RootDir\docker-compose.yml" -Destination "$StageDir\docker-compose.yml" -Force
    Copy-Item -Path "$RootDir\README.md" -Destination "$StageDir\README.md" -Force
    Copy-Item -Path "$RootDir\LICENSE" -Destination "$StageDir\LICENSE" -Force

    return $StageDir
}

# 4. Packaging
Write-Host "📦 [2/4] Packaging Linux amd64 Release Bundle..." -ForegroundColor Yellow
$stageAmd64 = Assemble-Bundle "CubeX-$Version-linux-amd64" "$DistDir\cubexd-linux-amd64" "cubexd"
tar -czf "$ReleaseDir\CubeX-$Version-linux-amd64.tar.gz" -C "$DistDir\staging" "CubeX-$Version-linux-amd64"

Write-Host "📦 [3/4] Packaging Linux arm64 Release Bundle..." -ForegroundColor Yellow
$stageArm64 = Assemble-Bundle "CubeX-$Version-linux-arm64" "$DistDir\cubexd-linux-arm64" "cubexd"
tar -czf "$ReleaseDir\CubeX-$Version-linux-arm64.tar.gz" -C "$DistDir\staging" "CubeX-$Version-linux-arm64"

Write-Host "📦 [4/4] Packaging Windows amd64 Release Bundle..." -ForegroundColor Yellow
$stageWin = Assemble-Bundle "CubeX-$Version-windows-amd64" "$DistDir\cubexd.exe" "cubexd.exe"
Compress-Archive -Path "$stageWin\*" -DestinationPath "$ReleaseDir\CubeX-$Version-windows-amd64.zip" -Force

# Clean up staging
Remove-Item -Path "$DistDir\staging" -Recurse -Force

Write-Host "🎉 All Release Bundles with Geodata & Rules Successfully Generated in:" -ForegroundColor Magenta
Write-Host "   $ReleaseDir" -ForegroundColor White
Get-ChildItem -Path $ReleaseDir | Select-Object Name, Length
