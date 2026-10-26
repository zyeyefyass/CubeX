#!/usr/bin/env bash
# ═════════════════════════════════════════════════════════════════════════════
# CubeX Release Packaging Script (Bundles Dual Cores + Geodata + Rules + Config)
# Target Version: Test v2.5
# Formats: Native .zip for General Releases + Uncompressed .tar for RouterOS
# ═════════════════════════════════════════════════════════════════════════════
set -e

VERSION="Test-v4.1"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
RELEASE_DIR="$DIST_DIR/release"
INCLUDE_WINDOWS="${INCLUDE_WINDOWS:-0}"

echo "📦 Starting CubeX Test v2.8 Complete Release Packaging (Dual-Core Embedded)..."
if [ "$INCLUDE_WINDOWS" != "1" ]; then
    echo "ℹ️  Windows 平台包发布已按策略暂停 (未收到明确指令前绝不发布 Windows 版本)"
fi

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

if [ "$INCLUDE_WINDOWS" == "1" ]; then
    echo "   -> Compiling Windows amd64 (Explicitly requested)..."
    GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "$DIST_DIR/cubexd.exe" ./cmd/cubexd
else
    echo "   -> Skipping Windows amd64 build per security policy."
fi

# 4. Helper to assemble general release package (.zip format, NO .tar.gz)
assemble_zip_package() {
    local target_name="$1"
    local gateway_bin="$2"
    local arch_type="$3"
    local stage_dir="$DIST_DIR/staging/$target_name"

    rm -rf "$stage_dir"
    mkdir -p "$stage_dir/bin" "$stage_dir/etc" "$stage_dir/data"

    if [ "$arch_type" == "windows" ]; then
        cp "$gateway_bin" "$stage_dir/cubexd.exe"
        cp "$ROOT_DIR/bin/windows/mihomo.exe" "$stage_dir/bin/mihomo.exe"
        cp "$ROOT_DIR/bin/windows/sing-box.exe" "$stage_dir/bin/sing-box.exe"
    else
        cp "$gateway_bin" "$stage_dir/bin/cubexd"
        cp "$ROOT_DIR/bin/$arch_type/mihomo" "$stage_dir/bin/mihomo"
        cp "$ROOT_DIR/bin/$arch_type/sing-box" "$stage_dir/bin/sing-box"
        chmod +x "$stage_dir/bin/"*
    fi

    # Bundle Geodata and 217heidai Dual-Format Rules (.mrs for Mihomo, .srs for sing-box)
    cp -r "$ROOT_DIR/etc/"* "$stage_dir/etc/"
    cp -r "$ROOT_DIR/data/"* "$stage_dir/data/"

    # Strict Desensitization: strip runtime caches, logs, databases, and sanitize configs
    python3 "$ROOT_DIR/scripts/desensitize.py" "$stage_dir"

    # Bundle Deployment & Documentation
    cp "$ROOT_DIR/meta.json" "$stage_dir/"
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

if [ "$INCLUDE_WINDOWS" == "1" ]; then
    echo "📦 [*] Packaging Windows amd64 Release (.zip with dual cores)..."
    assemble_zip_package "CubeX-$VERSION-windows-amd64" "$DIST_DIR/cubexd.exe" "windows"
fi

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

    cp "$ROOT_DIR/meta.json" "$stage_dir/app/meta.json"
    cp -r "$ROOT_DIR/etc/"* "$stage_dir/app/etc/"
    cp -r "$ROOT_DIR/data/"* "$stage_dir/app/data/"

    # Strict Desensitization: strip runtime caches, logs, databases, and sanitize configs
    python3 "$ROOT_DIR/scripts/desensitize.py" "$stage_dir"

    # Uncompressed .tar for RouterOS /container/add
    tar -cf "$RELEASE_DIR/$target_name.tar" -C "$stage_dir" app
    rm -rf "$stage_dir"
}

echo "📦 [5/5] Packaging RouterOS v7 Dedicated Container Images (Uncompressed .tar)..."
assemble_ros_tar "CubeX-$VERSION-ros-amd64" "$DIST_DIR/cubexd-linux-amd64" "amd64"
assemble_ros_tar "CubeX-$VERSION-ros-arm64" "$DIST_DIR/cubexd-linux-arm64" "arm64"

# Copy standalone cubexd gateway binaries for 1-click in-place hot-upgrade
cp "$DIST_DIR/cubexd-linux-amd64" "$RELEASE_DIR/cubexd-linux-amd64"
cp "$DIST_DIR/cubexd-linux-arm64" "$RELEASE_DIR/cubexd-linux-arm64"
if [ -f "$DIST_DIR/cubex-linux-amd64" ]; then
    cp "$DIST_DIR/cubex-linux-amd64" "$RELEASE_DIR/cubex-linux-amd64"
fi
if [ -f "$DIST_DIR/cubex-linux-arm64" ]; then
    cp "$DIST_DIR/cubex-linux-arm64" "$RELEASE_DIR/cubex-linux-arm64"
fi

# Produce latest alias packages for 1-click CI/CD and direct download without tag lookup
cp "$RELEASE_DIR/CubeX-$VERSION-linux-amd64.zip" "$RELEASE_DIR/CubeX-latest-linux-amd64.zip"
cp "$RELEASE_DIR/CubeX-$VERSION-linux-arm64.zip" "$RELEASE_DIR/CubeX-latest-linux-arm64.zip"
cp "$RELEASE_DIR/CubeX-$VERSION-ros-amd64.tar" "$RELEASE_DIR/CubeX-latest-ros-amd64.tar"
cp "$RELEASE_DIR/CubeX-$VERSION-ros-arm64.tar" "$RELEASE_DIR/CubeX-latest-ros-arm64.tar"

# Protect & Encrypt deployment scripts (Zero Source Code Exposure)
python3 "$ROOT_DIR/scripts/protect_script.py"

rm -rf "$DIST_DIR/staging"
echo "🎉 All Dual-Core Release Packages Generated Successfully in: $RELEASE_DIR"
ls -lh "$RELEASE_DIR"

# 6. Strict Automated Post-Packaging Desensitization & Unpack Audit
echo "🛡️  Executing Mandatory Post-Packaging Desensitization Audit (.tar & .zip independent unpack)..."
python3 "$ROOT_DIR/scripts/audit_release_artifacts.py" "$RELEASE_DIR"
