#!/usr/bin/env bash
# ═════════════════════════════════════════════════════════════════════════════
# CubeX Release Packaging Script (Bundles Binaries + Geodata + Rules + Config)
# Target Version: Test v0.9 (Next: Test v0.99 -> Test v0.999 -> v1.0)
# ═════════════════════════════════════════════════════════════════════════════
set -e

VERSION="Test-v0.9"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
RELEASE_DIR="$DIST_DIR/release"

echo "📦 Starting CubeX Release Packaging for version: $VERSION..."

# 1. Clean and prepare output directories
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

# 2. Build Web Frontend
echo "🔨 [1/4] Building Web Frontend..."
cd "$ROOT_DIR/web"
npm run build

# 3. Build Multi-Arch Go Binaries
echo "🔨 [2/4] Cross-compiling Multi-Arch Binaries..."
cd "$ROOT_DIR"
export CGO_ENABLED=0

echo "   -> Compiling Linux amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$DIST_DIR/cubexd-linux-amd64" ./cmd/cubexd

echo "   -> Compiling Linux arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$DIST_DIR/cubexd-linux-arm64" ./cmd/cubexd

# 4. Helper Function to Assemble Bundle
assemble_bundle() {
    local target_name="$1"
    local binary_src="$2"
    local binary_name="$3"
    local stage_dir="$DIST_DIR/staging/$target_name"

    rm -rf "$stage_dir"
    mkdir -p "$stage_dir/bin" "$stage_dir/etc" "$stage_dir/data"

    cp "$binary_src" "$stage_dir/bin/$binary_name"
    chmod +x "$stage_dir/bin/$binary_name"

    # Bundle Geodata and 217heidai MRS Rules
    cp -r "$ROOT_DIR/etc/"* "$stage_dir/etc/"
    cp -r "$ROOT_DIR/data/"* "$stage_dir/data/"

    # Bundle Deployment & Documentation
    cp "$ROOT_DIR/install.sh" "$stage_dir/"
    cp "$ROOT_DIR/docker-compose.yml" "$stage_dir/"
    cp "$ROOT_DIR/README.md" "$stage_dir/"
    cp "$ROOT_DIR/LICENSE" "$stage_dir/"

    tar -czf "$RELEASE_DIR/$target_name.tar.gz" -C "$DIST_DIR/staging" "$target_name"
    rm -rf "$stage_dir"
}

echo "📦 [3/4] Packaging Linux amd64 Release..."
assemble_bundle "CubeX-$VERSION-linux-amd64" "$DIST_DIR/cubexd-linux-amd64" "cubexd"

echo "📦 [4/4] Packaging Linux arm64 Release..."
assemble_bundle "CubeX-$VERSION-linux-arm64" "$DIST_DIR/cubexd-linux-arm64" "cubexd"

rm -rf "$DIST_DIR/staging"
echo "🎉 Packaging Complete! Release bundles located in: $RELEASE_DIR"
ls -lh "$RELEASE_DIR"
