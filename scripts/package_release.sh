#!/usr/bin/env bash
# ═════════════════════════════════════════════════════════════════════════════
# CubeX Release Packaging Script (Bundles Dual Cores + Geodata + Rules + Clean Defaults)
# Formats: Native .zip for General Releases + Clean-Room OCI/Docker Images
# ═════════════════════════════════════════════════════════════════════════════
set -e

VERSION="v9.992"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
RELEASE_DIR="$DIST_DIR/release"
SKIP_RETENTION="${SKIP_RETENTION:-0}"

echo "📦 Starting CubeX $VERSION Complete Release Packaging (Dual-Core Embedded)..."

# 0. Mandatory Pre-Release Full-Stack Health Audit
echo "🩺 [0/4] Executing Mandatory Full-Stack Health Audit Doctor..."
python3 "$ROOT_DIR/scripts/audit_codebase_health.py"

# 1. Clean and prepare output directories
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

# 2. Build Web Frontend
echo "🔨 [1/4] Building Web Frontend..."
cd "$ROOT_DIR/web"
npm run build
python3 "$ROOT_DIR/scripts/precompress_dist.py"

# 3. Cross-compile Pure Linux Multi-Arch Binaries
echo "🔨 [2/4] Cross-compiling Multi-Arch Binaries..."
cd "$ROOT_DIR"
python3 "$ROOT_DIR/scripts/compile_linux.py"

# 4. Helper to assemble general release package (.zip format)
assemble_zip_package() {
    local target_name="$1"
    local gateway_bin="$2"
    local arch_type="$3"
    local stage_dir="$DIST_DIR/staging/$target_name"

    rm -rf "$stage_dir"
    mkdir -p "$stage_dir/bin" "$stage_dir/etc/rules" "$stage_dir/data"

    cp "$gateway_bin" "$stage_dir/bin/cubexd"
    if [ -f "$DIST_DIR/cubex-linux-$arch_type" ]; then
        cp "$DIST_DIR/cubex-linux-$arch_type" "$stage_dir/bin/cubex"
    fi
    cp "$ROOT_DIR/bin/$arch_type/mihomo" "$stage_dir/bin/mihomo"
    cp "$ROOT_DIR/bin/$arch_type/sing-box" "$stage_dir/bin/sing-box"
    chmod +x "$stage_dir/bin/"*

    # Bundle clean-room default IR, geodata, and dual-format rules
    cp -r "$ROOT_DIR/etc/rules/"* "$stage_dir/etc/rules/"
    for gf in GeoIP.dat geosite.dat; do
        [ -f "$ROOT_DIR/etc/$gf" ] && cp "$ROOT_DIR/etc/$gf" "$stage_dir/etc/$gf"
    done
    cp -r "$ROOT_DIR/packaging/defaults/data/"* "$stage_dir/data/"

    # Strict Desensitization safety check
    python3 "$ROOT_DIR/scripts/desensitize.py" "$stage_dir"

    cp "$ROOT_DIR/meta.json" "$stage_dir/"
    cp "$ROOT_DIR/install.sh" "$stage_dir/"
    cp "$ROOT_DIR/docker-compose.yml" "$stage_dir/"
    cp "$ROOT_DIR/README.md" "$stage_dir/"
    cp "$ROOT_DIR/LICENSE" "$stage_dir/"

    (cd "$DIST_DIR/staging" && zip -rq "$RELEASE_DIR/$target_name.zip" "$target_name")
    rm -rf "$stage_dir"
}

echo "📦 [3/4] Packaging Linux amd64 & arm64 (.zip)..."
assemble_zip_package "CubeX-$VERSION-linux-amd64" "$DIST_DIR/cubexd-linux-amd64" "amd64"
assemble_zip_package "CubeX-$VERSION-linux-arm64" "$DIST_DIR/cubexd-linux-arm64" "arm64"

echo "📦 [4/4] Packaging Unified Clean-Room OCI/Docker Images & fnOS .fpk..."
python3 "$ROOT_DIR/scripts/build_oci.py" --arch all --format all
python3 "$ROOT_DIR/scripts/build_fnos_fpk.py"
[ -d "$ROOT_DIR/dist-fpk" ] && cp "$ROOT_DIR/dist-fpk/"*.fpk "$RELEASE_DIR/"

# Copy standalone binaries
cp "$DIST_DIR/cubexd-linux-amd64" "$RELEASE_DIR/cubexd-linux-amd64"
cp "$DIST_DIR/cubexd-linux-arm64" "$RELEASE_DIR/cubexd-linux-arm64"
[ -f "$DIST_DIR/cubex-linux-amd64" ] && cp "$DIST_DIR/cubex-linux-amd64" "$RELEASE_DIR/cubex-linux-amd64"
[ -f "$DIST_DIR/cubex-linux-arm64" ] && cp "$DIST_DIR/cubex-linux-arm64" "$RELEASE_DIR/cubex-linux-arm64"

# Protect deployment scripts
python3 "$ROOT_DIR/scripts/protect_script.py"
rm -rf "$DIST_DIR/staging"

# Generate SHA256SUMS
(cd "$RELEASE_DIR" && sha256sum * | grep -v "SHA256SUMS" > SHA256SUMS)

echo "🛡️  Executing Mandatory Post-Packaging Desensitization & Size-Gate Audit..."
python3 "$ROOT_DIR/scripts/audit_release_artifacts.py" "$RELEASE_DIR" --expected-version "$VERSION" --size-gate

if [ "$SKIP_RETENTION" != "1" ]; then
    python3 "$ROOT_DIR/scripts/enforce_10_version_retention.py"
fi
