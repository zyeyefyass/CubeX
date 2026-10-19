# -----------------------------------------------------------------------------
# CubeX Monorepo Build Script (Windows PowerShell)
# -----------------------------------------------------------------------------
param(
    [switch]$BuildWindows
)
$ErrorActionPreference = "Stop"

Write-Host "[1/3] Building Cyber Aurora Web Frontend..." -ForegroundColor Cyan
Set-Location -Path "$PSScriptRoot\..\web"
npm.cmd run build

if ($BuildWindows) {
    Write-Host "[2/3] Compiling Windows Host Binaries (cubexd.exe / cubex.exe)..." -ForegroundColor Green
    Set-Location -Path "$PSScriptRoot\.."
    go build -ldflags="-s -w" -o dist/cubexd.exe ./cmd/cubexd
    go build -ldflags="-s -w" -o dist/cubex.exe ./cmd/cubex
} else {
    Write-Host "[2/3] Skipping Windows Binaries (Strict Gating Policy active, use -BuildWindows to enable)" -ForegroundColor DarkGray
    Set-Location -Path "$PSScriptRoot\.."
}

Write-Host "[3/3] Cross-compiling Linux Multi-Arch Binaries (amd64 / arm64)..." -ForegroundColor Yellow
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"

$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o dist/cubexd-linux-amd64 ./cmd/cubexd
go build -ldflags="-s -w" -o dist/cubex-linux-amd64 ./cmd/cubex

$env:GOARCH = "arm64"
go build -ldflags="-s -w" -o dist/cubexd-linux-arm64 ./cmd/cubexd
go build -ldflags="-s -w" -o dist/cubex-linux-arm64 ./cmd/cubex

Write-Host "Success! Binaries generated in CubeX/dist directory." -ForegroundColor Magenta
