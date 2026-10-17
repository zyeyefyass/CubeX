#!/usr/bin/env bash
# ═════════════════════════════════════════════════════════════════════════════
# CubeX Release Packaging Script (Bundles Dual Cores + Geodata + Rules + Config)
# Target Version: Test v0.999 (Next: v1.0)
# Formats: Native .zip for General Releases + Uncompressed .tar for RouterOS
# ═════════════════════════════════════════════════════════════════════════════
set -e

VERSION="Test-v0.999"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
RELEASE_DIR="$DIST_DIR/release"

echo "📦 Starting CubeX Test v0.999 Complete Release Packaging (Dual-Core Embedded)..."

# 1. Clean and prepare output directories
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

# 2. Build Web Frontend
echo "🔨 [1/5] Building Web Frontend..."
cd "$ROOT_DIR/web"
npm run build

# 3. Build Multi-Arch Go Binaries
echo "🔨 [2/5] Cross-compiling Multi-Arch Binaries..."
cd "$ROOT_DIR"
export CGO_ENABLED=0

echo "   -> Compiling Linux amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$DIST_DIR/cubexd-linux-amd64" ./cmd/cubexd

echo "   -> Compiling Linux arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$DIST_DIR/cubexd-linux-arm64" ./cmd/cubexd

# 4. Helper to assemble general release package (.zip format, NO .tar.gz)
assemble_zip_package() {
    local target_name="$1"
    local gateway_bin="$2"
    local arch_type="$3"
    local stage_dir="$DIST_DIR/staging/$target_name"

    rm -rf "$stage_dir"
    mkdir -p "$stage_dir/bin" "$stage_dir/etc" "$stage_dir/data"

    cp "$gateway_bin" "$stage_dir/bin/cubexd"
    cp "$ROOT_DIR/bin/$arch_type/mihomo" "$stage_dir/bin/mihomo"
    cp "$ROOT_DIR/bin/$arch_type/sing-box" "$stage_dir/bin/sing-box"
    chmod +x "$stage_dir/bin/"*

    # Bundle Geodata and 217heidai Dual-Format Rules (.mrs for Mihomo, .srs for sing-box)
    cp -r "$ROOT_DIR/etc/"* "$stage_dir/etc/"
    cp -r "$ROOT_DIR/data/"* "$stage_dir/data/"

    # Bundle Deployment & Documentation
    cp "$ROOT_DIR/install.sh" "$stage_dir/"
    cp "$ROOT_DIR/docker-compose.yml" "$stage_dir/"
    cp "$ROOT_DIR/README.md" "$stage_dir/"
    cp "$ROOT_DIR/LICENSE" "$stage_dir/"

    (cd "$DIST_DIR/staging" && zip -rq "$RELEASE_DIR/$target_name.zip" "$target_name")
    rm -rf "$stage_dir"
}

echo "📦 [3/5] Packaging Linux amd64 Release (.zip with dual cores)..."
assemble_zip_package "CubeX-$VERSION-linux-amd64" "$DIST_DIR/cubexd-linux-amd64" "amd64"

echo "📦 [4/5] Packaging Linux arm64 Release (.zip with dual cores)..."
assemble_zip_package "CubeX-$VERSION-linux-arm64" "$DIST_DIR/cubexd-linux-arm64" "arm64"

# 5. Helper to assemble RouterOS uncompressed .tar container image
assemble_ros_tar() {
    local target_name="$1"
    local gateway_bin="$2"
    local arch_type="$3"
    local stage_dir="$DIST_DIR/staging/$target_name"

    rm -rf "$stage_dir"
    mkdir -p "$stage_dir/app/bin" "$stage_dir/app/etc" "$stage_dir/app/data"

    cp "$gateway_bin" "$stage_dir/app/bin/cubexd"
    cp "$ROOT_DIR/bin/$arch_type/mihomo" "$stage_dir/app/bin/mihomo"
    cp "$ROOT_DIR/bin/$arch_type/sing-box" "$stage_dir/app/bin/sing-box"
    chmod +x "$stage_dir/app/bin/"*

    cp -r "$ROOT_DIR/etc/"* "$stage_dir/app/etc/"
    cp -r "$ROOT_DIR/data/"* "$stage_dir/app/data/"

    # Uncompressed .tar for RouterOS /container/add
    tar -cf "$RELEASE_DIR/$target_name.tar" -C "$stage_dir" app
    rm -rf "$stage_dir"
}

echo "📦 [5/5] Packaging RouterOS v7 Dedicated Container Images (Uncompressed .tar)..."
assemble_ros_tar "CubeX-$VERSION-ros-amd64" "$DIST_DIR/cubexd-linux-amd64" "amd64"
assemble_ros_tar "CubeX-$VERSION-ros-arm64" "$DIST_DIR/cubexd-linux-arm64" "arm64"

# Copy RouterOS WinBox 1-Click RSC Script
cp "$ROOT_DIR/scripts/ros/setup-routeros-v7.rsc" "$RELEASE_DIR/setup-routeros-v7.rsc"

rm -rf "$DIST_DIR/staging"
echo "🎉 All Dual-Core Release Packages Generated Successfully in: $RELEASE_DIR"
ls -lh "$RELEASE_DIR"
